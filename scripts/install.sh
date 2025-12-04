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
#   REMEMBRANCES_VERSION=v1.14.1     - Specific version to install
#   REMEMBRANCES_NVIDIA=yes|no       - Force NVIDIA or CPU-only build
#   REMEMBRANCES_DOWNLOAD_MODEL=yes|no - Download GGUF model or skip
#
# Available builds:
#   - darwin-arm64: macOS with Apple Silicon (M1/M2/M3)
#   - linux-amd64-nvidia: Linux with NVIDIA GPU + modern Intel CPU
#   - linux-amd64-nvidia-portable: Linux with NVIDIA GPU + AMD Ryzen or older Intel
#   - linux-amd64-cpu-only: Linux without NVIDIA GPU

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

# Version to install
VERSION="${REMEMBRANCES_VERSION:-v1.14.1}"
REPO="madeindigio/remembrances-mcp"
GITHUB_RELEASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

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
    if command -v nvidia-smi &> /dev/null; then
        return 0
    fi
    return 1
}

# Check if CPU is modern Intel (not AMD Ryzen or older Intel)
# Returns 0 (true) if modern Intel, 1 (false) if AMD Ryzen or older Intel
check_modern_intel_cpu() {
    local cpu_info
    
    # Get CPU model name
    if [ -f /proc/cpuinfo ]; then
        cpu_info=$(grep -m1 "model name" /proc/cpuinfo 2>/dev/null | cut -d: -f2 | tr '[:upper:]' '[:lower:]')
    else
        return 1  # Cannot determine, use portable version
    fi
    
    # Check for AMD Ryzen - use portable version
    if echo "$cpu_info" | grep -q "amd\|ryzen"; then
        return 1
    fi
    
    # Check for Intel - determine if modern (10th gen or newer)
    if echo "$cpu_info" | grep -q "intel"; then
        # Try to extract generation from Intel Core processors
        # Modern formats: "11th gen intel", "12th gen intel", "13th gen intel", "14th gen intel"
        # Also: Intel Core i7-10xxx, i7-11xxx, etc.
        
        # Check for "Nth gen" format (10th gen and above = modern)
        if echo "$cpu_info" | grep -qE "(1[0-9]th|2[0-9]th)\s*gen"; then
            return 0  # Modern Intel
        fi
        
        # Check for Core iX-NXXXX format where N >= 10
        if echo "$cpu_info" | grep -qE "i[3579]-1[0-9][0-9][0-9][0-9]"; then
            return 0  # Modern Intel (10th gen+)
        fi
        
        # Check for 12th/13th/14th gen patterns (like i7-12700, i9-13900)
        if echo "$cpu_info" | grep -qE "i[3579]-(1[2-9][0-9][0-9][0-9]|[2-9][0-9][0-9][0-9][0-9])"; then
            return 0  # Modern Intel
        fi
        
        # Older Intel - use portable version for better compatibility
        return 1
    fi
    
    # Unknown CPU, use portable version for safety
    return 1
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

# Get the download URL for the release
get_download_url() {
    local os="$1"
    local arch="$2"
    local variant="$3"  # "nvidia", "nvidia-portable", or "cpu-only"

    local filename

    if [ "$os" = "darwin" ]; then
        # macOS - only arm64 supported
        filename="remembrances-mcp-darwin-arm64.zip"
    elif [ "$os" = "linux" ]; then
        case "$variant" in
            "nvidia")
                filename="remembrances-mcp-linux-amd64-nvidia.zip"
                ;;
            "nvidia-portable")
                filename="remembrances-mcp-linux-amd64-nvidia-portable.zip"
                ;;
            *)
                filename="remembrances-mcp-linux-amd64-cpu-only.zip"
                ;;
        esac
    fi

    echo "${GITHUB_RELEASE_URL}/${filename}"
}

# Download and extract release
download_release() {
    local url="$1"
    local temp_dir
    temp_dir="$(mktemp -d)"
    local zip_file="${temp_dir}/release.zip"

    print_step "Downloading release from ${url}..."

    if command -v curl &> /dev/null; then
        curl -fsSL -o "${zip_file}" "${url}"
    elif command -v wget &> /dev/null; then
        wget -q -O "${zip_file}" "${url}"
    else
        print_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi

    print_success "Download complete"

    print_step "Extracting files..."

    if command -v unzip &> /dev/null; then
        unzip -q "${zip_file}" -d "${temp_dir}"
    else
        print_error "unzip command not found. Please install it."
        exit 1
    fi

    # Find the extracted directory (should be named like linux-amd64, darwin-aarch64, etc.)
    local extracted_dir
    extracted_dir=$(find "${temp_dir}" -mindepth 1 -maxdepth 1 -type d | head -n 1)

    if [ -z "${extracted_dir}" ]; then
        print_error "Could not find extracted directory"
        exit 1
    fi

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

    # Copy binary
    if [ -f "${src_dir}/remembrances-mcp" ]; then
        cp "${src_dir}/remembrances-mcp" "${BIN_DIR}/"
        chmod +x "${BIN_DIR}/remembrances-mcp"
        print_success "Binary installed to ${BIN_DIR}/remembrances-mcp"
    else
        print_error "Binary not found in release"
        exit 1
    fi

    # Copy shared libraries to the SAME directory as binary
    # The binary is compiled to look for libraries in its own directory first
    local lib_count=0
    for lib in "${src_dir}"/*.so "${src_dir}"/*.so.* "${src_dir}"/*.dylib; do
        if [ -f "$lib" ]; then
            cp "$lib" "${BIN_DIR}/"
            lib_count=$((lib_count + 1))
        fi
    done

    if [ $lib_count -gt 0 ]; then
        print_success "${lib_count} shared libraries installed to ${BIN_DIR}/"
    fi

    # Copy sample configs for reference
    if [ -f "${src_dir}/config.sample.yaml" ]; then
        cp "${src_dir}/config.sample.yaml" "${CONFIG_DIR}/"
    fi
    if [ -f "${src_dir}/config.sample.gguf.yaml" ]; then
        cp "${src_dir}/config.sample.gguf.yaml" "${CONFIG_DIR}/"
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
    echo -e "Installation directory: ${BLUE}${INSTALL_DIR}${NC}"
    echo -e "Binary & libraries:     ${BLUE}${BIN_DIR}/${NC}"
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

# Main installation function
main() {
    echo ""
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}              Remembrances-MCP Installer ${VERSION}              ${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
    echo ""
    
    if [ "$INTERACTIVE" = "false" ]; then
        print_warning "Running in non-interactive mode (piped input detected)"
        print_warning "Using default values for all prompts"
        print_warning "Set environment variables to customize: REMEMBRANCES_VERSION, REMEMBRANCES_NVIDIA=yes/no, REMEMBRANCES_DOWNLOAD_MODEL=yes/no"
        echo ""
    fi

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

    # Check for macOS x86_64 (not supported in releases)
    if [ "${os}" = "darwin" ] && [ "${arch}" = "amd64" ]; then
        print_error "macOS Intel (x86_64) binaries are not available in the current release."
        print_error "Only macOS ARM64 (M1/M2/M3) is supported."
        print_error "Please compile from source or use a Mac with Apple Silicon."
        exit 1
    fi

    # Check for Linux ARM (not supported in releases)
    if [ "${os}" = "linux" ] && [ "${arch}" = "aarch64" ]; then
        print_error "Linux ARM64 binaries are not available in the current release."
        print_error "Please compile from source or use a different architecture."
        exit 1
    fi

    # Determine variant for Linux
    local variant="cpu-only"
    if [ "${os}" = "linux" ] && [ "${arch}" = "amd64" ]; then
        # Check environment variable first
        if [ "${REMEMBRANCES_NVIDIA:-}" = "yes" ]; then
            if check_modern_intel_cpu; then
                variant="nvidia"
                print_success "Using NVIDIA build (modern Intel CPU detected, from env var)"
            else
                variant="nvidia-portable"
                print_success "Using NVIDIA portable build (AMD Ryzen/older Intel, from env var)"
            fi
        elif [ "${REMEMBRANCES_NVIDIA:-}" = "no" ]; then
            variant="cpu-only"
            print_success "Using CPU-only build (from env var)"
        elif check_nvidia; then
            print_success "NVIDIA GPU detected"
            
            # Detect CPU type
            local cpu_info=""
            if [ -f /proc/cpuinfo ]; then
                cpu_info=$(grep -m1 "model name" /proc/cpuinfo 2>/dev/null | cut -d: -f2)
            fi
            if [ -n "$cpu_info" ]; then
                print_step "Detected CPU:$cpu_info"
            fi
            
            echo ""
            ask_yes_no "Do you want to install the NVIDIA/CUDA optimized version? [Y/n] " "y"
            if [[ ! $REPLY =~ ^[Nn]$ ]]; then
                # Determine which NVIDIA variant based on CPU
                if check_modern_intel_cpu; then
                    variant="nvidia"
                    print_success "Using NVIDIA build (optimized for modern Intel CPUs)"
                else
                    variant="nvidia-portable"
                    print_success "Using NVIDIA portable build (compatible with AMD Ryzen and older Intel)"
                fi
            else
                variant="cpu-only"
                print_success "Using CPU-only build"
            fi
        else
            print_step "No NVIDIA GPU detected, using CPU-only build"
            variant="cpu-only"
        fi
    fi

    # Get installation directories
    get_install_dirs "${os}"

    # Get download URL
    local download_url
    download_url=$(get_download_url "${os}" "${arch}" "${variant}")
    print_step "Download URL: ${download_url}"

    # Download and extract
    local extracted_dir
    extracted_dir=$(download_release "${download_url}")

    # Install files
    install_files "${extracted_dir}"

    # Create configuration
    create_config

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

    # Setup PATH
    setup_path "${os}"

    # Cleanup
    cleanup "$(dirname "${extracted_dir}")"

    # Print final instructions
    print_instructions "${os}"
}

# Run main
main "$@"
