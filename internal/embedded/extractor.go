package embedded

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	// ErrPlatformUnsupported is returned when the current OS/arch does not have
	// embedded libraries available.
	ErrPlatformUnsupported = errors.New("embedded libraries are not available for this platform")
)

// ExtractResult holds information about the extracted libraries for the
// current platform.
// Files maps the filename (e.g., libsurrealdb_embedded_rs.so) to the fully
// qualified path on disk.
type ExtractResult struct {
	Platform  string
	Directory string
	Files     map[string]string
}

// HasEmbeddedLibraries reports whether this binary ships with embedded
// libraries for the current platform.
func HasEmbeddedLibraries() bool {
	return platformSupported
}

// ExtractLibraries writes the embedded shared libraries for the current
// platform to the provided destination directory. When destDir is empty, a
// temporary directory is created.
func ExtractLibraries(ctx context.Context, destDir string) (*ExtractResult, error) {
	if !platformSupported {
		return nil, ErrPlatformUnsupported
	}

	if destDir == "" {
		temp, err := os.MkdirTemp("", "remembrances-libs-*")
		if err != nil {
			return nil, fmt.Errorf("create temp dir: %w", err)
		}
		destDir = temp
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("ensure destination: %w", err)
	}

	entries, err := fs.ReadDir(platformLibs, platformLibDir)
	if err != nil {
		return nil, fmt.Errorf("list embedded libs: %w", err)
	}

	files := make(map[string]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		name := entry.Name()
		source := filepath.Join(platformLibDir, name)
		target := filepath.Join(destDir, name)

		if err := copyEmbeddedFile(platformLibs, source, target); err != nil {
			return nil, fmt.Errorf("extract %s: %w", name, err)
		}

		files[name] = target
	}

	return &ExtractResult{
		Platform:  fmt.Sprintf("%s/%s", platformOS, platformArch),
		Directory: destDir,
		Files:     files,
	}, nil
}

// AppendLibraryPath prepends the extraction directory to the appropriate
// dynamic loader path for the platform (e.g., LD_LIBRARY_PATH on Linux or
// DYLD_LIBRARY_PATH on macOS). It is idempotent.
func AppendLibraryPath(dir string) error {
	if dir == "" {
		return nil
	}

	key := libraryPathEnvKey()
	current := os.Getenv(key)
	parts := []string{}
	if current != "" {
		parts = append(parts, strings.Split(current, string(os.PathListSeparator))...)
	}

	for _, existing := range parts {
		if existing == dir {
			return nil
		}
	}

	updated := dir
	if current != "" {
		updated = dir + string(os.PathListSeparator) + current
	}

	return os.Setenv(key, updated)
}

func libraryPathEnvKey() string {
	if runtime.GOOS == "darwin" {
		return "DYLD_LIBRARY_PATH"
	}
	return "LD_LIBRARY_PATH"
}

func copyEmbeddedFile(fsys fs.FS, src, dest string) error {
	r, err := fsys.Open(src)
	if err != nil {
		return err
	}
	defer r.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}

	tmp := dest + ".tmp"
	w, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}

	if _, err := io.Copy(w, r); err != nil {
		w.Close()
		_ = os.Remove(tmp)
		return err
	}

	if err := w.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}

	return os.Rename(tmp, dest)
}
