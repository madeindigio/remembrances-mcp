# Intel OpenVINO Build Guide

Compile and run **remembrances-mcp** with hardware-accelerated inference on Intel GPUs and NPUs using the [Intel OpenVINO](https://github.com/openvinotoolkit/openvino) backend for llama.cpp/ggml.

## Supported Hardware

| Device class | Examples | Support status |
|---|---|---|
| Intel iGPU (Xe-LPG / Arc) | Arrow Lake-U, Meteor Lake, Raptor Lake GT2 | ✅ Production |
| Intel Arc discrete GPU | Arc A770, Arc B580 | ✅ Production |
| Intel NPU (AI Boost) | Arrow Lake NPU (`/dev/accel/accel0`), Meteor Lake NPU | ✅ Supported (experimental) |
| Intel CPU (fallback) | Any x86-64 with AVX2 | ✅ Always available |

This repository was developed on a system with:
- **CPU/iGPU**: Intel Arrow Lake-U `00:02.0 Intel Corporation Arrow Lake-U [Intel Graphics]`
- **NPU**: Intel Arrow Lake NPU `00:0b.0 Processing accelerators: Intel Corporation Arrow Lake NPU`
- **Kernel**: Linux 6.18 with `i915`, `xe`, and `intel_vpu` modules loaded
- **NPU device**: `/dev/accel/accel0`

---

## Architecture Overview

The OpenVINO backend in llama.cpp/ggml:
- CMake flag: `-DGGML_OPENVINO=ON`
- Produces: `libggml-openvino.so` alongside `libggml.so` / `libllama.so`
- Requires: OpenVINO SDK (C++ headers + CMake config + runtime libraries)
- Runtime device selection: `CPU`, `GPU`, `NPU` (set via env or build flag)
- Falls back to CPU when GPU/NPU is unavailable

---

## Prerequisites

### 1. OpenVINO SDK

**Option A – APT from Intel (recommended for production)**

```bash
# Add Intel APT repository
curl -fsSL https://apt.repos.intel.com/intel-gpg-keys/GPG-PUB-KEY-INTEL-SW-PRODUCTS.PUB \
  | sudo gpg --dearmor -o /usr/share/keyrings/intel-sw-products.gpg

echo "deb [signed-by=/usr/share/keyrings/intel-sw-products.gpg] \
  https://apt.repos.intel.com/openvino/2025 ubuntu24 main" \
  | sudo tee /etc/apt/sources.list.d/intel-openvino.list

sudo apt-get update
sudo apt-get install -y openvino openvino-dev
# SDK is installed at /opt/intel/openvino/
# CMake config: /opt/intel/openvino/runtime/cmake/
export OPENVINO_DIR=/opt/intel/openvino/runtime/cmake
```

**Option B – PyPI wheel extraction (no sudo, already done on this machine)**

```bash
# Already installed at ~/intel/openvino_sdk/openvino/cmake
# Version: 2026.1.0
export OPENVINO_DIR=~/intel/openvino_sdk/openvino/cmake
```

This is the wheel-based SDK installed by the project. The wheel was downloaded from PyPI
(`openvino-2026.1.0`) and extracted to `~/intel/openvino_sdk/`.

**Note on wheel-based SDK**: The PyPI wheel ships C++ headers, CMake config, and runtime
libraries. A compatibility symlink for the TBB cmake path was created at
`~/intel/openvino_sdk/openvino/3rdparty/tbb/lib/cmake/TBB/` to match what the
llama.cpp ggml-openvino CMakeLists.txt expects.

### 2. Intel GPU Compute Runtime (for GPU execution)

To run models on the Intel iGPU or Arc GPU, the Intel compute runtime (OpenCL/Level Zero)
must be installed. **This requires sudo.**

```bash
# Add Intel graphics repository
curl -fsSL https://repositories.intel.com/graphics/intel-graphics.key \
  | sudo gpg --dearmor -o /usr/share/keyrings/intel-graphics-archive-keyring.gpg

echo "deb [arch=amd64 signed-by=/usr/share/keyrings/intel-graphics-archive-keyring.gpg] \
  https://repositories.intel.com/graphics/ubuntu noble client" \
  | sudo tee /etc/apt/sources.list.d/intel-graphics.list

sudo apt-get update
sudo apt-get install -y \
  intel-opencl-icd \
  intel-level-zero-gpu \
  level-zero \
  libze-intel-gpu1 \
  intel-media-va-driver-non-free

# Add user to render group (required for GPU access without root)
sudo usermod -aG render $USER
sudo usermod -aG video $USER
# Log out and back in (or: newgrp render)
```

Verify GPU is accessible:
```bash
ls -la /dev/dri/           # Should show renderD128, card0, card1
clinfo --list              # Shows OpenCL platforms (after intel-opencl-icd install)
```

### 3. Intel NPU Driver (for NPU execution, Arrow Lake / Meteor Lake)

The NPU kernel module `intel_vpu` is already loaded on this system. For the user-space
driver (needed for OpenVINO NPU plugin):

```bash
# Check if NPU device is accessible
ls -la /dev/accel/accel0   # Should exist on Arrow Lake

# Add user to accel group (if it exists)
sudo usermod -aG accel $USER 2>/dev/null || true

# Install Intel NPU driver package (Ubuntu 24.04)
# Check https://github.com/intel/linux-npu-driver/releases for latest
NPU_DRIVER_URL="https://github.com/intel/linux-npu-driver/releases/download/v1.10.0/intel-driver-compiler-npu_1.10.0.20241107_ubuntu24.04_amd64.deb"
curl -L "$NPU_DRIVER_URL" -o /tmp/intel-npu-driver.deb
sudo dpkg -i /tmp/intel-npu-driver.deb
```

### 4. Build Tools

```bash
# Required (already installed on this system)
cmake --version   # >= 3.14 required; 3.28.3 installed
gcc --version     # >= 9.0
g++ --version
git --version
```

---

## Build Instructions

### Quick Start

```bash
# Set OpenVINO SDK path (wheel-based, already installed on this machine)
export OPENVINO_DIR=~/intel/openvino_sdk/openvino/cmake

# Build OpenVINO variant libraries
make build-libs-openvino

# Build and run with OpenVINO
BUILD_TYPE=openvino make build
```

### Using the Build Script Directly

```bash
# Auto-detects OPENVINO_DIR from ~/intel/openvino_sdk/openvino/cmake
./scripts/build-variant-libs.sh openvino

# With custom SDK path
OPENVINO_DIR=/opt/intel/openvino/runtime/cmake ./scripts/build-variant-libs.sh openvino

# Target specific device
OPENVINO_DEVICE=GPU ./scripts/build-variant-libs.sh openvino
OPENVINO_DEVICE=NPU ./scripts/build-variant-libs.sh openvino
```

### Build Variant Binary

```bash
export OPENVINO_DIR=~/intel/openvino_sdk/openvino/cmake

# Build libraries + binary
make build-variant VARIANT=openvino

# Package for distribution
make dist-variant VARIANT=openvino
```

### Embedded Build (single binary)

```bash
export OPENVINO_DIR=~/intel/openvino_sdk/openvino/cmake

make build-embedded-openvino

# Package embedded variant
make dist-embedded-variant EMBEDDED_VARIANT=openvino
```

---

## CMake Flags Reference

| CMake flag | Value | Description |
|---|---|---|
| `GGML_OPENVINO` | `ON` | Enable OpenVINO backend |
| `OpenVINO_DIR` | path to cmake dir | Where `OpenVINOConfig.cmake` lives |
| `GGML_OPENVINO_DEVICE` | `CPU`/`GPU`/`NPU` | Default device (optional; can set at runtime) |

Full cmake command example:
```bash
cd $GO_LLAMA_DIR
cmake -B build llama.cpp \
  -DLLAMA_STATIC=OFF \
  -DGGML_OPENVINO=ON \
  -DOpenVINO_DIR="$HOME/intel/openvino_sdk/openvino/cmake" \
  -DCMAKE_BUILD_TYPE=Release
cmake --build build --config Release -j$(nproc)
```

---

## Runtime Configuration

### Device Selection

At runtime, OpenVINO selects the device based on:
1. Environment variable `GGML_OPENVINO_DEVICE` (or model-level setting)
2. Or the default device set at build time
3. Or automatic selection (prefers GPU > CPU)

```bash
# Use Intel iGPU
GGML_OPENVINO_DEVICE=GPU ./build/remembrances-mcp --gguf-model-path model.gguf

# Use Intel NPU (Arrow Lake AI Boost)
GGML_OPENVINO_DEVICE=NPU ./build/remembrances-mcp --gguf-model-path model.gguf

# CPU fallback (always works)
GGML_OPENVINO_DEVICE=CPU ./build/remembrances-mcp --gguf-model-path model.gguf
```

### LD_LIBRARY_PATH for Wheel-Based SDK

When using the PyPI wheel SDK, OpenVINO runtime libs are not in the system lib path.
Set `LD_LIBRARY_PATH` before running:

```bash
export LD_LIBRARY_PATH=~/intel/openvino_sdk/openvino/libs:$LD_LIBRARY_PATH
./build/remembrances-mcp --gguf-model-path model.gguf
```

Or add to `run-remembrances.sh`:
```bash
# Add to run-remembrances.sh:
export LD_LIBRARY_PATH="$HOME/intel/openvino_sdk/openvino/libs:${LD_LIBRARY_PATH:-}"
```

### System SDK (APT install) - No LD_LIBRARY_PATH needed

If installed via the Intel APT repo, the libraries are in `/opt/intel/openvino/runtime/lib/intel64/`
and a system `ldconfig` entry may be created automatically.

---

## Installation State on This Machine

```
Hardware:
  iGPU  : Intel Arrow Lake-U [Intel Graphics] (00:02.0)
  NPU   : Intel Arrow Lake NPU (00:0b.0) → /dev/accel/accel0
  Kernel: Linux 6.18, modules: i915 + xe + intel_vpu (loaded)

Software installed:
  OpenVINO SDK : ~/intel/openvino_sdk/openvino/cmake (v2026.1.0, from PyPI wheel)
  OpenCL ICD   : /etc/OpenCL/vendors/nvidia.icd only (Intel ICD missing → install intel-opencl-icd)
  Level Zero   : NOT installed (install intel-level-zero-gpu)
  NPU driver   : kernel module present; user-space driver NOT installed

Required (need sudo) to complete GPU/NPU enablement:
  sudo apt-get install intel-opencl-icd intel-level-zero-gpu level-zero
  sudo usermod -aG render $USER

Build status:
  GGML_OPENVINO cmake flag: ✅ supported in current llama.cpp HEAD
  OpenVINO_DIR auto-detect: ✅ ~/intel/openvino_sdk/openvino/cmake
  TBB compatibility links : ✅ ~/intel/openvino_sdk/openvino/3rdparty/tbb/lib/cmake/TBB/
  libOpenCL.so.1          : ✅ /usr/lib/x86_64-linux-gnu/libOpenCL.so.1 (present)
```

---

## Upgrading llama.cpp for OpenVINO Support

The `local-go-llama/Makefile` and build scripts reference the llama.cpp submodule at
`GO_LLAMA_DIR` (default: `$HOME/www/MCP/Remembrances/go-llama.cpp`). If the current
submodule version does not contain `ggml/src/ggml-openvino/`, update it:

```bash
cd $GO_LLAMA_DIR
git submodule update --init --recursive

# Check if GGML_OPENVINO is supported
grep -r "GGML_OPENVINO" llama.cpp/ggml/CMakeLists.txt
# Should show: option(GGML_OPENVINO "ggml: use OPENVINO" OFF)

# If not present, the llama.cpp submodule needs updating
cd llama.cpp
git fetch origin
git checkout origin/master   # or a specific tag
cd ..
```

As of llama.cpp HEAD (checked 2026-04), `GGML_OPENVINO=ON` is fully supported in
`ggml/CMakeLists.txt` and the backend lives at `ggml/src/ggml-openvino/`.

---

## Troubleshooting

### `find_package(OpenVINO REQUIRED)` fails

```
CMake Error: Could not find a package configuration file provided by "OpenVINO"
```

Set `OPENVINO_DIR` explicitly:
```bash
export OPENVINO_DIR=~/intel/openvino_sdk/openvino/cmake
# or
cmake ... -DOpenVINO_DIR=~/intel/openvino_sdk/openvino/cmake
```

### TBB not found during cmake

```
include could not find load file: .../3rdparty/tbb/lib/cmake/TBB/TBBConfig.cmake
```

The compatibility symlink may be missing. Recreate it:
```bash
OPENVINO_BASE=~/intel/openvino_sdk/openvino
mkdir -p "$OPENVINO_BASE/3rdparty/tbb/lib/cmake/TBB"
ln -sf "$OPENVINO_BASE/cmake/TBBConfig.cmake" "$OPENVINO_BASE/3rdparty/tbb/lib/cmake/TBB/TBBConfig.cmake"
ln -sf "$OPENVINO_BASE/cmake/TBBTargets.cmake" "$OPENVINO_BASE/3rdparty/tbb/lib/cmake/TBB/TBBTargets.cmake"
ln -sf "$OPENVINO_BASE/cmake/TBBConfigVersion.cmake" "$OPENVINO_BASE/3rdparty/tbb/lib/cmake/TBB/TBBConfigVersion.cmake"
```

### `libopenvino.so` not found at runtime

```
error while loading shared libraries: libopenvino.so.2610: cannot open shared object
```

Set `LD_LIBRARY_PATH`:
```bash
export LD_LIBRARY_PATH=~/intel/openvino_sdk/openvino/libs:$LD_LIBRARY_PATH
```

Or create a system ldconfig entry (requires sudo):
```bash
echo "$HOME/intel/openvino_sdk/openvino/libs" | sudo tee /etc/ld.so.conf.d/openvino-sdk.conf
sudo ldconfig
```

### Intel GPU not detected / OpenCL not available

```bash
# Check OpenCL ICDs
ls /etc/OpenCL/vendors/      # Should include intel.icd after intel-opencl-icd install

# If only nvidia.icd is present, install Intel ICD:
sudo apt-get install intel-opencl-icd
```

### NPU: device not accessible (`/dev/accel/accel0` permission denied)

```bash
# Check device permissions
ls -la /dev/accel/accel0
# drwxrwxr-x ... accel accel ...

# Add user to accel group
sudo usermod -aG accel $USER
newgrp accel
```

---

## Performance Notes

- **Intel Arc A770 / B580**: Full GPU offload, similar perf to NVIDIA RTX 3060 range
- **Arrow Lake iGPU (Xe-LPG)**: Limited VRAM (~8-16GB shared system RAM), good for 7B models at Q4
- **Arrow Lake NPU**: Best for INT4/INT8 quantized models; memory-limited; latency-optimized
- **CPU fallback**: Uses AVX-512 / AMX instructions where available (Arrow Lake has AVX-512)

---

## Related Documentation

- [MULTI_VARIANT_BUILD.md](MULTI_VARIANT_BUILD.md) – Overview of all variants
- [GPU_COMPILATION.md](GPU_COMPILATION.md) – CUDA / ROCm build details
- [Intel OpenVINO GitHub](https://github.com/openvinotoolkit/openvino)
- [llama.cpp ggml-openvino source](https://github.com/ggerganov/llama.cpp/tree/master/ggml/src/ggml-openvino)
