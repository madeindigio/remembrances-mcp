#!/bin/bash
# Remembrances-MCP Installation Script
# Usage: curl -fsSL https://raw.githubusercontent.com/madeindigio/remembrances-mcp/main/scripts/install.sh | bash
#
# This script will:
# 1. Detect your operating system and architecture
# 2. Detect CPU type (Intel/AMD) and NVIDIA GPU availability
# 3. Download the appropriate binary release
# 4. Install to ~/.local/share/remembrances/ (Linux) or ~/Library/Application Support/remembrances/ (macOS)
# 5. Add the bin directory to your PATH
# 6. Create a default configuration file
# 7. Download the GGUF embedding model
#
# Environment variables for non-interactive mode (curl | bash):
#   REMEMBRANCES_VERSION=latest|vX.Y.Z     - Version to install (default: latest)
#   REMEMBRANCES_NVIDIA=yes|no            - Force NVIDIA or CPU-only build (Linux only)
#   REMEMBRANCES_PORTABLE=yes|no          - Force portable/non-portable build (Linux NVIDIA builds)
#   REMEMBRANCES_DOWNLOAD_MODEL=yes|no    - Download GGUF model or skip
#
# Notes:
#   - This installer prefers *embedded* release assets from the latest GitHub release.
#   - Supported platforms:
#       * Linux x86_64 (amd64)
#       * macOS Apple Silicon (aarch64)
#   - Any other OS/architecture is not supported.

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if running interactively (for curl | bash support)
# When piped, stdin is not a terminal, so we use defaults
INTERACTIVE=false
if [ -t 0 ]; then
    INTERACTIVE=true
fi

# Repository
REPO="madeindigio/remembrances-mcp"

# Version selection (resolved dynamically via GitHub API)
REQUESTED_VERSION="${REMEMBRANCES_VERSION:-latest}"
VERSION="" # resolved tag (e.g., v1.16.10)

# CUDA runtime libraries fallback (Linux only)
CUDA_LIBS_URL="https://github.com/madeindigio/remembrances-mcp/releases/download/v1.16.4/cuda-libs-linux-x64.tar.xz"

# GGUF Model
GGUF_MODEL_URL="https://huggingface.co/nomic-ai/nomic-embed-text-v1.5-GGUF/resolve/main/nomic-embed-text-v1.5.Q4_K_M.gguf?download=true"
GGUF_MODEL_NAME="nomic-embed-text-v1.5.Q4_K_M.gguf"

# Print functions
print_step() {
    echo -e "${BLUE}==>${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}!${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_kv() {
    # Usage: print_kv "Label" "Value"
    printf "  ${BLUE}%-22s${NC} %s\n" "$1" "$2"
}

TOTAL_STEPS=10
CURRENT_STEP=0

progress_step() {
    CURRENT_STEP=$((CURRENT_STEP + 1))
    local pct=$(( (CURRENT_STEP * 100) / TOTAL_STEPS ))
    echo -e "${BLUE}[${pct}%]${NC} $1"
}

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

die() {
    print_error "$1"
    exit 1
}

# Detect OS
detect_os() {
    local os
    os="$(uname -s)"
    case "${os}" in
        Linux*)     echo "linux";;
        Darwin*)    echo "darwin";;
        *)          echo "unsupported";;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch="$(uname -m)"
    case "${arch}" in
        x86_64|amd64)   echo "amd64";;
        arm64|aarch64)  echo "aarch64";;
        *)              echo "unsupported";;
    esac
}

# Check if NVIDIA GPU is available (Linux only)
check_nvidia() {
    if command_exists nvidia-smi; then
        # nvidia-smi may exist but fail if driver isn't loaded
        if nvidia-smi >/dev/null 2>&1; then
            return 0
        fi
    fi
    return 1
}

# Check if CPU supports AVX2 (Linux x86_64)
cpu_has_avx2() {
    if [ -f /proc/cpuinfo ]; then
        if grep -qiE "(^flags\s*:.*\bavx2\b)" /proc/cpuinfo 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

# Check if CPU supports AVX-512 (Linux x86_64)
# We use avx512f as the primary feature bit.
cpu_has_avx512() {
    if [ -f /proc/cpuinfo ]; then
        if grep -qiE "(^flags\s*:.*\bavx512f\b)" /proc/cpuinfo 2>/dev/null; then
            return 0
        fi
    fi
    return 1
}

detect_cuda_major_version() {
    # Returns a CUDA major version number (e.g., 12) or empty if unknown/not found.
    # 1) Prefer nvidia-smi reported CUDA version
    if command_exists nvidia-smi; then
        local line
        line=$(nvidia-smi 2>/dev/null | grep -m1 -E "CUDA Version" || true)
        if [ -n "$line" ]; then
            # Example: "| NVIDIA-SMI 535.183.01   Driver Version: 535.183.01   CUDA Version: 12.2     |"
            echo "$line" | sed -n 's/.*CUDA Version:[[:space:]]*\([0-9][0-9]*\)\..*/\1/p'
            return 0
        fi
    fi

    # 2) Check for libcudart.so.12 presence in dynamic linker cache
    if command_exists ldconfig; then
        if ldconfig -p 2>/dev/null | grep -q "libcudart.so.12"; then
            echo "12"
            return 0
        fi
    fi

    # 3) Fallback: common locations
    if [ -f /usr/local/cuda/lib64/libcudart.so.12 ] || [ -f /usr/local/cuda/lib64/libcudart.so.12.0 ]; then
        echo "12"
        return 0
    fi

    echo ""
}

shared_lib_exists_linux() {
    # Usage: shared_lib_exists_linux "libcudart.so.12"
    # Checks ldconfig cache (if available) and common library directories.
    local soname="$1"

    if command_exists ldconfig; then
        if ldconfig -p 2>/dev/null | grep -qE "\\b${soname//./\\.}\\b"; then
            return 0
        fi
    fi

    # Common locations for CUDA and system libraries
    local d
    for d in \
        /usr/local/cuda/lib64 \
        /usr/lib/x86_64-linux-gnu \
        /lib/x86_64-linux-gnu \
        /usr/lib64 \
        /lib64 \
        /usr/lib \
        /lib
    do
        # Match exact soname or versioned variants (e.g., libcublas.so.12.6.4.1)
        if ls "${d}/${soname}" "${d}/${soname}."* "${d}/${soname}"* >/dev/null 2>&1; then
            return 0
        fi
    done

    return 1
}

cuda_required_runtime_libs_present_linux() {
    # The NVIDIA build of llama/ggml typically requires these CUDA 12 runtime libs.
    # We check for the SONAMEs shown in the release bundle.
    shared_lib_exists_linux "libcudart.so.12" || return 1
    shared_lib_exists_linux "libcublas.so.12" || return 1
    shared_lib_exists_linux "libcublasLt.so.12" || return 1
    return 0
}

cuda_report_missing_runtime_libs_linux() {
    local missing=0
    local lib
    for lib in "libcudart.so.12" "libcublas.so.12" "libcublasLt.so.12"; do
        if ! shared_lib_exists_linux "$lib"; then
            print_warning "Missing CUDA runtime library: ${lib}"
            missing=$((missing + 1))
        fi
    done
    return $missing
}

cuda_llama_deps_resolvable() {
    # If libllama is present, prefer checking what the loader can actually resolve.
    # This is more reliable than trying to infer from CUDA version strings.
    # Returns 0 if the core CUDA deps resolve, 1 otherwise.
    local llama_path=""

    if [ -n "${BIN_DIR:-}" ] && [ -f "${BIN_DIR}/libllama.so" ]; then
        llama_path="${BIN_DIR}/libllama.so"
    elif [ -f "./libllama.so" ]; then
        llama_path="./libllama.so"
    fi

    if [ -z "$llama_path" ]; then
        return 1
    fi
    if ! command_exists ldd; then
        return 1
    fi

    # Note: ldd output is localized on some systems; we only depend on "not found".
    local out
    out=$(ldd "$llama_path" 2>/dev/null || true)
    if echo "$out" | grep -qE "(libcudart\\.so\\.12|libcublas\\.so\\.12|libcublasLt\\.so\\.12).*not found"; then
        return 1
    fi

    # If none of the CUDA libs are referenced at all, fall back to the generic check.
    if ! echo "$out" | grep -qE "libcudart\\.so\\.12|libcublas\\.so\\.12|libcublasLt\\.so\\.12"; then
        return 1
    fi

    return 0
}

http_get() {
    local url="$1"
    if command_exists curl; then
        curl -fsSL "$url"
        return $?
    fi
    if command_exists wget; then
        wget -q -O - "$url"
        return $?
    fi
    die "Neither curl nor wget found. Please install one of them."
}

download_with_progress() {
    local url="$1"
    local out="$2"
    if command_exists curl; then
        curl -fL --progress-bar -o "$out" "$url"
        return $?
    fi
    if command_exists wget; then
        wget --show-progress -q -O "$out" "$url"
        return $?
    fi
    die "Neither curl nor wget found. Please install one of them."
}

github_release_api_url() {
    # Prints GitHub API URL for release metadata
    # Supports: latest, vX.Y.Z
    if [ "${REQUESTED_VERSION}" = "latest" ] || [ -z "${REQUESTED_VERSION}" ]; then
        echo "https://api.github.com/repos/${REPO}/releases/latest"
        return 0
    fi
    echo "https://api.github.com/repos/${REPO}/releases/tags/${REQUESTED_VERSION}"
}

get_release_json() {
    local url
    url=$(github_release_api_url)
    http_get "$url"
}

release_get_tag_name() {
    local json="$1"
    echo "$json" | tr -d '\r' | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}

release_list_asset_urls() {
    local json="$1"
    # Extract browser_download_url values from the GitHub API JSON.
    #
    # IMPORTANT:
    # - This script must work with both GNU and BSD userlands.
    # - macOS/BSD sed does not treat "\?" as an "optional" operator in basic regex
    #   the same way GNU sed can, which previously caused the replacement to fail
    #   and left a trailing quote in URLs (breaking curl with "Malformed input").
    #
    # Use extended regex (-E) and capture the URL content inside quotes.
    echo "$json" | tr -d '\r' | sed -nE 's/.*"browser_download_url"[[:space:]]*:[[:space:]]*"([^\"]+)".*/\1/p'
}

release_find_asset_url() {
    # Usage: release_find_asset_url "$json" "filename.zip"
    local json="$1"
    local filename="$2"
    release_list_asset_urls "$json" | grep -F "/${filename}" | head -n 1
}

detect_release_platform_id() {
    # Maps to the filename platform segment used by releases
    #   linux amd64 -> linux-x64
    #   darwin aarch64 -> darwin-aarch64
    local os="$1"
    local arch="$2"

    if [ "$os" = "linux" ] && [ "$arch" = "amd64" ]; then
        echo "linux-x64"
        return 0
    fi
    if [ "$os" = "darwin" ] && [ "$arch" = "aarch64" ]; then
        echo "darwin-aarch64"
        return 0
    fi
    echo ""
}

ensure_cuda_runtime_libs_linux() {
    # Downloads CUDA runtime libs bundle and installs shared libraries into ~/.local/lib
    # Also marks LD_LIBRARY_PATH setup requirement.
    local target_dir="$HOME/.local/lib"

    if [ -n "${REMEMBRANCES_SKIP_CUDA_LIBS:-}" ] && [ "${REMEMBRANCES_SKIP_CUDA_LIBS}" = "yes" ]; then
        print_warning "Skipping CUDA runtime libs download (REMEMBRANCES_SKIP_CUDA_LIBS=yes)"
        return 0
    fi

    if ! command_exists tar; then
        die "tar is required to install CUDA libraries (tar.xz). Please install tar."
    fi

    mkdir -p "$target_dir"

    local temp_dir
    temp_dir="$(mktemp -d)"
    local archive="$temp_dir/cuda-libs-linux-x64.tar.xz"

    print_step "Downloading CUDA runtime libraries (CUDA 12+) ..."
    download_with_progress "$CUDA_LIBS_URL" "$archive"

    print_step "Extracting CUDA runtime libraries..."
    tar -xJf "$archive" -C "$temp_dir"

    # Copy .so files into ~/.local/lib
    local copied=0
    while IFS= read -r -d '' f; do
        cp -f "$f" "$target_dir/"
        copied=$((copied + 1))
    done < <(find "$temp_dir" -type f \( -name '*.so' -o -name '*.so.*' \) -print0)

    if [ "$copied" -gt 0 ]; then
        print_success "Installed ${copied} CUDA libraries to ${target_dir}"
        NEEDS_LD_LIBRARY_PATH=true
    else
        print_warning "No .so CUDA libraries found in the archive; the bundle format may have changed"
    fi

    rm -rf "$temp_dir"
}

# Get installation directories based on OS
get_install_dirs() {
    local os="$1"

    if [ "$os" = "darwin" ]; then
        INSTALL_DIR="$HOME/Library/Application Support/remembrances"
        CONFIG_DIR="$HOME/Library/Application Support/remembrances"
        DATA_DIR="$HOME/Library/Application Support/remembrances"
    else
        INSTALL_DIR="$HOME/.local/share/remembrances"
        CONFIG_DIR="$HOME/.config/remembrances"
        DATA_DIR="$HOME/.local/share/remembrances"
    fi

    # Binary and shared libraries go in the same directory
    BIN_DIR="${INSTALL_DIR}/bin"
    MODELS_DIR="${INSTALL_DIR}/models"
}

choose_release_filename() {
    # Chooses a filename from release assets.
    # Prefers embedded assets whenever possible.
    local os="$1"      # linux|darwin
    local arch="$2"    # amd64|aarch64
    local want_nvidia="$3"   # true|false
    local want_portable="$4" # true|false

    if [ "$os" = "darwin" ] && [ "$arch" = "aarch64" ]; then
        echo "remembrances-mcp-darwin-aarch64-embedded.zip"
        return 0
    fi

    if [ "$os" = "linux" ] && [ "$arch" = "amd64" ]; then
        if [ "$want_nvidia" = "true" ]; then
            if [ "$want_portable" = "true" ]; then
                echo "remembrances-mcp-embedded-cuda-portable-linux-x86_64.zip"
            else
                echo "remembrances-mcp-embedded-cuda-linux-x86_64.zip"
            fi
            return 0
        fi

        # CPU build: prefer embedded if present; otherwise fall back to cpu.zip.
        echo "remembrances-mcp-embedded-cpu-linux-x86_64.zip"
        return 0
    fi

    echo ""
}

# Download and extract release
download_release() {
    local url="$1"
    local temp_dir
    temp_dir="$(mktemp -d)"
    local zip_file="${temp_dir}/release.zip"

    # Expose temp dir to caller for cleanup
    DOWNLOAD_TEMP_DIR="${temp_dir}"

    # IMPORTANT: this function is often called in contexts where stdout is parsed.
    # Keep stdout reserved for structured output only; send status to stderr.
    print_step "Downloading release..." >&2
    download_with_progress "${url}" "${zip_file}" >&2

    print_success "Download complete" >&2

    print_step "Extracting files..." >&2

    if command -v unzip &> /dev/null; then
        unzip -q "${zip_file}" -d "${temp_dir}"
    else
        print_error "unzip command not found. Please install it."
        exit 1
    fi

    # Find the extracted directory (many zips include a top-level folder, but not always)
    local extracted_dir
    extracted_dir=$(find "${temp_dir}" -mindepth 1 -maxdepth 1 -type d | head -n 1)

    # If the zip extracted files directly into temp_dir, use temp_dir
    if [ -z "${extracted_dir}" ]; then
        extracted_dir="${temp_dir}"
    fi

    EXTRACTED_DIR="${extracted_dir}"

    # Backward compatible: return the path on stdout (single line)
    echo "${extracted_dir}"
}

# Install files to destination
install_files() {
    local src_dir="$1"

    print_step "Installing to ${INSTALL_DIR}..."

    # Create directories
    mkdir -p "${BIN_DIR}"
    mkdir -p "${CONFIG_DIR}"
    mkdir -p "${DATA_DIR}"
    mkdir -p "${MODELS_DIR}"

    # Copy binary (some zips contain a top-level folder; search recursively)
    local bin_path=""
    if [ -f "${src_dir}/remembrances-mcp" ]; then
        bin_path="${src_dir}/remembrances-mcp"
    else
        bin_path=$(find "${src_dir}" -type f -name "remembrances-mcp" 2>/dev/null | head -n 1 || true)
    fi

    if [ -z "${bin_path}" ]; then
        print_error "Binary not found in release"
        print_warning "Debug hint: extracted dir was: ${src_dir}"
        print_warning "Top-level contents:"
        ls -la "${src_dir}" 2>/dev/null || true
        exit 1
    fi

    local base_dir
    base_dir="$(dirname "${bin_path}")"
    print_step "Using release directory: ${base_dir}"

    cp "${bin_path}" "${BIN_DIR}/"
    chmod +x "${BIN_DIR}/remembrances-mcp"
    print_success "Binary installed to ${BIN_DIR}/remembrances-mcp"

    # Copy shared libraries to the SAME directory as binary
    # The binary is compiled to look for libraries in its own directory first
    local lib_count=0
    for lib in "${base_dir}"/*.so "${base_dir}"/*.so.* "${base_dir}"/*.dylib; do
        if [ -f "$lib" ]; then
            cp "$lib" "${BIN_DIR}/"
            lib_count=$((lib_count + 1))
        fi
    done

    if [ $lib_count -gt 0 ]; then
        print_success "${lib_count} shared libraries installed to ${BIN_DIR}/"
    fi

    # Copy sample configs for reference (may live alongside binary or one folder up)
    local cfg
    cfg="${base_dir}/config.sample.yaml"
    if [ -f "$cfg" ]; then
        cp "$cfg" "${CONFIG_DIR}/"
    else
        cfg=$(find "${src_dir}" -type f -name "config.sample.yaml" 2>/dev/null | head -n 1 || true)
        if [ -n "$cfg" ] && [ -f "$cfg" ]; then
            cp "$cfg" "${CONFIG_DIR}/"
        fi
    fi

    cfg="${base_dir}/config.sample.gguf.yaml"
    if [ -f "$cfg" ]; then
        cp "$cfg" "${CONFIG_DIR}/"
    else
        cfg=$(find "${src_dir}" -type f -name "config.sample.gguf.yaml" 2>/dev/null | head -n 1 || true)
        if [ -n "$cfg" ] && [ -f "$cfg" ]; then
            cp "$cfg" "${CONFIG_DIR}/"
        fi
    fi
}

# Create configuration file
create_config() {
    local config_file="${CONFIG_DIR}/config.yaml"
    local db_path="surrealkv://${DATA_DIR}/remembrances.db"
    local model_path="${MODELS_DIR}/${GGUF_MODEL_NAME}"
    local kb_path="${DATA_DIR}/knowledge-base"

    # Create knowledge base directory
    mkdir -p "${kb_path}"

    if [ -f "${config_file}" ]; then
        print_warning "Configuration file already exists at ${config_file}"
        print_warning "Saving new config as ${config_file}.new"
        config_file="${config_file}.new"
    fi

    print_step "Creating configuration file..."

    cat > "${config_file}" << EOF
# Remembrances-MCP Configuration
# Generated by install.sh on $(date)
#
# For all available options, see config.sample.gguf.yaml
#
# Environment variables use the GOMEM_ prefix (e.g., GOMEM_SSE_ADDR).
# Command-line flags take precedence over YAML, and environment variables over both.

# Path to the knowledge base directory
knowledge-base: "${kb_path}"

# ========== SurrealDB Configuration ==========
# Path to the embedded SurrealDB database
db-path: "${db_path}"

# SurrealDB credentials
surrealdb-user: "root"
surrealdb-pass: "root"
surrealdb-namespace: "test"
surrealdb-database: "test"

# ========== GGUF Local Model Configuration ==========
# Path to GGUF model file for local embeddings
# Using nomic-embed-text v1.5 for high-quality embeddings
gguf-model-path: "${model_path}"

# Number of threads for GGUF model (0 = auto-detect)
gguf-threads: 0

# Number of GPU layers for GGUF model (0 = CPU only)
# Increase this value if you have a GPU to offload computation
gguf-gpu-layers: 0

# ========== Text Chunking Configuration ==========
# Maximum chunk size in characters for text splitting
chunk-size: 1500

# Overlap between chunks in characters
chunk-overlap: 200

# ========== Logging Configuration ==========
# Uncomment to enable logging to file
#log: "${DATA_DIR}/remembrances-mcp.log"
EOF

    print_success "Configuration created at ${config_file}"
}

# Download GGUF model
download_gguf_model() {
    local model_path="${MODELS_DIR}/${GGUF_MODEL_NAME}"

    if [ -f "${model_path}" ]; then
        print_warning "GGUF model already exists at ${model_path}"
        return 0
    fi

    print_step "Downloading GGUF embedding model (this may take a few minutes)..."
    print_warning "Model size: ~260MB"

    mkdir -p "${MODELS_DIR}"

    if command -v curl &> /dev/null; then
        curl -fsSL --progress-bar -o "${model_path}" "${GGUF_MODEL_URL}"
    elif command -v wget &> /dev/null; then
        wget --show-progress -q -O "${model_path}" "${GGUF_MODEL_URL}"
    fi

    if [ -f "${model_path}" ]; then
        print_success "GGUF model downloaded to ${model_path}"
    else
        print_error "Failed to download GGUF model"
        print_warning "You can download it manually from:"
        print_warning "${GGUF_MODEL_URL}"
        print_warning "And save it to: ${model_path}"
    fi
}

# Add to PATH in shell configuration files
setup_path() {
    local path_line="export PATH=\"\$PATH:${BIN_DIR}\""
    local ld_line="export LD_LIBRARY_PATH=\"$HOME/.local/lib:\${LD_LIBRARY_PATH:-}\""

    local shell_configs=()
    local os="$1"

    # Check which shell config files exist or should be created
    if [ -f "$HOME/.bashrc" ] || [ ! -f "$HOME/.zshrc" ]; then
        shell_configs+=("$HOME/.bashrc")
    fi

    if [ -f "$HOME/.zshrc" ] || [ "$SHELL" = "/bin/zsh" ] || [ "$SHELL" = "/usr/bin/zsh" ]; then
        shell_configs+=("$HOME/.zshrc")
    fi

    # Also check for .bash_profile on macOS
    if [ "$os" = "darwin" ] && [ -f "$HOME/.bash_profile" ]; then
        shell_configs+=("$HOME/.bash_profile")
    fi

    print_step "Setting up PATH..."

    for config in "${shell_configs[@]}"; do
        # Create file if it doesn't exist
        touch "${config}"

        # Check if our path is already there
        if ! grep -q "remembrances/bin" "${config}" 2>/dev/null; then
            echo "" >> "${config}"
            echo "# Remembrances-MCP" >> "${config}"
            echo "${path_line}" >> "${config}"

            print_success "Added to ${config}"
        else
            print_warning "PATH already configured in ${config}"
        fi

        # If we installed CUDA runtime libs into ~/.local/lib, ensure LD_LIBRARY_PATH includes it
        if [ "${NEEDS_LD_LIBRARY_PATH:-false}" = "true" ]; then
            if ! grep -q "LD_LIBRARY_PATH=.*\\.local/lib" "${config}" 2>/dev/null; then
                echo "${ld_line}" >> "${config}"
                print_success "Added LD_LIBRARY_PATH to ${config}"
            else
                print_warning "LD_LIBRARY_PATH already configured in ${config}"
            fi
        fi
    done
}

# Print final instructions
print_instructions() {
    local os="$1"

    echo ""
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}           Remembrances-MCP Installation Complete!              ${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    echo -e "Version installed:      ${BLUE}${VERSION}${NC}"
    echo -e "Installation directory: ${BLUE}${INSTALL_DIR}${NC}"
    echo -e "Binary & libraries:     ${BLUE}${BIN_DIR}/${NC}"
        if [ "${NEEDS_LD_LIBRARY_PATH:-false}" = "true" ]; then
            echo -e "CUDA runtime libs:      ${BLUE}$HOME/.local/lib${NC} (added to LD_LIBRARY_PATH)"
        fi
    echo -e "Configuration file:     ${BLUE}${CONFIG_DIR}/config.yaml${NC}"
    echo -e "Database location:      ${BLUE}${DATA_DIR}/remembrances.db${NC}"
    echo -e "GGUF model:             ${BLUE}${MODELS_DIR}/${GGUF_MODEL_NAME}${NC}"
    echo ""
    echo -e "${YELLOW}To complete the installation, run one of the following:${NC}"
    echo ""
    echo -e "  ${BLUE}source ~/.bashrc${NC}     # If using bash"
    echo -e "  ${BLUE}source ~/.zshrc${NC}      # If using zsh"
    echo ""
    echo -e "Or simply open a new terminal window."
    echo ""
    echo -e "${YELLOW}To verify the installation:${NC}"
    echo ""
    echo -e "  ${BLUE}remembrances-mcp --help${NC}"
    echo ""
    echo -e "${YELLOW}To configure for your MCP client (e.g., Claude Desktop):${NC}"
    echo ""

    if [ "$os" = "darwin" ]; then
        echo -e "Add to ${BLUE}~/Library/Application Support/Claude/claude_desktop_config.json${NC}:"
    else
        echo -e "Add to your MCP client configuration:"
    fi

    echo ""
    echo -e '  {
    "mcpServers": {
      "remembrances": {
        "command": "'${BIN_DIR}'/remembrances-mcp"
      }
    }
  }'
    echo ""
    echo -e "${YELLOW}For GPU acceleration (if available):${NC}"
    echo -e "Edit ${BLUE}${CONFIG_DIR}/config.yaml${NC} and set ${BLUE}gguf-gpu-layers${NC} to a positive value"
    echo ""
    echo -e "Documentation: ${BLUE}https://github.com/${REPO}${NC}"
    echo ""
}

# Cleanup function
cleanup() {
    local temp_dir="$1"
    if [ -d "${temp_dir}" ]; then
        rm -rf "${temp_dir}"
    fi
}

# Ask user a yes/no question with default
# Usage: ask_yes_no "prompt" "default" -> sets REPLY to y or n
ask_yes_no() {
    local prompt="$1"
    local default="$2"
    
    if [ "$INTERACTIVE" = "true" ]; then
        read -p "$prompt" -n 1 -r
        echo ""
        if [ -z "$REPLY" ]; then
            REPLY="$default"
        fi
    else
        # Non-interactive mode: use default
        REPLY="$default"
        print_warning "Non-interactive mode: using default ($default) for: $prompt"
    fi
}

ask_choice() {
    # Usage: ask_choice "Prompt" "default" "valid_regex" -> sets REPLY
    local prompt="$1"
    local default="$2"
    local valid_re="$3"

    if [ "$INTERACTIVE" = "true" ]; then
        while true; do
            read -p "$prompt" -r
            if [ -z "$REPLY" ]; then
                REPLY="$default"
            fi
            if echo "$REPLY" | grep -qE "$valid_re"; then
                return 0
            fi
            print_warning "Invalid choice. Please try again."
        done
    else
        REPLY="$default"
        print_warning "Non-interactive mode: using default ($default) for: $prompt"
    fi
}

# Main installation function
main() {
    echo ""
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}                Remembrances-MCP Installer (Wizard)            ${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    
    if [ "$INTERACTIVE" = "false" ]; then
        print_warning "Running in non-interactive mode (piped input detected)"
        print_warning "Using default values for all prompts"
        print_warning "Set environment variables to customize: REMEMBRANCES_VERSION, REMEMBRANCES_NVIDIA, REMEMBRANCES_PORTABLE, REMEMBRANCES_DOWNLOAD_MODEL"
        echo ""
    fi

    NEEDS_LD_LIBRARY_PATH=false

    progress_step "Detecting platform"

    # Detect OS
    local os
    os=$(detect_os)
    print_step "Detected OS: ${os}"

    if [ "${os}" = "unsupported" ]; then
        print_error "Unsupported operating system: $(uname -s)"
        print_error "This installer supports Linux and macOS only."
        exit 1
    fi

    # Detect architecture
    local arch
    arch=$(detect_arch)
    print_step "Detected architecture: ${arch}"

    if [ "${arch}" = "unsupported" ]; then
        print_error "Unsupported architecture: $(uname -m)"
        print_error "This installer supports amd64 and aarch64/arm64 only."
        exit 1
    fi

    # Enforce supported platforms: linux/amd64 or darwin/aarch64
    if [ "${os}" = "darwin" ] && [ "${arch}" != "aarch64" ]; then
        die "Unsupported macOS architecture: $(uname -m). Only Apple Silicon (aarch64/arm64) is supported."
    fi

    if [ "${os}" = "linux" ] && [ "${arch}" != "amd64" ]; then
        die "Unsupported Linux architecture: $(uname -m). Only x86_64 (amd64) is supported."
    fi

    progress_step "Fetching latest release metadata"

    # Fetch release metadata via GitHub API
    local release_json
    release_json=$(get_release_json) || die "Failed to fetch release metadata from GitHub."

    VERSION=$(release_get_tag_name "$release_json")
    if [ -z "${VERSION}" ]; then
        die "Could not determine latest release tag from GitHub API response."
    fi
    print_success "Selected release: ${VERSION}"

    # Show embedded assets found (assistant-style)
    local embedded_list
    embedded_list=$(release_list_asset_urls "$release_json" | grep -E "embedded" | sed 's#.*/##' || true)
    if [ -n "$embedded_list" ]; then
        print_step "Embedded assets available in this release:"
        echo "$embedded_list" | while read -r a; do
            [ -n "$a" ] && echo "  - $a"
        done
    else
        print_warning "No embedded assets detected in the release metadata. Will try best-effort selection."
    fi

    progress_step "Detecting CPU/GPU capabilities"

    local has_nvidia=false
    local cuda_major=""
    local has_avx2=false
    local has_avx512=false

    if [ "${os}" = "linux" ] && [ "${arch}" = "amd64" ]; then
        if check_nvidia; then
            has_nvidia=true
        fi
        cuda_major=$(detect_cuda_major_version)
        if cpu_has_avx2; then
            has_avx2=true
        fi
        if cpu_has_avx512; then
            has_avx512=true
        fi
    fi

    print_step "Detected capabilities:"
    print_kv "NVIDIA GPU" "${has_nvidia}"
    if [ -n "$cuda_major" ]; then
        print_kv "CUDA version" "${cuda_major}.x (detected)"
    else
        print_kv "CUDA version" "unknown/not found"
    fi
    if [ "${os}" = "linux" ]; then
        print_kv "AVX2" "${has_avx2}"
        print_kv "AVX-512" "${has_avx512}"
    fi

    progress_step "Choosing build (wizard)"

    local want_nvidia=false
    local want_portable=false

    # Defaults
    if [ "${os}" = "linux" ] && [ "${has_nvidia}" = "true" ]; then
        want_nvidia=true
    fi
    # Portable selection (Linux x86_64 NVIDIA builds):
    # - If CPU supports an instruction set ABOVE AVX2 (AVX-512), prefer the non-portable build.
    # - Otherwise, prefer portable for broader compatibility.
    if [ "${os}" = "linux" ]; then
        if [ "${has_avx512}" = "true" ]; then
            want_portable=false
        else
            want_portable=true
        fi
    fi

    # Env overrides
    if [ "${REMEMBRANCES_NVIDIA:-}" = "yes" ]; then
        want_nvidia=true
    elif [ "${REMEMBRANCES_NVIDIA:-}" = "no" ]; then
        want_nvidia=false
    fi

    if [ "${REMEMBRANCES_PORTABLE:-}" = "yes" ]; then
        want_portable=true
    elif [ "${REMEMBRANCES_PORTABLE:-}" = "no" ]; then
        want_portable=false
    fi

    # Interactive wizard (Linux only)
    if [ "${INTERACTIVE}" = "true" ] && [ "${os}" = "linux" ] && [ "${arch}" = "amd64" ]; then
        echo ""
        print_step "Installer wizard"
        if [ "${has_nvidia}" = "true" ]; then
            ask_yes_no "Install NVIDIA/CUDA build? [Y/n] " "y"
            if [[ $REPLY =~ ^[Nn]$ ]]; then
                want_nvidia=false
            else
                want_nvidia=true
            fi
        else
            want_nvidia=false
        fi

        if [ "${want_nvidia}" = "true" ]; then
            local default_portable="n"
            if [ "${want_portable}" = "true" ]; then
                default_portable="y"
            fi
            ask_yes_no "Use portable build (recommended unless your CPU supports AVX-512)? [y/N] " "$default_portable"
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                want_portable=true
            else
                want_portable=false
            fi
        fi
    fi

    progress_step "Preparing install directories"

    # Get installation directories
    get_install_dirs "${os}"

    # Choose filename and resolve asset URL
    local filename
    filename=$(choose_release_filename "${os}" "${arch}" "${want_nvidia}" "${want_portable}")
    if [ -z "$filename" ]; then
        die "Could not determine a release filename for this platform."
    fi

    local download_url
    download_url=$(release_find_asset_url "$release_json" "$filename")

    # If CPU embedded isn't available, fall back to non-embedded CPU asset.
    if [ -z "$download_url" ] && [ "${os}" = "linux" ] && [ "${want_nvidia}" != "true" ]; then
        if [ "$filename" = "remembrances-mcp-embedded-cpu-linux-x86_64.zip" ]; then
            print_warning "CPU embedded asset not found. Falling back to standard CPU build."
            filename="remembrances-mcp-cpu-linux-x86_64.zip"
            download_url=$(release_find_asset_url "$release_json" "$filename")
        fi
    fi

    if [ -z "$download_url" ]; then
        die "Could not find download URL for asset: ${filename}"
    fi
    print_step "Selected asset: ${filename}"

    progress_step "Downloading & extracting release"

    # Download and extract
    local extracted_dir
    DOWNLOAD_TEMP_DIR=""
    EXTRACTED_DIR=""
    download_release "${download_url}"
    extracted_dir="${EXTRACTED_DIR}"
    if [ -z "${extracted_dir}" ]; then
        die "Failed to determine extracted release directory."
    fi

    progress_step "Installing files"

    # Install files
    install_files "${extracted_dir}"

    # If Linux + NVIDIA selected, ensure CUDA 12+ runtime libs exist; otherwise install fallback bundle.
    if [ "${os}" = "linux" ] && [ "${want_nvidia}" = "true" ]; then
        local cuda_ok=false

        # 1) Best-effort: check if libllama's CUDA deps are actually resolvable.
        if cuda_llama_deps_resolvable; then
            cuda_ok=true
        # 2) Fallback: check for required CUDA runtime SONAMEs in known locations.
        elif cuda_required_runtime_libs_present_linux; then
            cuda_ok=true
        fi

        if [ "$cuda_ok" = "true" ]; then
            if [ -n "$cuda_major" ]; then
                print_success "CUDA runtime detected (CUDA ${cuda_major}.x and required libs present)."
            else
                print_success "CUDA runtime detected (required libs present)."
            fi
        else
            print_warning "NVIDIA GPU detected but CUDA runtime libraries required by llama were not found in the system loader paths."
            cuda_report_missing_runtime_libs_linux || true
            print_warning "Downloading CUDA runtime bundle and configuring LD_LIBRARY_PATH..."
            ensure_cuda_runtime_libs_linux
        fi
    fi

    progress_step "Creating configuration"

    # Create configuration
    create_config

    progress_step "Optional: downloading GGUF model"

    # Download GGUF model
    echo ""
    local download_model="y"
    if [ "${REMEMBRANCES_DOWNLOAD_MODEL:-}" = "no" ]; then
        download_model="n"
        print_warning "Skipping GGUF model download (from env var)"
    elif [ "${REMEMBRANCES_DOWNLOAD_MODEL:-}" = "yes" ]; then
        download_model="y"
    else
        ask_yes_no "Do you want to download the GGUF embedding model (~260MB)? [Y/n] " "y"
        download_model="$REPLY"
    fi
    
    if [[ ! $download_model =~ ^[Nn]$ ]]; then
        download_gguf_model
    else
        print_warning "Skipping GGUF model download"
        print_warning "You can download it later manually or configure Ollama/OpenAI instead"
    fi

    progress_step "Finalizing shell setup"

    # Setup PATH (+ LD_LIBRARY_PATH if needed)
    setup_path "${os}"

    # Cleanup (only remove the temp dir created by download_release)
    if [ -n "${DOWNLOAD_TEMP_DIR}" ]; then
        cleanup "${DOWNLOAD_TEMP_DIR}"
    fi

    # Print final instructions
    print_instructions "${os}"
}

# Run main unless the script is being sourced.
#
# IMPORTANT:
# - `curl ... | bash` executes the script from stdin, so $0 is typically "bash"
#   and BASH_SOURCE[0] is not a reliable signal.
# - The `(return 0)` trick works across GNU/BSD bash: it only succeeds when the
#   file is sourced (or inside a function). In an `if` condition, a non-zero
#   status does not trigger `set -e`.
if (return 0 2>/dev/null); then
    : # sourced: do not auto-run
else
    main "$@"
fi
