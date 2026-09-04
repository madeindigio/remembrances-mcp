package storage

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"
)

// ErrOutdatedDatastore is returned when the datastore on disk was written by
// SurrealDB 2.x. The 3.x engine cannot read it and there is no in-place upgrade,
// so the operator has to run the export/import path.
var ErrOutdatedDatastore = errors.New(
	"this datastore was written by SurrealDB 2.x and cannot be opened by the 3.x engine; " +
		"run `remembrances-mcp migrate-v3 --source <db path> --target <db path>` to export it on 2.3 and import it on 3.2")

// writeConflictRetries bounds the retry loop. SurrealDB 3.x surfaces optimistic
// transaction conflicts that 2.x silently swallowed (which meant Remembrances
// was losing failed writes without noticing). A conflicted transaction has
// committed nothing, so replaying the whole statement is safe.
const writeConflictRetries = 5

// isWriteConflictError reports whether an error is an optimistic transaction
// conflict, the one error class that is worth replaying unchanged.
func isWriteConflictError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "transaction conflict") ||
		strings.Contains(msg, "write conflict") ||
		strings.Contains(msg, "retry the transaction")
}

// isOutdatedStorageError reports whether an error is the 3.x engine refusing to
// open a 2.x datastore.
func isOutdatedStorageError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "outdatedstorageversion") ||
		strings.Contains(msg, "outdated storage version") ||
		strings.Contains(msg, "data from a previous surrealdb version")
}

// wrapOutdatedStorageError turns the engine's low-level refusal into the message
// the RM-1 design requires: an explicit pointer at migrate-v3, not a parse error
// or an opaque storage-version code.
func wrapOutdatedStorageError(err error) error {
	if isOutdatedStorageError(err) {
		return fmt.Errorf("%w (underlying error: %v)", ErrOutdatedDatastore, err)
	}
	return err
}

// RetryOnWriteConflict runs fn, replaying it with exponential backoff and jitter
// while it fails with an optimistic write conflict. Any other error, and success,
// return immediately. Exported so callers outside this package (Mem's ACL
// wrapper, the indexer) can wrap their own multi-statement writes.
func RetryOnWriteConflict(ctx context.Context, fn func() error) error {
	var err error
	backoff := 5 * time.Millisecond
	for attempt := 0; attempt <= writeConflictRetries; attempt++ {
		err = fn()
		if !isWriteConflictError(err) {
			return err
		}
		if attempt == writeConflictRetries {
			break
		}
		// Jitter keeps concurrent writers from colliding again in lockstep.
		delay := backoff + time.Duration(rand.Int63n(int64(backoff)+1))
		slog.Debug("Write conflict, retrying", "attempt", attempt+1, "delay", delay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		backoff *= 2
	}
	return fmt.Errorf("write conflict persisted after %d retries: %w", writeConflictRetries, err)
}
