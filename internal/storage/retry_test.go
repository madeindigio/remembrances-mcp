package storage

import (
	"context"
	"errors"
	"testing"
)

func TestIsWriteConflictError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("Transaction conflict: Write conflict, retry the transaction"), true},
		{errors.New("write conflict"), true},
		{errors.New("Retry the transaction"), true},
		{errors.New("Found field 'extra', but no such field exists"), false},
		{errors.New("Parse error: Unexpected token"), false},
	}
	for _, c := range cases {
		if got := isWriteConflictError(c.err); got != c.want {
			t.Errorf("isWriteConflictError(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

// TestRetryOnWriteConflictRetriesThenSucceeds covers the case decision 001
// found on the 3.2.4 build: ten concurrent writes to one table produce
// "Transaction conflict", and a bounded retry makes all ten succeed.
func TestRetryOnWriteConflictRetriesThenSucceeds(t *testing.T) {
	attempts := 0
	err := RetryOnWriteConflict(context.Background(), func() error {
		attempts++
		if attempts < 3 {
			return errors.New("Transaction conflict: Write conflict, retry the transaction")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected the retry to succeed, got %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// TestRetryOnWriteConflictDoesNotRetryOtherErrors asserts the retry is narrow:
// a schema error must surface immediately rather than being replayed.
func TestRetryOnWriteConflictDoesNotRetryOtherErrors(t *testing.T) {
	attempts := 0
	want := errors.New("Found field 'extra', but no such field exists for table 'events'")
	err := RetryOnWriteConflict(context.Background(), func() error {
		attempts++
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected the original error, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected a single attempt, got %d", attempts)
	}
}

// TestRetryOnWriteConflictGivesUp asserts a persistent conflict becomes a real
// error rather than an infinite loop or a silently dropped write.
func TestRetryOnWriteConflictGivesUp(t *testing.T) {
	attempts := 0
	err := RetryOnWriteConflict(context.Background(), func() error {
		attempts++
		return errors.New("Transaction conflict: Write conflict, retry the transaction")
	})
	if err == nil {
		t.Fatal("expected a persistent conflict to fail")
	}
	if attempts != writeConflictRetries+1 {
		t.Errorf("expected %d attempts, got %d", writeConflictRetries+1, attempts)
	}
}

// TestRetryOnWriteConflictHonoursContext asserts a cancelled context stops the
// loop instead of sleeping through it.
func TestRetryOnWriteConflictHonoursContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := RetryOnWriteConflict(ctx, func() error {
		return errors.New("Transaction conflict: Write conflict, retry the transaction")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

// TestOutdatedDatastoreIsReportedWithTheMigrationPath asserts the engine's
// low-level refusal to open a 2.x datastore becomes the message the RM-1 design
// requires: an explicit pointer at migrate-v3, not an opaque version code.
func TestOutdatedDatastoreIsReportedWithTheMigrationPath(t *testing.T) {
	raw := errors.New("OutdatedStorageVersion: This storage contains data from a previous SurrealDB version")
	wrapped := wrapOutdatedStorageError(raw)
	if !errors.Is(wrapped, ErrOutdatedDatastore) {
		t.Fatalf("expected ErrOutdatedDatastore, got %v", wrapped)
	}
	if !contains(wrapped.Error(), "migrate-v3") {
		t.Errorf("the error must name the migration command, got: %v", wrapped)
	}

	other := errors.New("connection refused")
	if wrapOutdatedStorageError(other) != other {
		t.Error("an unrelated error must pass through untouched")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		len(needle) == 0 || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
