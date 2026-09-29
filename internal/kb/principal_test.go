package kb

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/madeindigio/remembrances-mcp/v2/internal/storage"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/principal"
)

// The knowledge-base watcher has no user and must still reach storage with an
// identity: with require_principal on, an unscoped write is refused, so a
// watcher without a principal silently stops synchronising the directory
// (RM-2, RM-10).

// scopedStorage refuses any call whose context carries no principal, which is
// what the ACL wrapper of a commercial build does, and records the identity of
// the ones that do.
type scopedStorage struct {
	storage.Storage

	mu    sync.Mutex
	saved *principal.Principal
}

func (s *scopedStorage) GetDocument(ctx context.Context, filePath string) (*storage.Document, error) {
	if _, ok := principal.FromContext(ctx); !ok {
		return nil, storage.ErrNoPrincipal
	}
	return nil, nil
}

func (s *scopedStorage) SaveDocumentChunks(ctx context.Context, filePath string, chunks []string, embeddings [][]float32, metadata map[string]interface{}) error {
	p, ok := principal.FromContext(ctx)
	if !ok {
		return storage.ErrNoPrincipal
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved = p
	return nil
}

func (s *scopedStorage) savedPrincipal() *principal.Principal {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saved
}

// fixedEmbedder returns a constant vector, so the test needs no model.
type fixedEmbedder struct{}

func (fixedEmbedder) EmbedDocuments(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range texts {
		out[i] = []float32{0.1, 0.2, 0.3, 0.4}
	}
	return out, nil
}

func (fixedEmbedder) EmbedQuery(context.Context, string) ([]float32, error) {
	return []float32{0.1, 0.2, 0.3, 0.4}, nil
}

func (fixedEmbedder) Dimension() int { return 4 }

func TestTheKnowledgeBaseWatcherRunsAsTheServicePrincipal(t *testing.T) {
	previous := storage.RequirePrincipal()
	storage.SetRequirePrincipal(true)
	t.Cleanup(func() { storage.SetRequirePrincipal(previous) })

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "note.md"), []byte("# a note\n\nwith content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	st := &scopedStorage{}
	w, err := StartWatcher(context.Background(), dir, st, fixedEmbedder{}, 200, 20)
	if err != nil {
		t.Fatalf("StartWatcher: %v", err)
	}
	t.Cleanup(w.Stop)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && st.savedPrincipal() == nil {
		time.Sleep(5 * time.Millisecond)
	}

	p := st.savedPrincipal()
	if p == nil {
		t.Fatal("the initial scan never wrote a document: the watcher reached storage with no principal and was refused")
	}
	if !p.IsService {
		t.Errorf("the watcher wrote as %s, want the service principal", p)
	}
}
