package surrealembedded

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/madeindigio/remembrances-mcp/internal/embedded"
)

type loadedLib struct {
	dir    string
	handle uintptr
}

var (
	libOnce sync.Once
	libInst *loadedLib
	libErr  error
)

func ensureLibraryLoaded(ctx context.Context, destDir string) (*loadedLib, error) {
	libOnce.Do(func() {
		libInst, libErr = loadLibrary(ctx, destDir)
	})
	return libInst, libErr
}

func loadLibrary(ctx context.Context, destDir string) (*loadedLib, error) {
	ext := libExt()
	name := "libsurrealdb_embedded_rs" + ext

	// 1) Prefer libraries next to the running binary (distribution mode)
	if execDir, err := executableDir(); err == nil {
		candidate := filepath.Join(execDir, name)
		if _, err := os.Stat(candidate); err == nil {
			if err := embedded.AppendLibraryPath(execDir); err != nil {
				return nil, fmt.Errorf("update library path: %w", err)
			}
			h, err := purego.Dlopen(candidate, purego.RTLD_NOW|purego.RTLD_GLOBAL)
			if err != nil {
				return nil, fmt.Errorf("dlopen %s: %w", candidate, err)
			}
			return &loadedLib{dir: execDir, handle: h}, nil
		}
	}

	// 2) Development mode: try loading from the repository's checked-in libs
	if repoDir, ok := repoLibDir(); ok {
		candidate := filepath.Join(repoDir, name)
		if _, err := os.Stat(candidate); err == nil {
			if err := embedded.AppendLibraryPath(repoDir); err != nil {
				return nil, fmt.Errorf("update library path: %w", err)
			}
			h, err := purego.Dlopen(candidate, purego.RTLD_NOW|purego.RTLD_GLOBAL)
			if err != nil {
				return nil, fmt.Errorf("dlopen %s: %w", candidate, err)
			}
			return &loadedLib{dir: repoDir, handle: h}, nil
		}
	}

	// 3) Fall back to embedded extraction (requires embedded build tags)
	res, err := embedded.ExtractLibraries(ctx, destDir)
	if err != nil {
		return nil, err
	}
	path, ok := res.Files[name]
	if !ok {
		return nil, fmt.Errorf("embedded library %s not found", name)
	}
	if err := embedded.AppendLibraryPath(res.Directory); err != nil {
		return nil, fmt.Errorf("update library path: %w", err)
	}

	h, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		return nil, fmt.Errorf("dlopen %s: %w", path, err)
	}

	return &loadedLib{dir: res.Directory, handle: h}, nil
}

func libExt() string {
	switch runtime.GOOS {
	case "darwin":
		return ".dylib"
	case "linux":
		return ".so"
	default:
		return ".so"
	}
}

func executableDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", err
	}
	return filepath.Dir(execPath), nil
}

func repoLibDir() (string, bool) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", false
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	base := filepath.Join(root, "internal", "embedded", "libs", runtime.GOOS, runtime.GOARCH)

	candidates := []string{"cpu", "cuda", "cuda-portable", "metal"}
	for _, v := range candidates {
		dir := filepath.Join(base, v)
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return dir, true
		}
	}

	return "", false
}
