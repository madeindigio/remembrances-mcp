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
}

// NewLoader creates a new loader instance.
func NewLoader() *Loader {
	return &Loader{handles: make(map[string]uintptr)}
}

// Load opens all provided libraries using dlopen with RTLD_NOW|RTLD_GLOBAL so
// dependent libraries can resolve symbols.
func (l *Loader) Load(files map[string]string) error {
	if len(files) == 0 {
		return errors.New("no libraries to load")
	}

	for _, name := range orderedNames(files) {
		path := files[name]
		handle, err := purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			return fmt.Errorf("dlopen %s: %w", path, err)
		}
		l.handles[name] = handle
	}

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

func orderedNames(files map[string]string) []string {
	preferred := []string{
		"libggml-base.so",
		"libggml.so",
		"libggml-cpu.so",
		"libmtmd.so",
		"libllama.so",
		"libsurrealdb_embedded_rs.so",
	}

	seen := make(map[string]struct{}, len(files))
	order := make([]string, 0, len(files))

	for _, name := range preferred {
		if _, ok := files[name]; ok {
			order = append(order, name)
			seen[name] = struct{}{}
		}
	}

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
