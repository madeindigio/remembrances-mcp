#include "llama_shim.h"

#include "llama_min.h"

#include <math.h>
#include <stdlib.h>
#include <string.h>

static void rm_copy_f32(float *dst, const float *src, int32_t n) {
    if (n <= 0) {
        return;
    }
    memcpy(dst, src, (size_t)n * sizeof(float));
}

static void rm_l2_normalize(float *dst, const float *src, int32_t n) {
    if (n <= 0) {
        return;
    }

    double sum = 0.0;
    for (int32_t i = 0; i < n; i++) {
        const double v = (double)src[i];
        sum += v * v;
    }

    const double norm = sqrt(sum);
    if (norm <= 0.0) {
        rm_copy_f32(dst, src, n);
        return;
    }

    const float inv = (float)(1.0 / norm);
    for (int32_t i = 0; i < n; i++) {
        dst[i] = src[i] * inv;
    }
}

void rm_llama_backend_init(void) {
    llama_backend_init();
}

void rm_llama_backend_free(void) {
    llama_backend_free();
}

struct llama_model * rm_llama_model_load_from_file(const char * path_model, int32_t n_gpu_layers, bool use_mmap, bool use_mlock) {
    struct llama_model_params params = llama_model_default_params();
    params.n_gpu_layers = n_gpu_layers;
    params.use_mmap = use_mmap;
    params.use_mlock = use_mlock;

    // Conservative defaults for safety.
    params.vocab_only = false;
    params.check_tensors = false;

    return llama_model_load_from_file(path_model, params);
}

void rm_llama_model_free(struct llama_model * model) {
    if (model == NULL) {
        return;
    }
    llama_model_free(model);
}

struct llama_context * rm_llama_context_init(
    struct llama_model * model,
    uint32_t n_ctx,
    uint32_t n_batch,
    uint32_t n_ubatch,
    int32_t n_threads,
    int32_t n_threads_batch,
    int32_t pooling_type,
    int32_t attention_type,
    bool embeddings) {

    if (model == NULL) {
        return NULL;
    }

    struct llama_context_params params = llama_context_default_params();

    params.n_ctx = n_ctx;
    params.n_batch = n_batch;
    params.n_ubatch = n_ubatch;

    params.n_threads = n_threads;
    params.n_threads_batch = n_threads_batch;

    params.embeddings = embeddings;

    if (pooling_type != LLAMA_POOLING_TYPE_UNSPECIFIED) {
        params.pooling_type = (enum llama_pooling_type) pooling_type;
    }

    if (attention_type != LLAMA_ATTENTION_TYPE_UNSPECIFIED) {
        params.attention_type = (enum llama_attention_type) attention_type;
    }

    return llama_init_from_model(model, params);
}

void rm_llama_free(struct llama_context * ctx) {
    if (ctx == NULL) {
        return;
    }
    llama_free(ctx);
}

int32_t rm_llama_model_n_embd(const struct llama_model * model) {
    if (model == NULL) {
        return 0;
    }
    return llama_model_n_embd(model);
}

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
    int32_t normalize) {

    if (ctx == NULL || model == NULL || text == NULL || out == NULL) {
        return -1;
    }

    const int32_t n_embd = llama_model_n_embd(model);
    if (n_embd <= 0 || out_len < n_embd) {
        return -2;
    }

    // Reset memory state between calls so each embedding is independent.
    llama_memory_t mem = llama_get_memory(ctx);
    if (mem != NULL) {
        llama_memory_clear(mem, true);
    }

    // Apply per-call thread settings.
    if (n_threads > 0 || n_threads_batch > 0) {
        llama_set_n_threads(ctx, n_threads, n_threads_batch);
    }

    const struct llama_vocab * vocab = llama_model_get_vocab(model);
    if (vocab == NULL) {
        return -3;
    }

    const size_t text_len = strlen(text);
    int32_t n_tokens_max = (int32_t) (text_len + 8);
    if (n_tokens_max < 16) {
        n_tokens_max = 16;
    }

    llama_token * tokens = (llama_token *) malloc((size_t)n_tokens_max * sizeof(llama_token));
    if (tokens == NULL) {
        return -4;
    }

    int32_t n_tokens = llama_tokenize(vocab, text, (int32_t) text_len, tokens, n_tokens_max, add_special, parse_special);
    if (n_tokens < 0) {
        const int32_t needed = -n_tokens;
        free(tokens);
        tokens = (llama_token *) malloc((size_t)needed * sizeof(llama_token));
        if (tokens == NULL) {
            return -4;
        }
        n_tokens = llama_tokenize(vocab, text, (int32_t) text_len, tokens, needed, add_special, parse_special);
    }

    if (n_tokens <= 0) {
        free(tokens);
        return -5;
    }

    struct llama_batch batch = llama_batch_get_one(tokens, n_tokens);

    const int32_t rc = llama_decode(ctx, batch);
    free(tokens);

    if (rc != 0) {
        return rc;
    }

    const float * embd = llama_get_embeddings_seq(ctx, 0);
    if (embd == NULL) {
        embd = llama_get_embeddings(ctx);
    }

    if (embd == NULL) {
        return -6;
    }

    if (normalize == 2) {
        rm_l2_normalize(out, embd, n_embd);
    } else {
        rm_copy_f32(out, embd, n_embd);
    }

    return 0;
}
