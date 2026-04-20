package indexer

import (
	"path/filepath"
	"testing"
)

// TestCodeWatcher_isCodeFile_StripsLeadingDot is a regression test for the bug where
// filepath.Ext returns ".go" (with a leading dot) but GetLanguageByExtension expects
// "go" (without the dot).  Before the fix, isCodeFile always returned false and the
// watcher silently dropped every file-change event.
func TestCodeWatcher_isCodeFile_StripsLeadingDot(t *testing.T) {
	w := &CodeWatcher{
		indexer:  &Indexer{config: DefaultIndexerConfig()},
		rootPath: filepath.Join("/", "stub"),
	}

	cases := []struct {
		path string
		want bool
	}{
		{"/project/main.go", true},
		{"/project/app.js", true},
		{"/project/lib.ts", true},
		{"/project/notes.txt", false},
		{"/project/noextension", false},
	}

	for _, tc := range cases {
		got := w.isCodeFile(tc.path)
		if got != tc.want {
			t.Errorf("isCodeFile(%q) = %v; want %v", tc.path, got, tc.want)
		}
	}
}
