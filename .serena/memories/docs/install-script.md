# Remembrances-MCP Installation Script

## Overview

The installation script (`scripts/install.sh`) automates the installation of Remembrances-MCP on Linux and macOS systems.

## Supported Platforms and Builds

| Platform | Architecture | Build Variant | Description |
|----------|-------------|---------------|-------------|
| macOS | arm64 (M1/M2/M3) | `darwin-arm64` | Apple Silicon Macs |
| Linux | amd64 | `linux-amd64-nvidia` | NVIDIA GPU + Modern Intel CPU (10th gen+) |
| Linux | amd64 | `linux-amd64-nvidia-portable` | NVIDIA GPU + AMD Ryzen or older Intel |
| Linux | amd64 | `linux-amd64-cpu-only` | No NVIDIA GPU (CPU only) |

**Not supported:**
- macOS Intel (x86_64)
- Linux ARM64

## Environment Variables

For non-interactive mode (`curl | bash`), use these environment variables:

| Variable | Values | Description |
|----------|--------|-------------|
| `REMEMBRANCES_VERSION` | e.g., `v1.14.1` | Specific version to install |
| `REMEMBRANCES_NVIDIA` | `yes` or `no` | Force NVIDIA or CPU-only build |
| `REMEMBRANCES_DOWNLOAD_MODEL` | `yes` or `no` | Download GGUF model or skip |

## Usage Examples

### Interactive Mode (asks user for input)
```bash
bash <(curl -fsSL https://raw.githubusercontent.com/madeindigio/remembrances-mcp/main/scripts/install.sh)
```

### Non-interactive Mode (uses defaults)
```bash
curl -fsSL https://raw.githubusercontent.com/madeindigio/remembrances-mcp/main/scripts/install.sh | bash
```

### Custom Non-interactive Installation
```bash
# Install CPU-only version without downloading the model
curl -fsSL https://raw.githubusercontent.com/madeindigio/remembrances-mcp/main/scripts/install.sh | \
  REMEMBRANCES_NVIDIA=no REMEMBRANCES_DOWNLOAD_MODEL=no bash

# Install specific version with NVIDIA
curl -fsSL https://raw.githubusercontent.com/madeindigio/remembrances-mcp/main/scripts/install.sh | \
  REMEMBRANCES_VERSION=v1.14.1 REMEMBRANCES_NVIDIA=yes bash
```

## Installation Directories

### Linux
- Binary & libraries: `~/.local/share/remembrances/bin/`
- Configuration: `~/.config/remembrances/config.yaml`
- Database: `~/.local/share/remembrances/remembrances.db`
- Models: `~/.local/share/remembrances/models/`

### macOS
- All files: `~/Library/Application Support/remembrances/`

## CPU Detection Logic

The script automatically detects CPU type to choose the optimal NVIDIA build:

1. **AMD Ryzen detected** → Uses `nvidia-portable`
2. **Modern Intel (10th gen+)** → Uses `nvidia`
3. **Older Intel or unknown** → Uses `nvidia-portable` (safer compatibility)

Detection is based on `/proc/cpuinfo` patterns:
- Looks for "amd" or "ryzen" keywords
- Checks for Intel generation patterns like "11th gen", "12th gen", or model numbers like i7-12700

## Interactive vs Non-interactive Mode

The script detects if stdin is a terminal using `[ -t 0 ]`:

- **Interactive (`[ -t 0 ]` is true)**: Prompts user for choices
- **Non-interactive (piped)**: Uses defaults (Y for all prompts) or environment variables

This fixes the issue where `curl ... | bash` would fail on `read` commands because stdin is the downloaded script, not user input.

## What the Script Installs

1. **Binary**: `remembrances-mcp` executable
2. **Shared libraries**: `.so` or `.dylib` files for GGUF support
3. **Configuration**: `config.yaml` with default settings
4. **GGUF Model** (optional): `nomic-embed-text-v1.5.Q4_K_M.gguf` (~260MB)

## Release File Naming Convention

As of v1.14.1, release files follow this naming:
```
remembrances-mcp-{os}-{arch}[-variant].zip
```

Examples:
- `remembrances-mcp-darwin-arm64.zip`
- `remembrances-mcp-linux-amd64-cpu-only.zip`
- `remembrances-mcp-linux-amd64-nvidia.zip`
- `remembrances-mcp-linux-amd64-nvidia-portable.zip`
