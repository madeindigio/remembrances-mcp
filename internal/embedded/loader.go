package embedded

import (
	"errors"
	"fmt"
	"sort"

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

	for _, name := range orderedNames(files, variant) {
		path := files[name]
		handle, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			return fmt.Errorf("dlopen %s: %w", path, err)
		}
		l.handles[name] = handle
	}

	l.variant = variant

	return nil
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
	ext := platformLibExt
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
