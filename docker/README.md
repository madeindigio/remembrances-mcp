# Docker Support for Remembrances-MCP

This directory contains Docker configuration for building and running Remembrances-MCP with CUDA/GPU support.

## Quick Start

```bash
# 1. Prepare (build binary + download model)
make docker-prepare-cuda

# 2. Build Docker image
make docker-build-cuda

# 3. Run with GPU support
make docker-run-cuda
```

## Prerequisites

1. **NVIDIA GPU** with CUDA support
2. **NVIDIA Container Toolkit** installed ([installation guide](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html))
3. **Docker** with GPU support enabled
4. **Local build environment** for compiling CUDA variant (CUDA toolkit, Go, etc.)

### Verify GPU Support

```bash
# Check NVIDIA driver
nvidia-smi

# Check Docker GPU access
docker run --rm --gpus all nvidia/cuda:12.4.1-base-ubuntu22.04 nvidia-smi
```

## Build Process

The Docker image uses **locally pre-compiled binaries** because CUDA libraries require native compilation. This is a two-step process:

### Step 1: Build CUDA Binary Locally

```bash
# Build the CUDA variant (requires CUDA toolkit)
make dist-variant VARIANT=cuda

# This creates:
# - dist-variants/remembrances-mcp-cuda-linux-x86_64/remembrances-mcp
# - dist-variants/remembrances-mcp-cuda-linux-x86_64/*.so (shared libraries)
```

### Step 2: Download GGUF Model

```bash
# Download the embedding model (~260MB)
make docker-download-model

# This downloads to:
# - models/nomic-embed-text-v1.5.Q4_K_M.gguf
```

### Step 3: Build Docker Image

```bash
make docker-build-cuda
```

This creates the following tags:
- `ghcr.io/madeindigio/remembrances-mcp:cuda`
- `ghcr.io/madeindigio/remembrances-mcp:vX.Y.Z-cuda`
- `ghcr.io/madeindigio/remembrances-mcp:latest`

## Publishing to GitHub Container Registry

### Setup Authentication

```bash
# Create a Personal Access Token with 'write:packages' scope
# https://github.com/settings/tokens

export GITHUB_TOKEN=ghp_your_token_here

# Login to GHCR
make docker-login
```

### Push Image

```bash
make docker-push-cuda
```

## Running the Container

### Basic Run

```bash
make docker-run-cuda
```

### Custom Configuration

```bash
# Override paths and port
make docker-run-cuda \
    DOCKER_DATA_PATH=/my/data/path \
    DOCKER_KB_PATH=/my/knowledge-base \
    DOCKER_PORT=9090
```

### Manual Docker Run

```bash
docker run -d \
    --name remembrances-mcp \
    --gpus all \
    -p 8080:8080 \
    -v /path/to/data:/data \
    -v /path/to/knowledge-base:/knowledge-base \
    ghcr.io/madeindigio/remembrances-mcp:cuda
```

### Environment Variables

Override configuration at runtime using `GOMEM_*` environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `GOMEM_HTTP_ADDR` | `:8080` | HTTP API bind address |
| `GOMEM_GGUF_GPU_LAYERS` | `99` | GPU layers for GGUF model |
| `GOMEM_GGUF_THREADS` | `0` | CPU threads (0=auto) |
| `GOMEM_SURREALDB_USER` | `root` | SurrealDB username |
| `GOMEM_SURREALDB_PASS` | `root` | SurrealDB password |
| `GOMEM_SURREALDB_NAMESPACE` | `remembrances` | SurrealDB namespace |
| `GOMEM_SURREALDB_DATABASE` | `production` | SurrealDB database |

Example:

```bash
docker run -d \
    --name remembrances-mcp \
    --gpus all \
    -p 8080:8080 \
    -e GOMEM_GGUF_GPU_LAYERS=50 \
    -e GOMEM_SURREALDB_PASS=mysecretpassword \
    -v /path/to/data:/data \
    -v /path/to/kb:/knowledge-base \
    ghcr.io/madeindigio/remembrances-mcp:cuda
```

## Volume Mounts

| Path | Purpose | Required |
|------|---------|----------|
| `/data` | SurrealDB database storage | **Yes** (for persistence) |
| `/knowledge-base` | Knowledge base documents | **Yes** (for persistence) |
| `/models` | Custom GGUF models | No (built-in model included) |
| `/config` | Custom configuration | No (built-in config included) |

### Using a Custom Model

```bash
docker run -d \
    --name remembrances-mcp \
    --gpus all \
    -p 8080:8080 \
    -v /path/to/custom-model.gguf:/models/nomic-embed-text-v1.5.Q4_K_M.gguf \
    -v /path/to/data:/data \
    -v /path/to/kb:/knowledge-base \
    ghcr.io/madeindigio/remembrances-mcp:cuda
```

### Using a Custom Configuration

```bash
# Create your custom config.yaml based on config.sample.gguf.yaml
docker run -d \
    --name remembrances-mcp \
    --gpus all \
    -p 8080:8080 \
    -v /path/to/my-config.yaml:/config/config.yaml \
    -v /path/to/data:/data \
    -v /path/to/kb:/knowledge-base \
    ghcr.io/madeindigio/remembrances-mcp:cuda
```

## Container Management

```bash
# View logs
docker logs -f remembrances-mcp

# Stop container
make docker-stop-cuda
# or
docker stop remembrances-mcp && docker rm remembrances-mcp

# Check status
make docker-status

# Shell access (for debugging)
docker exec -it remembrances-mcp /bin/bash
```

## Health Check

The container includes a health check that verifies the HTTP API is responding:

```bash
# Check health manually
curl http://localhost:8080/health

# View health status
docker inspect --format='{{.State.Health.Status}}' remembrances-mcp
```

## Exposed Ports

| Port | Protocol | Description |
|------|----------|-------------|
| 8080 | HTTP | HTTP JSON API (default) |

To expose MCP tools over Streamable HTTP (recommended) instead of the HTTP JSON API:

```bash
docker run -d \
    --name remembrances-mcp \
    --gpus all \
    -p 3000:3000 \
    -v /path/to/data:/data \
    -v /path/to/kb:/knowledge-base \
    ghcr.io/madeindigio/remembrances-mcp:cuda \
    --mcp-http --mcp-http-addr :3000
```

## Troubleshooting

### GPU Not Detected

```bash
# Check NVIDIA runtime is available
docker info | grep -i nvidia

# Verify GPU access in container
docker run --rm --gpus all ghcr.io/madeindigio/remembrances-mcp:cuda nvidia-smi
```

### Permission Denied on Volumes

```bash
# Ensure correct ownership (container runs as uid 999)
sudo chown -R 999:999 /path/to/data /path/to/kb
```

### Model Loading Fails

Check logs for specific errors:

```bash
docker logs remembrances-mcp 2>&1 | grep -i "gguf\|model\|cuda"
```

### Container Exits Immediately

```bash
# Run interactively to see errors
docker run -it --rm --gpus all \
    -v /path/to/data:/data \
    -v /path/to/kb:/knowledge-base \
    ghcr.io/madeindigio/remembrances-mcp:cuda
```

## Building Without Make

If you prefer not to use Make:

```bash
# 1. Build CUDA variant
make dist-variant VARIANT=cuda

# 2. Download model
mkdir -p models
curl -fsSL -o models/nomic-embed-text-v1.5.Q4_K_M.gguf \
    "https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf?download=true"

# 3. Build image
docker build -f docker/Dockerfile.cuda \
    -t ghcr.io/madeindigio/remembrances-mcp:cuda .

# 4. Login and push
echo $GITHUB_TOKEN | docker login ghcr.io -u madeindigio --password-stdin
docker push ghcr.io/madeindigio/remembrances-mcp:cuda
```

## Image Size

The Docker image includes:
- Base: NVIDIA CUDA 12.4.1 runtime (~2.5GB)
- GGUF Model: nomic-embed-text-v1.5.Q4_K_M (~260MB)
- Binary + Libraries: ~150MB
- **Total: ~3GB**

## Security Notes

1. The container runs as non-root user `remembrances` (uid 999)
2. Default SurrealDB credentials are `root/root` - change in production!
3. Set `GOMEM_SURREALDB_PASS` environment variable for custom password
4. Use Docker secrets or external secret management for sensitive data

## License

MIT License - See [LICENSE.txt](../LICENSE.txt)