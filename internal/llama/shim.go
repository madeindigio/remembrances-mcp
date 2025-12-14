package llama

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

// PoolingType selects how llama.cpp pools token embeddings into a single vector.
//
// NOTE: This is an internal option type. Its values are NOT the llama.cpp enum
// numeric values; they are mapped before calling into the C shim.
// The zero value is treated as "default".
type PoolingType int32

const (
	PoolingDefault PoolingType = iota
	PoolingNone
	PoolingMean
	PoolingCLS
	PoolingLast
	PoolingRank
)

// AttentionType selects whether the model runs causal or non-causal attention.
//
// NOTE: This is an internal option type. Its values are NOT the llama.cpp enum
// numeric values; they are mapped before calling into the C shim.
// The zero value is treated as "default".
type AttentionType int32

const (
	AttentionDefault AttentionType = iota
	AttentionCausal
	AttentionNonCausal
)

type Options struct {
	Threads      int
	ThreadsBatch int
	GPULayers    int
	ContextSize  uint32
	BatchSize    uint32
	UBatchSize   uint32

	// Behavior knobs
	UseMMap   bool
	UseMLock  bool
	Pooling   PoolingType
	Attention AttentionType
	Normalize int32 // 0 = none, 2 = L2
}

func (o *Options) withDefaults() Options {
	out := *o
	if out.Threads <= 0 {
		out.Threads = 8
	}
	if out.ThreadsBatch <= 0 {
		out.ThreadsBatch = out.Threads
	}
	if out.ContextSize == 0 {
		out.ContextSize = 2048
	}
	if out.BatchSize == 0 {
		out.BatchSize = 2048
	}
	if out.UBatchSize == 0 {
		out.UBatchSize = 2048
	}
	if out.Pooling == PoolingDefault {
		// Prefer a single vector per sequence. Most embedding models work well with MEAN.
		out.Pooling = PoolingMean
	}
	if out.Attention == AttentionDefault {
		// Embedding models typically use non-causal attention.
		out.Attention = AttentionNonCausal
	}
	if out.Normalize == 0 {
		out.Normalize = 2
	}
	return out
}

func llamaPoolingValue(p PoolingType) int32 {
	switch p {
	case PoolingNone:
		return 0
	case PoolingMean:
		return 1
	case PoolingCLS:
		return 2
	case PoolingLast:
		return 3
	case PoolingRank:
		return 4
	default:
		// -1 tells the shim to keep llama.cpp defaults.
		return -1
	}
}

func llamaAttentionValue(a AttentionType) int32 {
	switch a {
	case AttentionCausal:
		return 0
	case AttentionNonCausal:
		return 1
	default:
		// -1 tells the shim to keep llama.cpp defaults.
		return -1
	}
}

type api struct {
	backendInit func()
	backendFree func()

	modelLoad func(path string, nGPULayers int32, useMMap bool, useMLock bool) unsafe.Pointer
	modelFree func(model unsafe.Pointer)

	ctxInit func(model unsafe.Pointer, nCtx uint32, nBatch uint32, nUBatch uint32, nThreads int32, nThreadsBatch int32, poolingType int32, attentionType int32, embeddings bool) unsafe.Pointer
	ctxFree func(ctx unsafe.Pointer)

	modelNEmb func(model unsafe.Pointer) int32

	embedText func(ctx unsafe.Pointer, model unsafe.Pointer, text string, addSpecial bool, parseSpecial bool, out []float32, outLen int32, nThreads int32, nThreadsBatch int32, normalize int32) int32
}

var (
	apiOnce sync.Once
	apiInst *api
	apiErr  error
)

func ensureAPI(ctx context.Context) (*api, error) {
	apiOnce.Do(func() {
		if runtime.GOARCH == "386" {
			apiErr = errors.New("llama shim is not supported on 32-bit architectures")
			return
		}

		libs, err := ensureLibrariesLoaded(ctx, "")
		if err != nil {
			apiErr = err
			return
		}

		var a api
		purego.RegisterLibFunc(&a.backendInit, libs.llamaShim, "rm_llama_backend_init")
		purego.RegisterLibFunc(&a.backendFree, libs.llamaShim, "rm_llama_backend_free")
		purego.RegisterLibFunc(&a.modelLoad, libs.llamaShim, "rm_llama_model_load_from_file")
		purego.RegisterLibFunc(&a.modelFree, libs.llamaShim, "rm_llama_model_free")
		purego.RegisterLibFunc(&a.ctxInit, libs.llamaShim, "rm_llama_context_init")
		purego.RegisterLibFunc(&a.ctxFree, libs.llamaShim, "rm_llama_free")
		purego.RegisterLibFunc(&a.modelNEmb, libs.llamaShim, "rm_llama_model_n_embd")
		purego.RegisterLibFunc(&a.embedText, libs.llamaShim, "rm_llama_embed_text")

		// Must be called once per process.
		a.backendInit()

		apiInst = &a
	})
	return apiInst, apiErr
}

type Model struct {
	a    *api
	model unsafe.Pointer
	ctx   unsafe.Pointer
	normalize int32

	mu sync.Mutex
}

func LoadModel(ctx context.Context, modelPath string, opts Options) (*Model, error) {
	if modelPath == "" {
		return nil, fmt.Errorf("model path is required")
	}

	a, err := ensureAPI(ctx)
	if err != nil {
		return nil, err
	}

	opts = opts.withDefaults()

	model := a.modelLoad(modelPath, int32(opts.GPULayers), opts.UseMMap, opts.UseMLock)
	if model == nil {
		return nil, fmt.Errorf("failed to load model: %s", modelPath)
	}

	ctxPtr := a.ctxInit(model, opts.ContextSize, opts.BatchSize, opts.UBatchSize, int32(opts.Threads), int32(opts.ThreadsBatch), llamaPoolingValue(opts.Pooling), llamaAttentionValue(opts.Attention), true)
	if ctxPtr == nil {
		a.modelFree(model)
		return nil, fmt.Errorf("failed to create context")
	}

	return &Model{a: a, model: model, ctx: ctxPtr, normalize: opts.Normalize}, nil
}

func (m *Model) Dimension() int {
	if m == nil || m.model == nil {
		return 0
	}
	return int(m.a.modelNEmb(m.model))
}

func (m *Model) Embed(ctx context.Context, text string, threads int) ([]float32, error) {
	if m == nil || m.ctx == nil || m.model == nil {
		return nil, fmt.Errorf("model is not initialized")
	}
	if text == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	nEmb := m.Dimension()
	if nEmb <= 0 {
		return nil, fmt.Errorf("invalid embedding dimension")
	}

	out := make([]float32, nEmb)

	// llama_context is not thread-safe for concurrent decode calls.
	m.mu.Lock()
	defer m.mu.Unlock()

	nt := threads
	if nt <= 0 {
		nt = 8
	}

	rc := m.a.embedText(m.ctx, m.model, text, true, true, out, int32(len(out)), int32(nt), int32(nt), m.normalize)
	// Ensure the slice backing array isn't collected during the call.
	runtime.KeepAlive(out)

	if rc != 0 {
		return nil, fmt.Errorf("llama embedding failed (code=%d)", rc)
	}

	return out, nil
}

func (m *Model) Close() error {
	if m == nil {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ctx != nil {
		m.a.ctxFree(m.ctx)
		m.ctx = nil
	}
	if m.model != nil {
		m.a.modelFree(m.model)
		m.model = nil
	}
	return nil
}
