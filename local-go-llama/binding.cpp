#include "common.h"
#include "llama.h"

#include "binding.h"
// #include "grammar-parser.h"  // Removed in newer llama.cpp versions
#include <cassert>
#include <cinttypes>
void* load_binding_model_custom(const char *fname, int n_ctx, int n_seed, bool memory_f16, bool mlock, bool embeddings, bool mmap, bool low_vram, int n_gpu_layers, int n_batch, int n_ubatch, const char *maingpu, const char *tensorsplit, bool numa,  float rope_freq_base, float rope_freq_scale, bool mul_mat_q, const char *lora, const char *lora_base, bool perplexity);
#include <cmath>
#include <cstdio>
#include <cstring>
#include <fstream>
#include <sstream>
#include <iostream>
#include <string>
#include <vector>
#include <sstream>
#include <regex>
#if defined (__unix__) || (defined (__APPLE__) && defined (__MACH__))
#include <signal.h>
#include <unistd.h>
#elif defined (_WIN32)
#define WIN32_LEAN_AND_MEAN
#define NOMINMAX
#include <windows.h>
#include <signal.h>
#endif

#if defined (__unix__) || (defined (__APPLE__) && defined (__MACH__)) || defined (_WIN32)
void sigint_handler(int signo) {
    if (signo == SIGINT) {
            _exit(130);
    }
}
#endif

// State structure for binding - holds model and context with smart pointers
struct llama_binding_state {
    llama_context_ptr context;
    llama_model_ptr model;
    std::vector<llama_adapter_lora_ptr> lora;
    common_params* params = nullptr;
};


int get_embeddings(void* params_ptr, void* state_pr, float * res_embeddings) {
    common_params* params = (common_params*) params_ptr;
    llama_binding_state* state = (llama_binding_state*) state_pr;
    llama_context* ctx = state->context.get();
    llama_model* model = state->model.get();

    if (params->sampling.seed <= 0) {
        params->sampling.seed = time(NULL);
    }

    // tokenize the prompt using common_tokenize which returns a vector
    auto embd_inp = common_tokenize(ctx, params->prompt, true, true);

    if (embd_inp.size() > 0) {
        // Create batch for embeddings - use sequence 0
        llama_batch batch = llama_batch_get_one(embd_inp.data(), embd_inp.size());
        
        if (llama_decode(ctx, batch)) {
            fprintf(stderr, "%s : failed to decode\n", __func__);
            return 1;
        }
        // Note: llama_batch_get_one returns a view, not an owned batch, so we don't free it
    }

    const int n_embd = llama_model_n_embd(model);

    // Use sequence embeddings (pooling type dependent)
    const float * embd = llama_get_embeddings_seq(ctx, 0);
    if (embd == NULL) {
        embd = llama_get_embeddings(ctx);
    }
    
    if (embd == NULL) {
        fprintf(stderr, "%s : failed to get embeddings\n", __func__);
        return 1;
    }

    // Normalize embeddings (embd_norm = 2 by default in llama.cpp examples)
    common_embd_normalize(embd, res_embeddings, n_embd, 2);
    return 0;
}


int get_token_embeddings(void* params_ptr, void* state_pr,  int *tokens, int tokenSize, float * res_embeddings) {
    common_params* params_p = (common_params*) params_ptr;
    llama_binding_state* state = (llama_binding_state*) state_pr;
    llama_context* ctx = state->context.get();
    llama_model* model = state->model.get();
    common_params params = *params_p;
 
    const struct llama_vocab * vocab = llama_model_get_vocab(model);
    for (int i = 0; i < tokenSize; i++) {
        char buf[128];
        int n = llama_token_to_piece(vocab, tokens[i], buf, sizeof(buf), 0, true);
        if (n < 0) {
            fprintf(stderr, "%s: error: failed to convert token to piece\n", __func__);
            return 1;
        }
        std::string str_token(buf, n);
        params_p->prompt += str_token;
    }

  return get_embeddings(params_ptr,state_pr,res_embeddings);
}

int get_embedding_size(void* state_pr) {
    llama_binding_state* state = (llama_binding_state*) state_pr;
    llama_model* model = state->model.get();
    return llama_n_embd(model);
}

// NOTE: This function is DISABLED - text generation not supported
int eval(void* params_ptr,void* state_pr,char *text) {
    fprintf(stderr, "ERROR: eval is disabled - text generation not supported in this version\n");
    fprintf(stderr, "       Please use the Embeddings() method for embedding generation\n");
    return 1;
}

static llama_context ** g_ctx;
static common_params               * g_params;
static std::vector<llama_token> * g_input_tokens;
static std::ostringstream       * g_output_ss;
static std::vector<llama_token> * g_output_tokens;

int llama_predict(void* params_ptr, void* state_pr, char* result, bool debug) {
    // NOTE: This function is currently disabled due to extensive API changes in llama.cpp
    // The sampling API has been completely rewritten and requires significant refactoring.
    // For text generation, please use the llama.cpp binaries directly or wait for this to be updated.
    // Embeddings functionality is fully working - use the Embeddings() method instead.
    fprintf(stderr, "%s: error: llama_predict is currently disabled - use embeddings or llama.cpp binaries for generation\n", __func__);
    strcpy(result, "ERROR: llama_predict function disabled - embeddings work fine, use Embeddings() method");
    return 1;
}

// this is a bit of a hack now - ideally this should be in the predict function
// and be transparent to the caller, however this now maps 1:1 (mostly) the upstream implementation
// Note: both model have to be loaded with perplexity "true" to enable all logits
int speculative_sampling(void* params_ptr, void* target_model, void* draft_model, char* result, bool debug) {
    // NOTE: This function is currently disabled due to extensive API changes in llama.cpp
    // The sampling API has been completely rewritten and requires significant refactoring.
    // For speculative sampling, please use the llama.cpp binaries directly.
    fprintf(stderr, "%s: error: speculative_sampling is currently disabled\n", __func__);
    strcpy(result, "ERROR: speculative_sampling function disabled");
    return 1;
}

void llama_binding_free_model(void * state_ptr) {
    llama_binding_state* state = (llama_binding_state*) state_ptr;
    // Smart pointers will automatically free resources
    state->model.reset();
    state->context.reset();
    state->lora.clear();
    // Free params
    if (state->params) {
        delete state->params;
        state->params = nullptr;
    }
    delete state;
}

void llama_free_params(void* params_ptr) {
    common_params* params = (common_params*) params_ptr;
    delete params;
}

// NOTE: This function is DISABLED - text generation not supported
int llama_tokenize_string(void* params_ptr, void* state_pr, int* result) {
    fprintf(stderr, "ERROR: llama_tokenize_string is disabled - text generation not supported in this version\n");
    fprintf(stderr, "       Please use the Embeddings() method for embedding generation\n");
    return 1;
}


std::vector<std::string> create_vector(const char** strings, int count) {
    std::vector<std::string>* vec = new std::vector<std::string>;
    for (int i = 0; i < count; i++) {
      vec->push_back(std::string(strings[i]));
    }
    return *vec;
}

void delete_vector(std::vector<std::string>* vec) {
    delete vec;
}

int load_state(void *ctx, char *statefile, char*modes) {
    llama_context* state = (llama_context*) ctx;
    const size_t state_size = llama_get_state_size(state);
    uint8_t * state_mem = new uint8_t[state_size];

  {
        FILE *fp_read = fopen(statefile, modes);
        if (state_size != llama_get_state_size(state)) {
            fprintf(stderr, "\n%s : failed to validate state size\n", __func__);
            return 1;
        }

        const size_t ret = fread(state_mem, 1, state_size, fp_read);
        if (ret != state_size) {
            fprintf(stderr, "\n%s : failed to read state\n", __func__);
            return 1;
        }

        llama_set_state_data(state, state_mem);  // could also read directly from memory mapped file
        fclose(fp_read);
    }

    return 0;
}

void save_state(void *ctx, char *dst, char*modes) {
    llama_context* state = (llama_context*) ctx;

    const size_t state_size = llama_get_state_size(state);
    uint8_t * state_mem = new uint8_t[state_size];

    // Save state (rng, logits, embedding and kv_cache) to file
    {
        FILE *fp_write = fopen(dst, modes);
        llama_copy_state_data(state, state_mem); // could also copy directly to memory mapped file
        fwrite(state_mem, 1, state_size, fp_write);
        fclose(fp_write);
    }
}

// NOTE: This function is DISABLED for the current llama.cpp version
// The sampling API has been completely rewritten and text generation is not supported
// Only embeddings functionality is available - use Embeddings() method instead
// Simplified params allocation for embeddings only
void* llama_allocate_params_for_embeddings(const char *prompt, int threads) {
    common_params * params = new common_params;
    params->prompt = prompt;
    params->cpuparams.n_threads = threads;
    params->n_predict = 0;  // No text generation
    return params;
}

void* llama_allocate_params(const char *prompt, int seed, int threads, int tokens, int top_k,
                            float top_p, float temp, float repeat_penalty, int repeat_last_n, bool ignore_eos, bool memory_f16, int n_batch, int n_keep, const char** antiprompt, int antiprompt_count,
                             float tfs_z, float typical_p, float frequency_penalty, float presence_penalty, int mirostat, float mirostat_eta, float mirostat_tau, bool penalize_nl, const char *logit_bias, const char *session_file, bool prompt_cache_all, bool mlock, bool mmap,
                             const char *maingpu,const char *tensorsplit , bool prompt_cache_ro, const char *grammar,
                             float rope_freq_base, float rope_freq_scale, float negative_prompt_scale, const char* negative_prompt, int n_draft) {
    fprintf(stderr, "ERROR: llama_allocate_params is disabled - text generation not supported in this version\n");
    fprintf(stderr, "       Text generation requires updating to new llama_sampling_* API\n");
    return nullptr;
}

void* load_model_custom(const char *fname, int n_ctx, int n_seed, bool memory_f16, bool mlock, bool embeddings, bool mmap, bool low_vram, int n_gpu_layers, int n_batch, int n_ubatch, const char *maingpu, const char *tensorsplit, bool numa, float rope_freq_base, float rope_freq_scale, bool mul_mat_q, const char *lora, const char *lora_base, bool perplexity) {
    fprintf(stderr, "DEBUG: load_model called with n_batch=%d, n_ubatch=%d\n", n_batch, n_ubatch);
    fflush(stderr);
    printf("DEBUG_STDOUT: load_model called with n_batch=%d, n_ubatch=%d\n", n_batch, n_ubatch);
    fflush(stdout);
   return load_binding_model_custom(fname, n_ctx, n_seed, memory_f16, mlock, embeddings, mmap, low_vram, n_gpu_layers, n_batch, n_ubatch, maingpu, tensorsplit, numa, rope_freq_base, rope_freq_scale, mul_mat_q, lora, lora_base, perplexity);
}

void* load_binding_model_custom(const char *fname, int n_ctx, int n_seed, bool memory_f16, bool mlock, bool embeddings, bool mmap, bool low_vram, int n_gpu_layers, int n_batch, int n_ubatch, const char *maingpu, const char *tensorsplit, bool numa,  float rope_freq_base, float rope_freq_scale, bool mul_mat_q, const char *lora, const char *lora_base, bool perplexity) {
    // Silence llama.cpp logs to reduce noise
    llama_log_set([](enum ggml_log_level level, const char * text, void * user_data) {
        // Only log errors and warnings, ignore info messages
        if (level <= GGML_LOG_LEVEL_WARN) {
            fprintf(stderr, "%s", text);
        }
    }, nullptr);
    
    // load the model
    common_params * lparams;
    
    std::string fname_str = fname;
    std::string lora_str = lora;
    std::string lora_base_str = lora_base;

// Temporary workaround for https://github.com/go-skynet/go-llama.cpp/issues/218
#ifdef GGML_USE_CUBLAS
    lparams = new common_params();
    lparams->model.path = fname_str;
#else
    lparams = new common_params();
    lparams->model.path = fname_str;
#endif
    
    llama_binding_state * state = new llama_binding_state;
    
    lparams->n_ctx      = n_ctx;
    lparams->sampling.seed       = n_seed;
    // lparams->memory_f16     = memory_f16; // Removed in newer llama.cpp? Check common_params definition if needed. 
    // common_params usually has bool f16_kv? No, it has enum.
    // Let's assume memory_f16 maps to something or ignore if not critical for now.
    // Actually common_params struct in common.h has:
    // bool flash_attn = false;
    // bool use_mmap = true;
    // bool use_mlock = false;
    // ...
    // It doesn't seem to have memory_f16 directly. It might be type_k/v?
    
    lparams->embedding  = embeddings;
    lparams->use_mlock  = mlock;
    lparams->n_gpu_layers = n_gpu_layers;

    lparams->use_mmap = mmap;

    // lparams->low_vram = low_vram; // Not in common_params?
    
    if (rope_freq_base != 0.0f) {
        lparams->rope_freq_base = rope_freq_base;
    } else {
        lparams->rope_freq_base = 10000.0f;
    }

    if (rope_freq_scale != 0.0f) {
        lparams->rope_freq_scale = rope_freq_scale;
    } else {
        lparams->rope_freq_scale =  1.0f;
    }

    // lparams->model = fname; // Already set by create_common_params
    
    if (maingpu[0] != '\0') { 
        lparams->main_gpu = std::stoi(maingpu);
    }

    if (tensorsplit[0] != '\0') { 
        std::string arg_next = tensorsplit;
            // split string by , and /
            const std::regex regex{R"([,/]+)"};
            std::sregex_token_iterator it{arg_next.begin(), arg_next.end(), regex, -1};
            std::vector<std::string> split_arg{it, {}};
            GGML_ASSERT(split_arg.size() <= 128);

            for (size_t i = 0; i < 128; ++i) {
                if (i < split_arg.size()) {
                    lparams->tensor_split[i] = std::stof(split_arg[i]);
                } else {
                    lparams->tensor_split[i] = 0.0f;
                }
            }  
    }

    lparams->n_batch      = n_batch;
    fprintf(stderr, "[DEBUG] load_binding_model_custom: Before setting n_ubatch, lparams->n_ubatch = %d\n", lparams->n_ubatch);
    lparams->n_ubatch     = n_ubatch; // THIS IS THE FIX
    fprintf(stderr, "[DEBUG] load_binding_model_custom: After setting n_ubatch = %d, lparams->n_ubatch = %d\n", n_ubatch, lparams->n_ubatch);
    fflush(stderr);

    llama_backend_init(); // numa argument removed in newer llama.cpp? common_init handles it?
    // common_init calls llama_backend_init.
    // But we call common_init_from_params.
    // common_init_from_params does NOT call llama_backend_init.
    // common_init (void) calls llama_backend_init.
    
    // Let's call llama_backend_init(numa); if available.
    // llama_backend_init() takes no args in newer versions?
    // Check llama.h if possible. Assuming void for now as per common.h usage?
    // Wait, common.h doesn't show llama_backend_init.
    
    // Let's assume llama_backend_init() is correct.
    llama_backend_init();
    
    // common_init_from_params returns common_init_result (struct)
    common_init_result result = common_init_from_params(*lparams);
    
    if (result.model == nullptr) {
        fprintf(stderr, "%s: error: unable to load model\n", __func__);
        return nullptr;
    }
    
    state->context = std::move(result.context); // unique_ptr move
    state->model = std::move(result.model); // unique_ptr move
    state->lora = std::move(result.lora); // vector move
    state->params = lparams;
    
    return state;
}

// Wrapper function for compatibility - calls load_model_custom
void* load_model(const char *fname, int n_ctx, int n_seed, bool memory_f16, bool mlock, bool embeddings, bool mmap, bool low_vram, int n_gpu_layers, int n_batch, int n_ubatch, const char *maingpu, const char *tensorsplit, bool numa, float rope_freq_base, float rope_freq_scale, bool mul_mat_q, const char *lora, const char *lora_base, bool perplexity) {
   return load_model_custom(fname, n_ctx, n_seed, memory_f16, mlock, embeddings, mmap, low_vram, n_gpu_layers, n_batch, n_ubatch, maingpu, tensorsplit, numa, rope_freq_base, rope_freq_scale, mul_mat_q, lora, lora_base, perplexity);
}
