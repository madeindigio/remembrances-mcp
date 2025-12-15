package surrealembedded

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// Error definitions (mirrors the upstream CGO wrapper semantics).
var (
	ErrNullPtr       = errors.New("null pointer error")
	ErrInvalidHandle = errors.New("invalid database handle")
	ErrInitFailed    = errors.New("database initialization failed")
	ErrQueryFailed   = errors.New("query execution failed")
	ErrUseFailed     = errors.New("use namespace/database failed")
)

type api struct {
	init        func(url string) int32
	initMem     func() int32
	initRocksDB func(path string) int32
	use         func(handle int32, ns string, db string) int32

	query            func(handle int32, query string) unsafe.Pointer
	queryWithParams  func(handle int32, query string, params string) unsafe.Pointer
	create           func(handle int32, resource string, data string) unsafe.Pointer
	update           func(handle int32, resource string, data string) unsafe.Pointer
	deleteResource   func(handle int32, resource string) unsafe.Pointer
	freeString       func(p unsafe.Pointer)
	closeHandle      func(handle int32) int32
}

var (
	apiOnce sync.Once
	apiInst *api
	apiErr  error
)

func ensureAPI(ctx context.Context) (*api, error) {
	apiOnce.Do(func() {
		lib, err := ensureLibraryLoaded(ctx, "")
		if err != nil {
			apiErr = err
			return
		}

		var a api
		purego.RegisterLibFunc(&a.init, lib.handle, "surreal_init")
		purego.RegisterLibFunc(&a.initMem, lib.handle, "surreal_init_mem")
		purego.RegisterLibFunc(&a.initRocksDB, lib.handle, "surreal_init_rocksdb")
		purego.RegisterLibFunc(&a.use, lib.handle, "surreal_use")
		purego.RegisterLibFunc(&a.query, lib.handle, "surreal_query")
		purego.RegisterLibFunc(&a.queryWithParams, lib.handle, "surreal_query_with_params")
		purego.RegisterLibFunc(&a.create, lib.handle, "surreal_create")
		purego.RegisterLibFunc(&a.update, lib.handle, "surreal_update")
		purego.RegisterLibFunc(&a.deleteResource, lib.handle, "surreal_delete")
		purego.RegisterLibFunc(&a.freeString, lib.handle, "surreal_free_string")
		purego.RegisterLibFunc(&a.closeHandle, lib.handle, "surreal_close")

		apiInst = &a
	})
	return apiInst, apiErr
}

// DB represents an embedded SurrealDB instance (purego, runtime-loaded).
//
// This type intentionally matches the subset of methods used by the project.
type DB struct {
	a      *api
	handle int32
}

type backendKind int

const (
	backendMemory backendKind = iota
	backendRocksDB
)

func parseEmbeddedURL(url string) (backendKind, string, error) {
	normalized := strings.TrimSpace(url)
	switch {
	case normalized == "memory" || normalized == "memory://":
		return backendMemory, "", nil
	case strings.HasPrefix(normalized, "rocksdb://"):
		return backendRocksDB, expandUser(strings.TrimPrefix(normalized, "rocksdb://")), nil
	case strings.HasPrefix(normalized, "file://"):
		return backendRocksDB, expandUser(strings.TrimPrefix(normalized, "file://")), nil
	case strings.HasPrefix(normalized, "surrealkv://"):
		// SurrealKV is a supported embedded backend in the CLI/config. The current
		// purego wrapper exposes a file-backed initializer; keep backward
		// compatibility by accepting the scheme.
		return backendRocksDB, expandUser(strings.TrimPrefix(normalized, "surrealkv://")), nil
	default:
		// Treat a plain path as rocksdb for backward compatibility with existing config.
		if strings.Contains(normalized, "://") {
			return 0, "", fmt.Errorf("unsupported embedded SurrealDB URL: %s", url)
		}
		return backendRocksDB, expandUser(normalized), nil
	}
}

// NewFromURL creates a new embedded SurrealDB instance from a URL-like string.
//
// Supported forms:
//   - "memory" or "memory://" for in-memory
//   - "rocksdb://<path>" for RocksDB backend
//   - "file://<path>" (deprecated alias for rocksdb)
//   - "<path>" (no scheme) treated as RocksDB path for compatibility
func NewFromURL(ctx context.Context, url string) (*DB, error) {
	a, err := ensureAPI(ctx)
	if err != nil {
		return nil, err
	}

	normalized := strings.TrimSpace(url)
	switch {
	case normalized == "" :
		return nil, fmt.Errorf("embedded SurrealDB URL/path is empty")
	case normalized == "memory" || normalized == "memory://":
		h := a.initMem()
		if h <= 0 {
			if h < 0 {
				return nil, handleError(int(h))
			}
			return nil, fmt.Errorf("database initialization returned invalid handle: %d", h)
		}
		return &DB{a: a, handle: h}, nil
	case strings.Contains(normalized, "://"):
		// Let the embedded library parse backend URLs such as:
		// - surrealkv:///path
		// - rocksdb:///path
		// - file:///path
		// This keeps behavior aligned with the Rust embedded implementation.
		h := a.init(normalized)
		if h <= 0 {
			if h < 0 {
				return nil, handleError(int(h))
			}
			return nil, fmt.Errorf("database initialization returned invalid handle: %d", h)
		}
		return &DB{a: a, handle: h}, nil
	default:
		// Backward compatibility: a plain path is treated as RocksDB.
		path := expandUser(normalized)
		h := a.initRocksDB(path)
		if h <= 0 {
			if h < 0 {
				return nil, handleError(int(h))
			}
			return nil, fmt.Errorf("database initialization returned invalid handle: %d", h)
		}
		return &DB{a: a, handle: h}, nil
	}
}

func expandUser(path string) string {
	if path == "" {
		return path
	}
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			if path == "~" {
				return home
			}
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	return path
}

// Use selects a namespace and database to use.
func (db *DB) Use(namespace, database string) error {
	if db == nil || db.a == nil {
		return fmt.Errorf("database is not initialized")
	}
	result := db.a.use(db.handle, namespace, database)
	if result != 0 {
		return handleError(int(result))
	}
	return nil
}

// Query executes a SurrealQL query and returns the decoded JSON response.
func (db *DB) Query(query string, vars map[string]interface{}) ([]interface{}, error) {
	if db == nil || db.a == nil {
		return nil, fmt.Errorf("database is not initialized")
	}

	var resPtr unsafe.Pointer
	if vars != nil && len(vars) > 0 {
		varsJSON, err := json.Marshal(vars)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal query variables: %w", err)
		}
		resPtr = db.a.queryWithParams(db.handle, query, string(varsJSON))
	} else {
		resPtr = db.a.query(db.handle, query)
	}

	if resPtr == nil {
		return nil, ErrQueryFailed
	}
	defer db.a.freeString(resPtr)

	jsonStr := cStringToGo(resPtr)

	// Check for error in response
	var errorResp map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &errorResp); err == nil {
		if errMsg, ok := errorResp["error"]; ok {
			return nil, fmt.Errorf("query error: %v", errMsg)
		}
	}

	// Handle null result (e.g., from DEFINE statements)
	if jsonStr == "null" {
		return []interface{}{}, nil
	}

	var data []interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return data, nil
}

// Create creates a new record in the database.
func (db *DB) Create(resource string, data interface{}) (interface{}, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	resPtr := db.a.create(db.handle, resource, string(payload))
	if resPtr == nil {
		return nil, ErrQueryFailed
	}
	defer db.a.freeString(resPtr)

	parsed, err := parseResult(cStringToGo(resPtr))
	if err != nil {
		return nil, err
	}

	// Create always returns a single record; unwrap from array when applicable.
	if arr, ok := parsed.([]interface{}); ok && len(arr) > 0 {
		return arr[0], nil
	}
	return parsed, nil
}

// Update replaces a record in the database.
func (db *DB) Update(resource string, data interface{}) (interface{}, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	resPtr := db.a.update(db.handle, resource, string(payload))
	if resPtr == nil {
		return nil, ErrQueryFailed
	}
	defer db.a.freeString(resPtr)

	return parseResult(cStringToGo(resPtr))
}

// Delete removes records from the database.
func (db *DB) Delete(resource string) (interface{}, error) {
	resPtr := db.a.deleteResource(db.handle, resource)
	if resPtr == nil {
		return nil, ErrQueryFailed
	}
	defer db.a.freeString(resPtr)

	return parseResult(cStringToGo(resPtr))
}

// Close closes the database connection.
func (db *DB) Close() error {
	if db == nil || db.a == nil {
		return nil
	}
	result := db.a.closeHandle(db.handle)
	if result != 0 {
		return handleError(int(result))
	}
	return nil
}

func handleError(code int) error {
	switch code {
	case -1:
		return ErrNullPtr
	case -2:
		return ErrInvalidHandle
	case -3:
		return ErrInitFailed
	case -4:
		return ErrQueryFailed
	case -5:
		return ErrUseFailed
	default:
		return fmt.Errorf("unknown error code: %d", code)
	}
}

func parseResult(jsonStr string) (interface{}, error) {
	var errorResp map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &errorResp); err == nil {
		if errMsg, ok := errorResp["error"]; ok {
			return nil, fmt.Errorf("database error: %v", errMsg)
		}
	}

	var result interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	return result, nil
}

func cStringToGo(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}

	// Safety guard: cap at 64MiB to avoid scanning unbounded memory if something goes wrong.
	const max = 64 << 20
	b := make([]byte, 0, 1024)
	for i := 0; i < max; i++ {
		c := *(*byte)(unsafe.Add(p, uintptr(i)))
		if c == 0 {
			break
		}
		b = append(b, c)
	}
	return string(b)
}
