#ifndef SEARCH_STATIC_H
#define SEARCH_STATIC_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>

// Opaque pointer types for model and context
typedef void* model_ptr;
typedef void* context_ptr;

// Load a model from a GGUF file
// Returns NULL on failure
// Parameters:
//   - path_model: Path to the .gguf model file
//   - n_gpu_layers: Number of layers to offload to GPU (0 = CPU only)
model_ptr load_model(const char* path_model, uint32_t n_gpu_layers);

// Get the embedding dimension for the loaded model
// Returns -1 on error
// Parameters:
//   - model: Model pointer from load_model
int32_t get_embedding_size(model_ptr model);

// Create a context for the model
// Returns NULL on failure
// Parameters:
//   - model: Model pointer from load_model
//   - ctx_size: Context size (0 for default)
//   - embeddings: Whether to enable embeddings mode (1 = yes, 0 = no)
context_ptr create_context(model_ptr model, uint32_t ctx_size, int embeddings);

// Generate embeddings for text
// Returns 0 on success, non-zero on error
// Error codes:
//   0 = Success
//   1 = Token count exceeds batch size
//   2 = Last token is not SEP
//   3 = Failed to encode/decode text
//   4 = Invalid context or model
// Parameters:
//   - ctx: Context pointer from create_context
//   - text: UTF-8 encoded text to embed
//   - out_embeddings: Output array (must be allocated to embedding_size floats)
//   - out_tokens: Optional pointer to receive token count (can be NULL)
int embed_text(context_ptr ctx, const char* text, float* out_embeddings, uint32_t* out_tokens);

// Free a context
// Parameters:
//   - ctx: Context pointer to free
void free_context(context_ptr ctx);

// Free a model
// Parameters:
//   - model: Model pointer to free
void free_model(model_ptr model);

// Initialize the library with a specific log level
// Should be called once at program startup
// Parameters:
//   - log_level: Log level (0=debug, 1=info, 2=warn, 3=error, 4=none)
void init_library(int log_level);

#ifdef __cplusplus
}
#endif

#endif // SEARCH_STATIC_H