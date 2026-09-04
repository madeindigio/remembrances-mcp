package indexer

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
	"github.com/madeindigio/remembrances-mcp/pkg/principal"
	"github.com/madeindigio/remembrances-mcp/pkg/treesitter"
)

// Background work must reach storage with an identity (RM-2, RM-10).
//
// Two kinds of test guard it. The behavioural ones drive the real entry points
// with require_principal on and a storage that refuses an unscoped call, which
// is exactly what a commercial build with Alchemai Mem's wrapper does. The
// static one refuses the spelling that caused the defect in the first place:
// context.Background() inside a package that does background work throws the
// caller's identity away silently, and no behavioural test can cover every
// future goroutine.

// scopedStorage is a storage that fails closed: every call it implements
// records the principal it was given and refuses a context that has none.
type scopedStorage struct {
	storage.FullStorage

	mu      sync.Mutex
	seen    map[string]*principal.Principal
	project *storage.CodeProject
	files   []storage.CodeFile
}

func newScopedStorage() *scopedStorage {
	return &scopedStorage{seen: make(map[string]*principal.Principal)}
}

// record notes the identity of one call and reports the fail-closed error when
// there is none.
func (s *scopedStorage) record(ctx context.Context, method string) error {
	p, ok := principal.FromContext(ctx)
	if !ok {
		return storage.ErrNoPrincipal
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seen[method] = p
	return nil
}

func (s *scopedStorage) principalFor(method string) *principal.Principal {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.seen[method]
}

func (s *scopedStorage) ListCodeProjects(ctx context.Context) ([]storage.CodeProject, error) {
	if err := s.record(ctx, "ListCodeProjects"); err != nil {
		return nil, err
	}
	if s.project == nil {
		return nil, nil
	}
	return []storage.CodeProject{*s.project}, nil
}

func (s *scopedStorage) GetCodeProject(ctx context.Context, projectID string) (*storage.CodeProject, error) {
	if err := s.record(ctx, "GetCodeProject"); err != nil {
		return nil, err
	}
	return s.project, nil
}

func (s *scopedStorage) UpdateProjectWatcher(ctx context.Context, projectID string, enabled bool) error {
	return s.record(ctx, "UpdateProjectWatcher")
}

func (s *scopedStorage) ListCodeFiles(ctx context.Context, projectID string) ([]storage.CodeFile, error) {
	if err := s.record(ctx, "ListCodeFiles"); err != nil {
		return nil, err
	}
	return s.files, nil
}

func (s *scopedStorage) DeleteCodeFile(ctx context.Context, projectID, filePath string) error {
	return s.record(ctx, "DeleteCodeFile")
}

func (s *scopedStorage) CreateCodeProject(ctx context.Context, project *treesitter.CodeProject) error {
	return s.record(ctx, "CreateCodeProject")
}

func (s *scopedStorage) UpdateProjectStatus(ctx context.Context, projectID string, status treesitter.IndexingStatus) error {
	return s.record(ctx, "UpdateProjectStatus")
}

func (s *scopedStorage) GetCodeFile(ctx context.Context, projectID, filePath string) (*storage.CodeFile, error) {
	if err := s.record(ctx, "GetCodeFile"); err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *scopedStorage) CreateIndexingJob(ctx context.Context, job *treesitter.IndexingJob) (string, error) {
	if err := s.record(ctx, "CreateIndexingJob"); err != nil {
		return "", err
	}
	return job.ID, nil
}

func (s *scopedStorage) GetIndexingJob(ctx context.Context, jobID string) (*storage.CodeIndexingJob, error) {
	if err := s.record(ctx, "GetIndexingJob"); err != nil {
		return nil, err
	}
	return nil, nil
}

// requirePrincipalOn turns the fail-closed switch on for one test, which is
// the state a commercial build runs in.
func requirePrincipalOn(t *testing.T) {
	t.Helper()
	previous := storage.RequirePrincipal()
	storage.SetRequirePrincipal(true)
	t.Cleanup(func() { storage.SetRequirePrincipal(previous) })
}

func testPrincipal() *principal.Principal {
	return &principal.Principal{
		Org:        "org:acme",
		User:       "app_user:alice",
		WriteScope: "mem_scope:alice",
		Scopes:     []string{"mem_scope:alice"},
	}
}

// waitFor polls until cond is true or the deadline passes. The catch-up work
// runs on its own goroutine, so the assertion has to wait for it.
func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

// TestTheOutdatedFileCatchUpKeepsTheActivatingPrincipal is the regression test
// for the defect this work fixes: ActivateProject started the catch-up on
// context.Background(), so every storage call it made ran with no identity.
func TestTheOutdatedFileCatchUpKeepsTheActivatingPrincipal(t *testing.T) {
	requirePrincipalOn(t)

	// The directory is empty on purpose: the only outdated file is one the
	// index still knows about, so the catch-up does exactly one storage call
	// and nothing that needs a parser or an embedder.
	root := t.TempDir()

	st := newScopedStorage()
	st.project = &storage.CodeProject{
		ProjectID:      "proj_1",
		Name:           "proj",
		RootPath:       root,
		IndexingStatus: treesitter.IndexingStatusCompleted,
	}
	// A file that is indexed but no longer on disk: the catch-up deletes it,
	// which touches storage and nothing else.
	st.files = []storage.CodeFile{{ProjectID: "proj_1", FilePath: "gone.go", FileHash: "stale"}}

	wm := NewWatcherManager(NewIndexer(st, nil, DefaultIndexerConfig()), st)
	t.Cleanup(func() { _ = wm.Stop() })

	ctx := principal.NewContext(context.Background(), testPrincipal())
	if _, _, err := wm.ActivateProject(ctx, "proj_1"); err != nil {
		t.Fatalf("ActivateProject: %v", err)
	}

	waitFor(t, "the catch-up to reach storage", func() bool {
		return st.principalFor("DeleteCodeFile") != nil
	})
	if got := st.principalFor("DeleteCodeFile"); got.User != "app_user:alice" {
		t.Fatalf("the catch-up ran as %s, want the activating principal", got)
	}
}

// TestBootTimeActivationRunsAsTheServicePrincipal covers the other half: work
// nobody asked for still has an identity.
func TestBootTimeActivationRunsAsTheServicePrincipal(t *testing.T) {
	requirePrincipalOn(t)

	st := newScopedStorage()
	wm := NewWatcherManager(nil, st)
	t.Cleanup(func() { _ = wm.Stop() })

	if err := wm.AutoActivateOnStartup(context.Background()); err != nil {
		t.Fatalf("AutoActivateOnStartup refused a context with no principal: %v", err)
	}
	p := st.principalFor("ListCodeProjects")
	if p == nil {
		t.Fatal("the boot-time scan never reached storage")
	}
	if !p.IsService {
		t.Errorf("boot-time work ran as %s, want the service principal", p)
	}
}

// TestAnIndexingJobRunsAsItsSubmitter proves the asynchronous job carries the
// identity of the tool call that queued it, long after that call returned.
func TestAnIndexingJobRunsAsItsSubmitter(t *testing.T) {
	requirePrincipalOn(t)

	st := newScopedStorage()
	jm := NewJobManager(st, nil, DefaultIndexerConfig(), JobManagerConfig{MaxConcurrentJobs: 1, QueueSize: 4})
	t.Cleanup(jm.Stop)

	submit, cancel := context.WithCancel(principal.NewContext(context.Background(), testPrincipal()))
	if _, err := jm.SubmitJob(submit, t.TempDir(), "proj"); err != nil {
		t.Fatalf("SubmitJob: %v", err)
	}
	// The submitting request ends immediately, as a tool call does.
	cancel()

	waitFor(t, "the job to reach storage", func() bool {
		return st.principalFor("CreateCodeProject") != nil
	})
	if got := st.principalFor("CreateCodeProject"); got.User != "app_user:alice" {
		t.Fatalf("the job ran as %s, want its submitter", got)
	}
}

// TestAJobSubmittedWithNoIdentityRunsAsTheServicePrincipal keeps the fallback
// honest: it is a service principal, never an absent one.
func TestAJobSubmittedWithNoIdentityRunsAsTheServicePrincipal(t *testing.T) {
	requirePrincipalOn(t)

	st := newScopedStorage()
	jm := NewJobManager(st, nil, DefaultIndexerConfig(), JobManagerConfig{MaxConcurrentJobs: 1, QueueSize: 4})
	t.Cleanup(jm.Stop)

	if _, err := jm.SubmitJob(context.Background(), t.TempDir(), "proj"); err != nil {
		t.Fatalf("SubmitJob: %v", err)
	}
	waitFor(t, "the job to reach storage", func() bool {
		return st.principalFor("CreateCodeProject") != nil
	})
	if got := st.principalFor("CreateCodeProject"); !got.IsService {
		t.Fatalf("an unattributed job ran as %s, want the service principal", got)
	}
}

// TestGetJobStatusReadsStorageAsTheCaller: a job that has fallen out of the
// in-memory map is read back from storage, and that read is scoped like any
// other.
func TestGetJobStatusReadsStorageAsTheCaller(t *testing.T) {
	requirePrincipalOn(t)

	st := newScopedStorage()
	jm := NewJobManager(st, nil, DefaultIndexerConfig(), JobManagerConfig{MaxConcurrentJobs: 1, QueueSize: 1})
	t.Cleanup(jm.Stop)

	ctx := principal.NewContext(context.Background(), testPrincipal())
	if _, err := jm.GetJobStatus(ctx, "job_missing"); err == nil {
		t.Fatal("expected a not-found error for an unknown job")
	}
	got := st.principalFor("GetIndexingJob")
	if got == nil {
		t.Fatal("GetJobStatus did not reach storage")
	}
	if got.User != "app_user:alice" {
		t.Fatalf("GetJobStatus read storage as %s, want the caller", got)
	}
}

// TestNoBackgroundContextIsBuiltByHand refuses the spelling that caused the
// defect. A goroutine that needs to outlive its caller must use
// storage.BackgroundContext (cancellation dropped, identity kept) rather than
// context.Background() (cancellation dropped, identity lost).
func TestNoBackgroundContextIsBuiltByHand(t *testing.T) {
	// Packages that do background work on behalf of a caller. cmd/ is not
	// listed: a command's own root context is legitimately background, and it
	// wraps it in storage.ServiceContext.
	dirs := []string{".", "../kb"}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			path := filepath.Join(dir, name)
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			// A bare context.Background() is only acceptable as the argument
			// of one of the wrappers that immediately installs an identity in
			// it, which is how a root context is legitimately built.
			scoped := map[token.Pos]bool{}
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkg, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}

				switch {
				case pkg.Name == "storage" && (sel.Sel.Name == "ServiceContext" || sel.Sel.Name == "BackgroundContext"),
					pkg.Name == "principal" && sel.Sel.Name == "NewContext":
					for _, arg := range call.Args {
						scoped[arg.Pos()] = true
					}
					return true
				}

				if pkg.Name != "context" {
					return true
				}
				if scoped[call.Pos()] {
					return true
				}
				if sel.Sel.Name == "Background" || sel.Sel.Name == "TODO" {
					t.Errorf("%s builds context.%s() by hand: a background context assembled here has no principal and every storage call it makes fails closed. Use storage.BackgroundContext(ctx) to outlive a request, or storage.ServiceContext(ctx) for work nobody asked for.",
						path, sel.Sel.Name)
				}
				return true
			})
		}
	}
}
