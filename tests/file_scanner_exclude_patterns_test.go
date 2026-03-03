package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/madeindigio/remembrances-mcp/internal/config"
	"github.com/madeindigio/remembrances-mcp/internal/indexer"
)

func TestFileScanner_ExclusionPatterns_DefaultsAndConfiguredValues(t *testing.T) {
	tempDir := t.TempDir()

	mustWriteFile(t, filepath.Join(tempDir, "main.go"), "package main\nfunc main() {}\n")
	mustWriteFile(t, filepath.Join(tempDir, "auto.generated.go"), "package main\nfunc generated() {}\n")

	// Directories excluded by built-in defaults.
	mustWriteFile(t, filepath.Join(tempDir, "vendor", "ignored.go"), "package vendorpkg\nfunc IgnoredVendor() {}\n")
	mustWriteFile(t, filepath.Join(tempDir, "node_modules", "ignored.js"), "function ignoredNodeModules() {}\n")
	mustWriteFile(t, filepath.Join(tempDir, ".cache", "ignored.py"), "def ignored_cache():\n    pass\n")

	// Directories excluded only when user-configured patterns are merged.
	mustWriteFile(t, filepath.Join(tempDir, "var", "ignored.go"), "package varpkg\nfunc Ignored() {}\n")
	mustWriteFile(t, filepath.Join(tempDir, "custom_out", "ignored.go"), "package custompkg\nfunc IgnoredCustom() {}\n")

	defaultScanner := indexer.NewFileScanner()
	defaultScanResult, err := defaultScanner.Scan(tempDir)
	if err != nil {
		t.Fatalf("default scan failed: %v", err)
	}

	assertNotIndexed(t, defaultScanResult.Files, filepath.Join("vendor", "ignored.go"), "expected built-in default pattern to exclude vendor")
	assertNotIndexed(t, defaultScanResult.Files, filepath.Join("node_modules", "ignored.js"), "expected built-in default pattern to exclude node_modules")
	assertNotIndexed(t, defaultScanResult.Files, filepath.Join(".cache", "ignored.py"), "expected hidden/cache directories to be excluded by default")
	assertIndexed(t, defaultScanResult.Files, filepath.Join("var", "ignored.go"), "expected var to be indexed before user-configured exclude patterns are merged")
	assertIndexed(t, defaultScanResult.Files, filepath.Join("custom_out", "ignored.go"), "expected custom_out to be indexed before user-configured exclude patterns are merged")

	cfgYAML := "gguf-model-path: \"/tmp/fake-model.gguf\"\n" +
		"db-path: \"/tmp/remembrances-test.db\"\n" +
		"code-indexing-exclude-patterns:\n" +
		"  - var\n" +
		"  - custom_out\n" +
		"  - \"*.generated.go\"\n"

	cfgPath := filepath.Join(tempDir, "config.yaml")
	mustWriteFile(t, cfgPath, cfgYAML)

	originalArgs := os.Args
	t.Cleanup(func() {
		os.Args = originalArgs
	})
	os.Args = []string{"remembrances-mcp", "--config", cfgPath}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	patterns := cfg.GetCodeIndexingExcludePatterns()
	if len(patterns) != 3 {
		t.Fatalf("expected 3 configured exclude patterns, got %d (%v)", len(patterns), patterns)
	}

	scanner := indexer.NewFileScanner()
	if len(patterns) > 0 {
		scanner.MergeExcludePatterns(patterns)
	}

	scanResult, err := scanner.Scan(tempDir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	// Built-in defaults still apply after merging user patterns.
	assertNotIndexed(t, scanResult.Files, filepath.Join("vendor", "ignored.go"), "expected built-in default pattern to remain active after merging user patterns")
	assertNotIndexed(t, scanResult.Files, filepath.Join("node_modules", "ignored.js"), "expected built-in default pattern to remain active after merging user patterns")

	// User-configured exclusions (including multiple values) are enforced.
	assertNotIndexed(t, scanResult.Files, filepath.Join("var", "ignored.go"), "expected configured pattern var to exclude directory")
	assertNotIndexed(t, scanResult.Files, filepath.Join("custom_out", "ignored.go"), "expected configured pattern custom_out to exclude directory")
	assertNotIndexed(t, scanResult.Files, "auto.generated.go", "expected configured wildcard pattern *.generated.go to exclude generated file")

	// Non-excluded file is still indexed.
	assertIndexed(t, scanResult.Files, "main.go", "expected main.go to remain indexed")
}

func TestFileScanner_ExclusionPatterns_CommaSeparatedConfiguredValues(t *testing.T) {
	testCases := []struct {
		name     string
		raw      string
		expected []string
	}{
		{
			name:     "comma_with_spaces",
			raw:      "one, two, three",
			expected: []string{"one", "two", "three"},
		},
		{
			name:     "comma_without_spaces",
			raw:      "one,two,three",
			expected: []string{"one", "two", "three"},
		},
		{
			name:     "mixed_spacing",
			raw:      "one , two, three ,four",
			expected: []string{"one", "two", "three", "four"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()

			mustWriteFile(t, filepath.Join(tempDir, "main.go"), "package main\nfunc main() {}\n")

			// Directories excluded by built-in defaults.
			mustWriteFile(t, filepath.Join(tempDir, "vendor", "ignored.go"), "package vendorpkg\nfunc IgnoredVendor() {}\n")
			mustWriteFile(t, filepath.Join(tempDir, "node_modules", "ignored.js"), "function ignoredNodeModules() {}\n")

			// Directories expected to be excluded by configured comma-separated values.
			mustWriteFile(t, filepath.Join(tempDir, "one", "ignored.go"), "package onepkg\nfunc IgnoredOne() {}\n")
			mustWriteFile(t, filepath.Join(tempDir, "two", "ignored.go"), "package twopkg\nfunc IgnoredTwo() {}\n")
			mustWriteFile(t, filepath.Join(tempDir, "three", "ignored.go"), "package threepkg\nfunc IgnoredThree() {}\n")
			mustWriteFile(t, filepath.Join(tempDir, "four", "ignored.go"), "package fourpkg\nfunc IgnoredFour() {}\n")

			defaultScanner := indexer.NewFileScanner()
			defaultScanResult, err := defaultScanner.Scan(tempDir)
			if err != nil {
				t.Fatalf("default scan failed: %v", err)
			}

			assertNotIndexed(t, defaultScanResult.Files, filepath.Join("vendor", "ignored.go"), "expected built-in default pattern to exclude vendor")
			assertNotIndexed(t, defaultScanResult.Files, filepath.Join("node_modules", "ignored.js"), "expected built-in default pattern to exclude node_modules")
			assertIndexed(t, defaultScanResult.Files, filepath.Join("one", "ignored.go"), "expected one to be indexed before user-configured exclude patterns are merged")
			assertIndexed(t, defaultScanResult.Files, filepath.Join("two", "ignored.go"), "expected two to be indexed before user-configured exclude patterns are merged")
			assertIndexed(t, defaultScanResult.Files, filepath.Join("three", "ignored.go"), "expected three to be indexed before user-configured exclude patterns are merged")
			assertIndexed(t, defaultScanResult.Files, filepath.Join("four", "ignored.go"), "expected four to be indexed before user-configured exclude patterns are merged")

			cfg := &config.Config{CodeIndexingExcludePatterns: tc.raw}
			patterns := cfg.GetCodeIndexingExcludePatterns()
			if len(patterns) != len(tc.expected) {
				t.Fatalf("expected %d configured exclude patterns, got %d (%v)", len(tc.expected), len(patterns), patterns)
			}
			for i, expected := range tc.expected {
				if patterns[i] != expected {
					t.Fatalf("unexpected parsed pattern at index %d: got %q want %q (raw: %q)", i, patterns[i], expected, tc.raw)
				}
			}

			scanner := indexer.NewFileScanner()
			scanner.MergeExcludePatterns(patterns)

			scanResult, err := scanner.Scan(tempDir)
			if err != nil {
				t.Fatalf("scan failed: %v", err)
			}

			// Built-in defaults still apply after merging user patterns.
			assertNotIndexed(t, scanResult.Files, filepath.Join("vendor", "ignored.go"), "expected built-in default pattern to remain active after merging user patterns")
			assertNotIndexed(t, scanResult.Files, filepath.Join("node_modules", "ignored.js"), "expected built-in default pattern to remain active after merging user patterns")

			// User-configured comma-separated exclusions are enforced.
			for _, dir := range tc.expected {
				assertNotIndexed(t, scanResult.Files, filepath.Join(dir, "ignored.go"), "expected configured pattern to exclude directory")
			}

			if !containsString(tc.expected, "four") {
				assertIndexed(t, scanResult.Files, filepath.Join("four", "ignored.go"), "expected four to remain indexed when not part of configured patterns")
			}

			assertIndexed(t, scanResult.Files, "main.go", "expected main.go to remain indexed")
		})
	}
}

func TestFileScanner_ExclusionPatterns_PathPatternFolderPHP(t *testing.T) {
	tempDir := t.TempDir()

	mustWriteFile(t, filepath.Join(tempDir, "folder", "file.php"), "<?php echo 'skip';")
	mustWriteFile(t, filepath.Join(tempDir, "folder", "file.js"), "function keep() {}")

	scanner := indexer.NewFileScanner()
	scanner.MergeExcludePatterns([]string{"folder/*.php"})

	scanResult, err := scanner.Scan(tempDir)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}

	assertNotIndexed(t, scanResult.Files, filepath.Join("folder", "file.php"), "expected folder/*.php pattern to exclude PHP file")
	assertIndexed(t, scanResult.Files, filepath.Join("folder", "file.js"), "expected non-matching file in same folder to remain indexed")
}

func containsString(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func containsRelPath(files []indexer.ScannedFile, relPath string) bool {
	relPath = filepath.Clean(relPath)
	for _, file := range files {
		if filepath.Clean(file.RelPath) == relPath {
			return true
		}
	}
	return false
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("failed to create parent directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func assertIndexed(t *testing.T, files []indexer.ScannedFile, relPath, msg string) {
	t.Helper()
	if !containsRelPath(files, relPath) {
		t.Fatalf("%s: %q not found in indexed files", msg, relPath)
	}
}

func assertNotIndexed(t *testing.T, files []indexer.ScannedFile, relPath, msg string) {
	t.Helper()
	if containsRelPath(files, relPath) {
		t.Fatalf("%s: %q was unexpectedly indexed", msg, relPath)
	}
}
