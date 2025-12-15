package embedded

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ebitengine/purego"
)

// Loader handles dlopen/dlclose using purego for the extracted libraries.
type Loader struct {
	handles map[string]uintptr
	variant string
}

// NewLoader creates a new loader instance.
func NewLoader() *Loader {
	return &Loader{handles: make(map[string]uintptr)}
}

// Load opens all provided libraries using dlopen with RTLD_NOW|RTLD_GLOBAL so
// dependent libraries can resolve symbols.
func (l *Loader) Load(files map[string]string, variant string) error {
	if len(files) == 0 {
		return errors.New("no libraries to load")
	}

	ext := libraryFileExt()
	flags := purego.RTLD_NOW | purego.RTLD_GLOBAL

	for _, name := range orderedNames(files, variant) {
		if _, ok := l.handles[name]; ok {
			continue
		}

		path := files[name]
		handle, err := purego.Dlopen(path, flags)
		if err != nil {
			// Some builds encode optional backends as DT_NEEDED entries on libggml.so.
			// When this happens, loading libggml.so directly may fail unless the
			// backend shared object can be resolved by the dynamic loader.
			//
			// To keep the embedded extraction approach robust across variants, detect
			// the missing backend from the dlopen error, load it explicitly from the
			// same directory, and retry.
			if filepath.IsAbs(path) {
				if dep := missingKnownDependency(err, ext); dep != "" {
					if _, already := l.handles[dep]; !already {
						depPath := files[dep]
						if depPath == "" {
							depPath = filepath.Join(filepath.Dir(path), dep)
						}
						if _, stErr := os.Stat(depPath); stErr == nil {
							depHandle, depErr := purego.Dlopen(depPath, flags)
							if depErr != nil {
								return fmt.Errorf("dlopen %s: %w", depPath, depErr)
							}
							l.handles[dep] = depHandle
						} else {
							return fmt.Errorf(
								"dlopen %s: %w (missing dependency %s next to extracted libraries; ensure you built/embedded the full library set for your variant)",
								path,
								err,
								dep,
							)
						}
					}

					// Retry the original library now that the dependency is loaded.
					handle, err = purego.Dlopen(path, flags)
				}
			}
			if err != nil {
			return fmt.Errorf("dlopen %s: %w", path, err)
			}
		}
		l.handles[name] = handle
	}

	l.variant = variant

	return nil
}

func missingKnownDependency(err error, ext string) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	for _, dep := range []string{"libggml-cuda" + ext, "libggml-metal" + ext, "libggml-cpu" + ext} {
		if strings.Contains(msg, dep) {
			return dep
		}
	}
	return ""
}

// Close releases all loaded libraries. Errors are aggregated if multiple
// dlclose calls fail.
func (l *Loader) Close() error {
	var errs []error
	for name, handle := range l.handles {
		if err := purego.Dlclose(handle); err != nil {
			errs = append(errs, fmt.Errorf("dlclose %s: %w", name, err))
		}
		delete(l.handles, name)
	}

	return errors.Join(errs...)
}

// Variant reports the loaded variant name (e.g., cpu, cuda, cuda-portable, metal).
func (l *Loader) Variant() string {
	return l.variant
}

func orderedNames(files map[string]string, variant string) []string {
	ext := libraryFileExt()
	seen := make(map[string]struct{}, len(files))
	order := make([]string, 0, len(files))

	add := func(name string) bool {
		if _, ok := files[name]; ok {
			order = append(order, name)
			seen[name] = struct{}{}
			return true
		}
		return false
	}

	add("libggml-base" + ext)
	add("libggml" + ext)

	switch variant {
	case "cuda", "cuda-portable":
		if !add("libggml-cuda" + ext) {
			add("libggml-cpu" + ext)
		}
	case "metal":
		if !add("libggml-metal" + ext) {
			add("libggml-cpu" + ext)
		}
	default:
		add("libggml-cpu" + ext)
	}

	add("libmtmd" + ext)
	add("libllama" + ext)
	add("libsurrealdb_embedded_rs" + ext)

	extra := make([]string, 0, len(files)-len(order))
	for name := range files {
		if _, ok := seen[name]; !ok {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	order = append(order, extra...)
	return order
}
