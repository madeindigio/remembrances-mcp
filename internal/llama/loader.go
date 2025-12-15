package llama

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/madeindigio/remembrances-mcp/internal/embedded"
)

type loadedLibs struct {
	dir        string
	variant    string
	llamaShim  uintptr
	openHandles []uintptr
}

var (
	libsOnce sync.Once
	libs     *loadedLibs
	libsErr  error
)

func ensureLibrariesLoaded(ctx context.Context, destDir string) (*loadedLibs, error) {
	libsOnce.Do(func() {
		libs, libsErr = loadLibraries(ctx, destDir)
	})
	return libs, libsErr
}

func loadLibraries(ctx context.Context, destDir string) (*loadedLibs, error) {
	ext := libExt()
	// Required (ggml backend is selected dynamically).
	required := []string{
		"libggml-base" + ext,
		"libggml" + ext,
		"libmtmd" + ext,
		"libllama" + ext,
		"libllama_shim" + ext,
	}

	// 1) Prefer libraries next to the running binary (distribution mode)
	execDir, err := executableDir()
	if err == nil {
		if ok, files := findLibsInDir(execDir, ext, required); ok {
			if err := embedded.AppendLibraryPath(execDir); err != nil {
				return nil, fmt.Errorf("update library path: %w", err)
			}
			return dlopenInOrder(files, ext)
		}
	}

	// 2) Development mode: try loading from the repository's checked-in libs
	// (works even when the binary was built without embedded build tags).
	if repoDir, ok := repoLibDir(); ok {
		if ok, files := findLibsInDir(repoDir, ext, required); ok {
			if err := embedded.AppendLibraryPath(repoDir); err != nil {
				return nil, fmt.Errorf("update library path: %w", err)
			}
			return dlopenInOrder(files, ext)
		}
	}

	// 3) Fall back to embedded extraction (requires embedded build tags)
	res, err := embedded.ExtractLibraries(ctx, destDir)
	if err != nil {
		return nil, err
	}
	if err := embedded.AppendLibraryPath(res.Directory); err != nil {
		return nil, fmt.Errorf("update library path: %w", err)
	}

	// Keep the full file map from the extracted result.
	return dlopenInOrder(res.Files, ext)
}

func dlopenInOrder(files map[string]string, ext string) (*loadedLibs, error) {
	// Determine which ggml backend to load based on what's present.
	backend := "libggml-cpu" + ext
	if _, ok := files["libggml-cuda"+ext]; ok {
		backend = "libggml-cuda" + ext
	} else if _, ok := files["libggml-metal"+ext]; ok {
		backend = "libggml-metal" + ext
	}

	order := []string{
		"libggml-base" + ext,
		"libggml" + ext,
		backend,
		"libmtmd" + ext,
		"libllama" + ext,
		"libllama_shim" + ext,
	}

	var handles []uintptr
	for _, name := range order {
		path, ok := files[name]
		if !ok {
			// Allow running against system libraries if not embedded.
			path = name
		}

		h, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			// If we are loading from an explicit path next to the binary, we can
			// opportunistically try to load a missing dependency from the same
			// directory and retry. This helps with cases where libggml.so has an
			// explicit DT_NEEDED on libggml-cuda.so (or similar).
			if filepath.IsAbs(path) {
				if dep := missingKnownDependency(err, ext); dep != "" {
					depPath := filepath.Join(filepath.Dir(path), dep)
					if _, stErr := os.Stat(depPath); stErr == nil {
						depHandle, depErr := purego.Dlopen(depPath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
						if depErr != nil {
							return nil, fmt.Errorf("dlopen %s: %w", depPath, depErr)
						}
						handles = append(handles, depHandle)

						// Retry the original library now that the dependency is loaded.
						h, err = purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
						if err != nil {
							return nil, fmt.Errorf("dlopen %s: %w", path, err)
						}
					} else {
						return nil, fmt.Errorf(
							"dlopen %s: %w (missing dependency %s next to the binary; ensure you copied the full library set for your variant)",
							path,
							err,
							dep,
						)
					}
				} else {
					return nil, fmt.Errorf("dlopen %s: %w", path, err)
				}
			} else {
				return nil, fmt.Errorf("dlopen %s: %w", path, err)
			}
		}
		handles = append(handles, h)

		if name == "libllama_shim"+ext {
			return &loadedLibs{
				dir:         filepath.Dir(path),
				variant:     backend,
				llamaShim:   h,
				openHandles: handles,
			}, nil
		}
	}

	return nil, fmt.Errorf("libllama_shim%s not loaded", ext)
}

func missingKnownDependency(err error, ext string) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	// Keep this intentionally conservative: only detect the known optional backends.
	for _, dep := range []string{"libggml-cuda" + ext, "libggml-metal" + ext, "libggml-cpu" + ext} {
		// Typical dlopen message: "...: libggml-cuda.so: cannot open shared object file: No such file or directory"
		if strings.Contains(msg, dep) {
			return dep
		}
	}
	return ""
}

func libExt() string {
	switch runtime.GOOS {
	case "darwin":
		return ".dylib"
	case "linux":
		return ".so"
	default:
		// For now we only support the platforms that the project ships libs for.
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
	// Resolve module root based on this file location.
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", false
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	base := filepath.Join(root, "internal", "embedded", "libs", runtime.GOOS, runtime.GOARCH)

	// Prefer CPU libs by default; allow CUDA variants if present.
	candidates := []string{"cpu", "cuda", "cuda-portable", "metal"}
	for _, v := range candidates {
		dir := filepath.Join(base, v)
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return dir, true
		}
	}

	return "", false
}

func findLibsInDir(dir string, ext string, required []string) (bool, map[string]string) {
	files := make(map[string]string, len(required))
	for _, name := range required {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			files[name] = candidate
		}
	}

	// ggml backend is optional as long as cpu or cuda/metal is present; we'll decide later.
	needed := func(name string) bool {
		_, ok := files[name]
		return ok
	}

	if !needed("libggml-base"+ext) || !needed("libggml"+ext) || !needed("libmtmd"+ext) || !needed("libllama"+ext) || !needed("libllama_shim"+ext) {
		return false, nil
	}

	// Probe for a backend lib.
	for _, backend := range []string{"libggml-cuda" + ext, "libggml-metal" + ext, "libggml-cpu" + ext} {
		candidate := filepath.Join(dir, backend)
		if _, err := os.Stat(candidate); err == nil {
			files[backend] = candidate
			break
		}
	}

	return true, files
}
