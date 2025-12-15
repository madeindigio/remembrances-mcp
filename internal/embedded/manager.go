package embedded

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
)

// ExtractAndLoad first attempts to load libraries from the directory where
// the binary is located. If that fails (libraries not found), it falls back
// to extracting the embedded libraries to destDir (or a temp directory when
// empty) and loading them using purego.
//
// This allows distribution of libraries alongside the binary while still
// supporting the embedded approach as a fallback.
func ExtractAndLoad(ctx context.Context, destDir string) (*ExtractResult, *Loader, error) {
	// First, try to load libraries from the binary's directory
	res, ldr, err := tryLoadFromBinaryDir(ctx)
	if err == nil {
		slog.Info("Loaded libraries from binary directory", "dir", res.Directory, "variant", res.Variant)
		return res, ldr, nil
	}

	slog.Debug("Libraries not found next to binary, falling back to embedded extraction", "reason", err.Error())

	// Fall back to extracting embedded libraries
	res, err = ExtractLibraries(ctx, destDir)
	if err != nil {
		return nil, nil, err
	}

	// Ensure the extraction directory is visible to the dynamic loader before
	// attempting to dlopen the libraries, so transitive dependencies located in
	// the same directory can be resolved.
	if err := AppendLibraryPath(res.Directory); err != nil {
		return nil, nil, fmt.Errorf("update library path: %w", err)
	}

	ldr = NewLoader()
	if err := ldr.Load(res.Files, res.Variant); err != nil {
		return nil, nil, fmt.Errorf("load embedded libraries: %w", err)
	}

	slog.Info("Loaded libraries from embedded extraction", "dir", res.Directory, "variant", res.Variant)
	return res, ldr, nil
}

// tryLoadFromBinaryDir attempts to find and load shared libraries from the
// same directory where the executable binary is located.
func tryLoadFromBinaryDir(ctx context.Context) (*ExtractResult, *Loader, error) {
	// Get the path to the current executable
	execPath, err := os.Executable()
	if err != nil {
		return nil, nil, fmt.Errorf("get executable path: %w", err)
	}

	// Resolve symlinks to get the real path
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve executable symlinks: %w", err)
	}

	binDir := filepath.Dir(execPath)

	// Look for library files in the binary's directory
	files, err := findLibrariesInDir(binDir)
	if err != nil {
		return nil, nil, err
	}

	if len(files) == 0 {
		return nil, nil, fmt.Errorf("no libraries found in binary directory: %s", binDir)
	}

	// Ensure the binary directory is part of the loader search path before
	// attempting to dlopen, so dependent libraries in the same directory are
	// discoverable.
	if err := AppendLibraryPath(binDir); err != nil {
		return nil, nil, fmt.Errorf("update library path: %w", err)
	}

	// Try to load the libraries
	ldr := NewLoader()
	variant := inferVariant(files)
	if err := ldr.Load(files, variant); err != nil {
		return nil, nil, fmt.Errorf("load libraries from binary dir: %w", err)
	}

	return &ExtractResult{
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Variant:   variant,
		Portable:  false,
		Directory: binDir,
		Files:     files,
	}, ldr, nil
}

// findLibrariesInDir scans a directory for shared library files matching the
// platform extension (.so on Linux, .dylib on macOS).
func findLibrariesInDir(dir string) (map[string]string, error) {
	ext := libraryFileExt()
	if ext == "" {
		return nil, fmt.Errorf("platform library extension not defined")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dir, err)
	}

	files := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !isLibraryFile(name) {
			continue
		}

		files[name] = filepath.Join(dir, name)
	}

	return files, nil
}

func inferVariant(files map[string]string) string {
	ext := libraryFileExt()
	if _, ok := files["libggml-cuda"+ext]; ok {
		return "cuda"
	}
	if _, ok := files["libggml-metal"+ext]; ok {
		return "metal"
	}
	if _, ok := files["libggml-cpu"+ext]; ok {
		return "cpu"
	}
	return ""
}
