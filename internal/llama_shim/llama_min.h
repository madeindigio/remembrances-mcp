// Minimal llama.cpp C API declarations for building libllama_shim.
//
// IMPORTANT:
// - This file must match the ABI of the bundled libllama.so.
// - It intentionally includes only the subset needed for embeddings.
//
// Derived from the pinned llama.cpp header used to build the bundled libraries.

#pragma once

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

// --- Forward declarations (opaque types) ---
struct llama_model;
struct llama_context;
struct llama_vocab;
struct llama_memory_i;

// --- Basic typedefs ---
typedef int32_t llama_pos;
typedef int32_t llama_seq_id;
typedef int32_t llama_token;

typedef struct llama_memory_i * llama_memory_t;

// --- ggml interop placeholders (pointer-sized) ---
// The real llama.cpp header defines these in ggml headers.
// We only need correct sizes here.

typedef void * ggml_backend_dev_t;
typedef void * ggml_backend_buffer_type_t;
typedef void * ggml_threadpool_t;

typedef void (*ggml_backend_sched_eval_callback)(void);
typedef bool (*ggml_abort_callback)(void);
typedef void (*ggml_log_callback)(int, const char *, void *);

enum ggml_type {
    GGML_TYPE_F32 = 0,
};

enum ggml_numa_strategy {
    GGML_NUMA_STRATEGY_DISABLED = 0,
};

// --- llama.cpp enums (numeric values must match llama.h) ---
enum llama_rope_scaling_type {
    LLAMA_ROPE_SCALING_TYPE_UNSPECIFIED = -1,
    LLAMA_ROPE_SCALING_TYPE_NONE        = 0,
    LLAMA_ROPE_SCALING_TYPE_LINEAR      = 1,
    LLAMA_ROPE_SCALING_TYPE_YARN        = 2,
    LLAMA_ROPE_SCALING_TYPE_LONGROPE    = 3,
    LLAMA_ROPE_SCALING_TYPE_MAX_VALUE   = LLAMA_ROPE_SCALING_TYPE_LONGROPE,
};

enum llama_pooling_type {
    LLAMA_POOLING_TYPE_UNSPECIFIED = -1,
    LLAMA_POOLING_TYPE_NONE = 0,
    LLAMA_POOLING_TYPE_MEAN = 1,
    LLAMA_POOLING_TYPE_CLS  = 2,
    LLAMA_POOLING_TYPE_LAST = 3,
    LLAMA_POOLING_TYPE_RANK = 4,
};

enum llama_attention_type {
    LLAMA_ATTENTION_TYPE_UNSPECIFIED = -1,
    LLAMA_ATTENTION_TYPE_CAUSAL      = 0,
    LLAMA_ATTENTION_TYPE_NON_CAUSAL  = 1,
};

enum llama_flash_attn_type {
    LLAMA_FLASH_ATTN_TYPE_AUTO     = -1,
    LLAMA_FLASH_ATTN_TYPE_DISABLED = 0,
    LLAMA_FLASH_ATTN_TYPE_ENABLED  = 1,
};

enum llama_split_mode {
    LLAMA_SPLIT_MODE_NONE  = 0,
    LLAMA_SPLIT_MODE_LAYER = 1,
    LLAMA_SPLIT_MODE_ROW   = 2,
};

// --- Structs used by the API (must match layout) ---

typedef bool (*llama_progress_callback)(float progress, void * user_data);

struct llama_model_tensor_buft_override {
    const char * pattern;
    ggml_backend_buffer_type_t buft;
};

struct llama_model_kv_override;

struct llama_model_params {
    // NULL-terminated list of devices to use for offloading (if NULL, all available devices are used)
    ggml_backend_dev_t * devices;

    // NULL-terminated list of buffer types to use for tensors that match a pattern
    const struct llama_model_tensor_buft_override * tensor_buft_overrides;

    int32_t n_gpu_layers;
    enum llama_split_mode split_mode;

    int32_t main_gpu;

    const float * tensor_split;

    llama_progress_callback progress_callback;
    void * progress_callback_user_data;

    const struct llama_model_kv_override * kv_overrides;

    // Keep the booleans together to avoid misalignment during copy-by-value.
    bool vocab_only;
    bool use_mmap;
    bool use_mlock;
    bool check_tensors;
    bool use_extra_bufts;
    bool no_host;
};

struct llama_context_params {
    uint32_t n_ctx;
    uint32_t n_batch;
    uint32_t n_ubatch;
    uint32_t n_seq_max;
    int32_t  n_threads;
    int32_t  n_threads_batch;

    enum llama_rope_scaling_type rope_scaling_type;
    enum llama_pooling_type      pooling_type;
    enum llama_attention_type    attention_type;
    enum llama_flash_attn_type   flash_attn_type;

    float    rope_freq_base;
    float    rope_freq_scale;
    float    yarn_ext_factor;
    float    yarn_attn_factor;
    float    yarn_beta_fast;
    float    yarn_beta_slow;
    uint32_t yarn_orig_ctx;
    float    defrag_thold;

    ggml_backend_sched_eval_callback cb_eval;
    void * cb_eval_user_data;

    enum ggml_type type_k;
    enum ggml_type type_v;

    ggml_abort_callback abort_callback;
    void *              abort_callback_data;

    // Keep the booleans together and at the end of the struct to avoid misalignment during copy-by-value.
    bool embeddings;
    bool offload_kqv;
    bool no_perf;
    bool op_offload;
    bool swa_full;
    bool kv_unified;
};

struct llama_batch {
    int32_t n_tokens;
    llama_token * token;
    float * embd;
    llama_pos * pos;
    int32_t * n_seq_id;
    llama_seq_id ** seq_id;
    int8_t * logits;
};

// --- llama.cpp C API function prototypes (subset) ---

void llama_backend_init(void);
void llama_backend_free(void);

struct llama_model_params   llama_model_default_params(void);
struct llama_context_params llama_context_default_params(void);

struct llama_model * llama_model_load_from_file(const char * path_model, struct llama_model_params params);
void llama_model_free(struct llama_model * model);

struct llama_context * llama_init_from_model(struct llama_model * model, struct llama_context_params params);
void llama_free(struct llama_context * ctx);

int32_t llama_model_n_embd(const struct llama_model * model);
const struct llama_vocab * llama_model_get_vocab(const struct llama_model * model);

int32_t llama_tokenize(const struct llama_vocab * vocab, const char * text, int32_t text_len, llama_token * tokens, int32_t n_tokens_max, bool add_special, bool parse_special);

struct llama_batch llama_batch_get_one(llama_token * tokens, int32_t n_tokens);
int32_t llama_decode(struct llama_context * ctx, struct llama_batch batch);

void llama_set_n_threads(struct llama_context * ctx, int32_t n_threads, int32_t n_threads_batch);

llama_memory_t llama_get_memory(const struct llama_context * ctx);
void llama_memory_clear(llama_memory_t mem, bool data);

float * llama_get_embeddings(struct llama_context * ctx);
float * llama_get_embeddings_seq(struct llama_context * ctx, llama_seq_id seq_id);

#ifdef __cplusplus
}
#endif
