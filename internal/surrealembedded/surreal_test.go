package surrealembedded

import "testing"

func TestParseEmbeddedURL(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		wantKind  backendKind
		wantPath  string
		wantError bool
	}{
		{name: "memory", in: "memory", wantKind: backendMemory},
		{name: "memory scheme", in: "memory://", wantKind: backendMemory},
		{name: "rocksdb", in: "rocksdb:///tmp/db", wantKind: backendRocksDB, wantPath: "/tmp/db"},
		{name: "file alias", in: "file:///tmp/db", wantKind: backendRocksDB, wantPath: "/tmp/db"},
		{name: "surrealkv alias", in: "surrealkv:///tmp/db", wantKind: backendRocksDB, wantPath: "/tmp/db"},
		{name: "plain path", in: "/tmp/db", wantKind: backendRocksDB, wantPath: "/tmp/db"},
		{name: "unsupported scheme", in: "http://example", wantError: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotKind, gotPath, err := parseEmbeddedURL(tc.in)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotKind != tc.wantKind {
				t.Fatalf("kind=%v, want %v", gotKind, tc.wantKind)
			}
			if gotPath != tc.wantPath {
				t.Fatalf("path=%q, want %q", gotPath, tc.wantPath)
			}
		})
	}
}
