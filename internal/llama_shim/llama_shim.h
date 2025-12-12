// Public header for libllama_shim.
//
// This shared library exists to avoid struct-by-value calls from Go via purego
// (struct arguments are not supported on Linux/Windows in purego).
//
// The shim exposes a small pointer-only ABI that wraps the llama.cpp C API.

#pragma once

#include <stdbool.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

struct llama_model;
struct llama_context;

void rm_llama_backend_init(void);
void rm_llama_backend_free(void);

struct llama_model * rm_llama_model_load_from_file(const char * path_model, int32_t n_gpu_layers, bool use_mmap, bool use_mlock);
void rm_llama_model_free(struct llama_model * model);

struct llama_context * rm_llama_context_init(
    struct llama_model * model,
    uint32_t n_ctx,
    uint32_t n_batch,
    uint32_t n_ubatch,
    int32_t n_threads,
    int32_t n_threads_batch,
    int32_t pooling_type,
    int32_t attention_type,
    bool embeddings);

void rm_llama_free(struct llama_context * ctx);

int32_t rm_llama_model_n_embd(const struct llama_model * model);

// Generates a normalized embedding for the provided UTF-8 text.
//
// normalize:
//   0 = no normalization (raw output)
//   2 = L2 normalize (matches llama.cpp examples common_embd_normalize(..., 2))
//
// Returns 0 on success; non-zero on failure.
int32_t rm_llama_embed_text(
    struct llama_context * ctx,
    const struct llama_model * model,
    const char * text,
    bool add_special,
    bool parse_special,
    float * out,
    int32_t out_len,
    int32_t n_threads,
    int32_t n_threads_batch,
    int32_t normalize);

#ifdef __cplusplus
}
#endif
