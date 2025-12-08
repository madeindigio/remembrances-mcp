package embedded

import (
	"context"
	"fmt"
)

// ExtractAndLoad extracts the embedded libraries for the current platform to
// destDir (or a temp directory when empty) and loads them using purego. It
// returns both the extraction result and the loader so callers can manage the
// lifetime of the loaded handles.
func ExtractAndLoad(ctx context.Context, destDir string) (*ExtractResult, *Loader, error) {
	res, err := ExtractLibraries(ctx, destDir)
	if err != nil {
		return nil, nil, err
	}

	ldr := NewLoader()
	if err := ldr.Load(res.Files, res.Variant); err != nil {
		return nil, nil, fmt.Errorf("load embedded libraries: %w", err)
	}

	if err := AppendLibraryPath(res.Directory); err != nil {
		return nil, nil, fmt.Errorf("update library path: %w", err)
	}

	return res, ldr, nil
}
