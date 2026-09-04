package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// docsDir is where the checked in artefacts live, relative to this package.
const docsDir = "../../docs/api"

// TestTheCheckedInDocumentsMatchTheServer is the test that keeps the published
// API document honest (RM-5).
//
// It regenerates the artefacts from the live route registrations and compares
// them byte for byte with the files under docs/api. Adding, removing or
// renaming a route, changing a method, a description or a schema, all fail
// here until the documents are regenerated:
//
//	go generate ./internal/transport/...
//
// The comparison runs once per build configuration: without tags it checks the
// core documents, with `-tags commercial` the commercial ones, because each
// build serves a different surface.
func TestTheCheckedInDocumentsMatchTheServer(t *testing.T) {
	generated, err := Generate()
	if err != nil {
		t.Fatalf("generating the documents failed: %v", err)
	}
	if len(generated) == 0 {
		t.Fatal("the generator produced nothing")
	}

	for name, want := range generated {
		path := filepath.Join(docsDir, name)
		got, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s is missing; run go generate ./internal/transport/... (%v)", path, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s is stale: it does not match the routes the server registers.\n"+
				"Regenerate it with: go generate ./internal/transport/...\n"+
				"checked in %d bytes, generated %d bytes", path, len(got), len(want))
		}
	}
}

// TestTheGeneratorProducesTheSameBytesTwice guards the comparison above: a
// generator whose output depended on map iteration order would fail the drift
// test at random and teach everyone to ignore it.
func TestTheGeneratorProducesTheSameBytesTwice(t *testing.T) {
	first, err := Generate()
	if err != nil {
		t.Fatalf("first generation failed: %v", err)
	}
	second, err := Generate()
	if err != nil {
		t.Fatalf("second generation failed: %v", err)
	}
	for name, want := range first {
		if !bytes.Equal(second[name], want) {
			t.Errorf("%s is not deterministic", name)
		}
	}
}
