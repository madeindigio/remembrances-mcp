// Package indexer provides code indexing functionality using tree-sitter parsing.
package indexer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// FileScanner discovers and filters source files in a project directory
type FileScanner struct {
	// Patterns to exclude from scanning (e.g., "node_modules", ".git")
	ExcludePatterns []string

	// Maximum file size to index (in bytes)
	MaxFileSize int64

	// Languages to include (empty means all supported)
	IncludeLanguages []treesitter.Language
}

// ScannedFile represents a discovered source file
type ScannedFile struct {
	// Absolute path to the file
	AbsPath string

	// Relative path from project root
	RelPath string

	// Detected language
	Language treesitter.Language

	// File size in bytes
	Size int64

	// SHA-256 hash of file content
	Hash string
}

// ScanResult contains the results of a directory scan
type ScanResult struct {
	// Root path that was scanned
	RootPath string

	// All discovered files
	Files []ScannedFile

	// Files by language
	ByLanguage map[treesitter.Language][]ScannedFile

	// Errors encountered during scanning
	Errors []error

	// Statistics
	TotalFiles    int
	TotalSize     int64
	SkippedFiles  int
	SkippedReason map[string]int
}

// DefaultExcludePatterns returns common patterns to exclude
func DefaultExcludePatterns() []string {
	return []string{
		// Version control
		".git",
		".svn",
		".hg",
		".bzr",
		"_darcs",

		// JavaScript / TypeScript / Node
		"node_modules",
		"bower_components",
		"jspm_packages",
		".pnpm",
		".next",
		".nuxt",
		".npm",
		".yarn",

		// Go
		"vendor",

		// Python
		".venv",
		"venv",
		".env",
		"env",
		"__pycache__",
		".tox",
		".mypy_cache",
		".pytest_cache",
		".ruff_cache",
		"eggs",
		"*.egg-info",
		".eggs",

		// Ruby
		".bundle",

		// PHP
		// vendor already listed under Go

		// Java / Kotlin / Android
		".gradle",
		".m2",

		// .NET / C#
		"obj",
		"packages",
		".nuget",

		// Rust
		"target",

		// Swift / iOS / macOS
		"Pods",
		"DerivedData",
		".build",
		"*.xcworkspace",

		// Dart / Flutter
		".dart_tool",
		".pub-cache",
		".pub",

		// Build outputs (general)
		"dist",
		"build",
		"out",
		"bin",

		// IDE and editors
		".idea",
		".vscode",
		".vs",
		".fleet",
		".eclipse",
		".settings",
		".project",
		".classpath",
		"*.swp",
		"*.swo",
		"*~",

		// Temporary and cache
		".cache",
		".tmp",
		"tmp",
		"temp",
		"coverage",
		".nyc_output",

		// Generated
		"generated",
		"*.generated.*",
		"*.min.js",
		"*.min.css",
		"*.bundle.js",

		// Test fixtures and mocks
		"__mocks__",
		"__fixtures__",
		"testdata",

		// Documentation builds
		"site",
		"docs/_build",
		"_site",

		// Infrastructure / DevOps
		".terraform",
		".vagrant",

		// Lock files (not code)
		"*.lock",
		"package-lock.json",
		"yarn.lock",
		"pnpm-lock.yaml",
		"Cargo.lock",
		"go.sum",
		"Gemfile.lock",
		"composer.lock",
		"Podfile.lock",
		"Packages.resolved",
	}
}

// NewFileScanner creates a new file scanner with default settings
func NewFileScanner() *FileScanner {
	return &FileScanner{
		ExcludePatterns: DefaultExcludePatterns(),
		MaxFileSize:     1024 * 1024, // 1MB default
	}
}

// MergeExcludePatterns adds user-configured patterns to the existing exclude list,
// avoiding duplicates.
func (s *FileScanner) MergeExcludePatterns(patterns []string) {
	existing := make(map[string]bool, len(s.ExcludePatterns))
	for _, p := range s.ExcludePatterns {
		existing[p] = true
	}
	for _, p := range patterns {
		if !existing[p] {
			s.ExcludePatterns = append(s.ExcludePatterns, p)
			existing[p] = true
		}
	}
}

// Scan discovers all source files in the given directory
func (s *FileScanner) Scan(rootPath string) (*ScanResult, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	result := &ScanResult{
		RootPath:      absRoot,
		Files:         make([]ScannedFile, 0),
		ByLanguage:    make(map[treesitter.Language][]ScannedFile),
		Errors:        make([]error, 0),
		SkippedReason: make(map[string]int),
	}

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, err)
			return nil // Continue scanning
		}

		// Get relative path
		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			relPath = path
		}

		// Check if path should be excluded
		if s.shouldExclude(path, relPath, d.IsDir()) {
			result.SkippedReason["excluded"]++
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories (we just walk into them)
		if d.IsDir() {
			return nil
		}

		// Check file extension for supported language
		ext := filepath.Ext(path)
		// Remove leading dot from extension (filepath.Ext returns ".go", we need "go")
		ext = strings.TrimPrefix(ext, ".")
		lang, ok := treesitter.GetLanguageByExtension(ext)
		if !ok {
			result.SkippedFiles++
			result.SkippedReason["unsupported_extension"]++
			return nil
		}

		// Check if language is in include list
		if len(s.IncludeLanguages) > 0 && !s.containsLanguage(lang) {
			result.SkippedFiles++
			result.SkippedReason["language_filtered"]++
			return nil
		}

		// Get file info
		info, err := d.Info()
		if err != nil {
			result.Errors = append(result.Errors, err)
			return nil
		}

		// Check file size
		if info.Size() > s.MaxFileSize {
			result.SkippedFiles++
			result.SkippedReason["too_large"]++
			return nil
		}

		// Calculate file hash
		hash, err := s.calculateHash(path)
		if err != nil {
			result.Errors = append(result.Errors, err)
			return nil
		}

		scannedFile := ScannedFile{
			AbsPath:  path,
			RelPath:  relPath,
			Language: lang,
			Size:     info.Size(),
			Hash:     hash,
		}

		result.Files = append(result.Files, scannedFile)
		result.ByLanguage[lang] = append(result.ByLanguage[lang], scannedFile)
		result.TotalFiles++
		result.TotalSize += info.Size()

		return nil
	})

	return result, err
}

// ShouldExclude checks if a path should be excluded based on patterns.
// It is exported so that CodeWatcher can reuse the same exclusion logic.
func (s *FileScanner) ShouldExclude(absPath, relPath string, isDir bool) bool {
	return s.shouldExclude(absPath, relPath, isDir)
}

// shouldExclude checks if a path should be excluded based on patterns
func (s *FileScanner) shouldExclude(absPath, relPath string, isDir bool) bool {
	// Get the base name
	name := filepath.Base(absPath)

	for _, pattern := range s.ExcludePatterns {
		// Check if pattern starts with * (wildcard)
		if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(name, suffix) {
				return true
			}
			continue
		}

		// Exact match on directory/file name
		if name == pattern {
			return true
		}

		// Check if any path component matches
		parts := strings.Split(relPath, string(filepath.Separator))
		for _, part := range parts {
			if part == pattern {
				return true
			}
		}
	}

	// Hidden files/directories (starting with .)
	if strings.HasPrefix(name, ".") && name != "." && name != ".." {
		// Allow some hidden files
		allowedHidden := map[string]bool{
			".github": true,
			".gitlab": true,
		}
		if !allowedHidden[name] {
			return true
		}
	}

	return false
}

// containsLanguage checks if a language is in the include list
func (s *FileScanner) containsLanguage(lang treesitter.Language) bool {
	for _, l := range s.IncludeLanguages {
		if l == lang {
			return true
		}
	}
	return false
}

// calculateHash computes SHA-256 hash of file content
func (s *FileScanner) calculateHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// FilterByLanguage returns only files of specified languages
func (r *ScanResult) FilterByLanguage(languages ...treesitter.Language) []ScannedFile {
	if len(languages) == 0 {
		return r.Files
	}

	langSet := make(map[treesitter.Language]bool)
	for _, l := range languages {
		langSet[l] = true
	}

	filtered := make([]ScannedFile, 0)
	for _, f := range r.Files {
		if langSet[f.Language] {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// GetLanguageStats returns file counts by language
func (r *ScanResult) GetLanguageStats() map[treesitter.Language]int {
	stats := make(map[treesitter.Language]int)
	for lang, files := range r.ByLanguage {
		stats[lang] = len(files)
	}
	return stats
}
