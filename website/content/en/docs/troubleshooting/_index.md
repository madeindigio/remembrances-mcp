---
title: "Troubleshooting"
linkTitle: "Troubleshooting"
weight: 6
description: >
  Common problems and solutions for Remembrances MCP
---

## Installation Problems

### Compilation Fails with llama.cpp Errors

**Problem**: The compilation process fails when compiling llama.cpp dependencies.

**Solutions**:

1. **Make sure you have the necessary compilation tools**:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install build-essential cmake

   # macOS
   xcode-select --install

   # Fedora
   sudo dnf install gcc-c++ cmake
   ```

2. **Check the Go version** (requires Go 1.20+):
   ```bash
   go version
   ```

3. **Clean and recompile**:
   ```bash
   make clean
   make build
   ```

### Missing GPU Support

**Problem**: GPU acceleration doesn't work even though you have a compatible GPU.

**Solutions**:

1. **NVIDIA (CUDA)**:
   ```bash
   # Check CUDA installation
   nvidia-smi
   nvcc --version
   
   # Make sure CUDA toolkit is installed
   # Ubuntu/Debian
   sudo apt-get install nvidia-cuda-toolkit
   ```

2. **AMD (ROCm)**:
   ```bash
   # Check ROCm installation
   rocm-smi
   
   # Install ROCm if missing
   # Follow instructions at https://rocm.docs.amd.com/
   ```

3. **Apple Silicon (Metal)**:
   Metal should work automatically on macOS with Apple Silicon. Make sure you're running the native ARM64 compilation.

## Runtime Problems

### Out of Memory (OOM) Errors

**Problem**: The server crashes with memory errors when processing embeddings.

**Solutions**:

1. **Reduce GPU layers**:
   ```bash
   # Use fewer GPU layers
   --gguf-gpu-layers 16  # instead of 32
   
   # Or disable GPU completely
   --gguf-gpu-layers 0
   ```

2. **Use a smaller model**:
   - Switch from `nomic-embed-text-v1.5` to `all-MiniLM-L6-v2`
   - Use a more quantized version (Q4_K_M instead of Q8_0)

3. **Reduce batch size** (if applicable):
   ```bash
   --gguf-batch-size 256  # default is usually higher
   ```

### Model Won't Load

**Problem**: The server won't start with "model not found" errors or similar.

**Solutions**:

1. **Check file path and permissions**:
   ```bash
   ls -lh ./model.gguf
   chmod +r ./model.gguf
   ```

2. **Verify the model file is not corrupted**:
   ```bash
   # Check that file size matches expected
   ls -lh ./model.gguf
   
   # Re-download if necessary
   wget https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf
   ```

3. **Use absolute path**:
   ```bash
   --gguf-model-path /complete/path/to/model.gguf
   ```

### Slow Performance

**Problem**: Embedding generation or search is slower than expected.

**Solutions**:

1. **Enable GPU acceleration**:
   ```bash
   --gguf-gpu-layers 32
   ```

2. **Increase thread count**:
   ```bash
   --gguf-threads 8  # adjust according to your CPU cores
   ```

3. **Use a faster model**:
   - `all-MiniLM-L6-v2` is significantly faster than `nomic-embed-text-v1.5`

4. **Check thermal throttling**:
   ```bash
   # Monitor CPU/GPU temperatures
   # NVIDIA
   nvidia-smi -l 1
   
   # CPU
   sensors  # Linux
   ```

## Database Problems

### Database Connection Fails

**Problem**: Cannot connect to SurrealDB (embedded or external).

**Solutions**:

1. **For embedded database**:
   ```bash
   # Check file permissions
   ls -la ./remembrances.db
   
   # Make sure directory exists and has write permissions
   mkdir -p ./data
   chmod 755 ./data
   --db-path ./data/remembrances.db
   ```

2. **For external SurrealDB**:
   ```bash
   # Verify SurrealDB is running
   curl http://localhost:8000/health
   
   # Check connection parameters
   --surrealdb-url ws://localhost:8000
   --surrealdb-user root
   --surrealdb-pass root
   ```

### Database Corruption

**Problem**: Database errors or inconsistent data after a crash.

**Solutions**:

1. **Backup and recreate**:
   ```bash
   # Backup existing data
   cp ./remembrances.db ./remembrances.db.backup
   
   # Remove corrupted database
   rm ./remembrances.db
   
   # Restart - will create new database
   ./remembrances-mcp --gguf-model-path ./model.gguf
   ```

2. **Run with debug logging** to identify issues:
   ```bash
   --log-level debug
   ```

## MCP Connection Problems

### Claude Desktop Won't Connect

**Problem**: Claude Desktop doesn't recognize or connect to Remembrances MCP.

**Solutions**:

1. **Check configuration file location**:
   - macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
   - Linux: `~/.config/claude/claude_desktop_config.json`

2. **Verify JSON syntax**:
   ```bash
   # Validate JSON
   cat ~/.config/claude/claude_desktop_config.json | python -m json.tool
   ```

3. **Use absolute paths** in configuration:
   ```json
   {
     "mcpServers": {
       "remembrances": {
         "command": "/usr/local/bin/remembrances-mcp",
         "args": [
           "--gguf-model-path",
           "/home/user/models/nomic-embed-text-v1.5.Q4_K_M.gguf"
         ]
       }
     }
   }
   ```

4. **Restart Claude Desktop** after configuration changes.

### MCP Streamable HTTP / HTTP API Problems

**Problem**: Cannot connect via MCP Streamable HTTP (MCP tools) or the HTTP JSON API.

**Solutions**:

1. **Check if port is in use**:
   ```bash
   # Check port availability
   lsof -i :3000  # MCP Streamable HTTP default
   lsof -i :8080  # HTTP default
   ```

2. **Use a different port**:
   ```bash
   --mcp-http --mcp-http-addr ":3001"
   --http --http-addr ":8081"
   ```

3. **Check firewall configuration**:
   ```bash
   # Allow port (Linux with ufw)
   sudo ufw allow 8080/tcp
   ```

## Embedding Problems

### Inconsistent Search Results

**Problem**: Search results vary or don't match expected content.

**Solutions**:

1. **Ensure consistent embedding model** - don't mix embeddings from different models

2. **Verify embedding dimensions match**:
   - `nomic-embed-text-v1.5`: 768 dimensions
   - `all-MiniLM-L6-v2`: 384 dimensions

3. **Re-index after model change**:
   ```bash
   # You may need to re-generate embeddings for all content if you change models
   ```

### Embeddings Not Generated

**Problem**: Content is stored but embeddings are empty or missing.

**Solutions**:

1. **Check embedder configuration**:
   ```bash
   # Verify model is specified
   --gguf-model-path ./model.gguf
   # Or
   --ollama-model nomic-embed-text
   # Or
   --openai-key sk-xxx
   ```

2. **Enable debug logging**:
   ```bash
   --log-level debug
   ```

## Getting Help

If you're still experiencing problems:

1. **Check logs** with debug mode:
   ```bash
   --log-level debug
   ```

2. **Search existing issues** in [GitHub Issues](https://github.com/madeindigio/remembrances-mcp/issues)

3. **Open a new issue** with:
   - Operating system and version
   - Go version (`go version`)
   - GPU type (if applicable)
   - Complete error message
   - Steps to reproduce

## See Also

- [Getting Started](../getting-started/) - Installation guide
- [Configuration](../configuration/) - Configuration options
- [GGUF Models](../gguf-models/) - Model selection and optimization
```
