package embedded

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestExtractLibrariesCreatesFiles(t *testing.T) {
	if !HasEmbeddedLibraries() {
		t.Skip("no embedded libraries available for this platform")
	}

	ctx := context.Background()
	res, err := ExtractLibraries(ctx, "")
	if err != nil {
		t.Fatalf("ExtractLibraries failed: %v", err)
	}
	if len(res.Files) == 0 {
		t.Fatalf("expected at least one embedded library, got %d", len(res.Files))
	}

	t.Cleanup(func() {
		_ = os.RemoveAll(res.Directory)
	})

	for name, path := range res.Files {
		if statErr := assertFileExecutable(path); statErr != nil {
			t.Fatalf("extracted library %s is not readable/executable: %v", name, statErr)
		}
	}
}

func TestAppendLibraryPathIdempotent(t *testing.T) {
	key := libraryPathEnvKey()
	base := "/tmp/example"
	os.Setenv(key, base)
	t.Cleanup(func() { os.Setenv(key, base) })

	if err := AppendLibraryPath("/tmp/newlib"); err != nil {
		t.Fatalf("AppendLibraryPath returned error: %v", err)
	}
	if err := AppendLibraryPath("/tmp/newlib"); err != nil {
		t.Fatalf("AppendLibraryPath second call returned error: %v", err)
	}

	value := os.Getenv(key)
	if strings.Count(value, "/tmp/newlib") != 1 {
		t.Fatalf("expected library path to contain single /tmp/newlib, got %q", value)
	}
}

func TestLoaderHandlesMissingLibrary(t *testing.T) {
	ldr := NewLoader()
	err := ldr.Load(map[string]string{"missing.so": "/path/does/not/exist"}, "cpu")
	if err == nil {
		t.Fatalf("expected error when loading missing library")
	}
}

func TestLoaderCanLoadEmbedded(t *testing.T) {
	if !HasEmbeddedLibraries() {
		t.Skip("no embedded libraries available for this platform")
	}

	ctx := context.Background()
	res, err := ExtractLibraries(ctx, "")
	if err != nil {
		t.Fatalf("ExtractLibraries failed: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(res.Directory) })

	ldr := NewLoader()
	t.Cleanup(func() { _ = ldr.Close() })

	if err := ldr.Load(res.Files, res.Variant); err != nil {
		// Loading may fail on platforms without required runtime dependencies.
		t.Skipf("skipping load test because dlopen failed: %v", err)
	}
}

func TestOrderedNamesPrefersVariant(t *testing.T) {
	ext := platformLibExt
	files := map[string]string{
		"libggml-base" + ext: "base",
		"libggml" + ext:      "ggml",
		"libggml-cuda" + ext: "cuda",
		"libllama" + ext:     "llama",
	}

	order := orderedNames(files, "cuda")
	expected := []string{
		"libggml-base" + ext,
		"libggml" + ext,
		"libggml-cuda" + ext,
		"libllama" + ext,
	}

	if len(order) < len(expected) {
		t.Fatalf("unexpected order length: got %d, want >= %d", len(order), len(expected))
	}

	for i, name := range expected {
		if order[i] != name {
			t.Fatalf("order[%d]=%s, expected %s", i, order[i], name)
		}
	}
}

func assertFileExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode()&0o111 == 0 {
		return os.ErrPermission
	}
	return nil
}
