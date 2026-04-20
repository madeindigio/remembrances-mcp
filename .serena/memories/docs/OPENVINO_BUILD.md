# Intel OpenVINO Build Guide

Compile and run **remembrances-mcp** with hardware-accelerated inference on Intel GPUs and NPUs using the Intel OpenVINO backend for llama.cpp/ggml.

## Supported Hardware

- Intel iGPU (Xe-LPG / Arc): Arrow Lake-U, Meteor Lake, Raptor Lake GT2 — Production
- Intel Arc discrete GPU: Arc A770, Arc B580 — Production
- Intel NPU (AI Boost): Arrow Lake NPU (/dev/accel/accel0), Meteor Lake NPU — Supported (experimental)
- Intel CPU (fallback): Any x86-64 with AVX2 — Always available

System: Intel Arrow Lake-U iGPU + Arrow Lake NPU at /dev/accel/accel0, kernel 6.18 with i915, xe, intel_vpu modules.

## Architecture

CMake flag: -DGGML_OPENVINO=ON
Produces: libggml-openvino.so alongside libggml.so / libllama.so
Requires: OpenVINO SDK (C++ headers + CMake config + runtime libraries)
Runtime device: CPU, GPU, or NPU (set via GGML_OPENVINO_DEVICE env var)

## OpenVINO SDK

Option A - APT (recommended):
  export OPENVINO_DIR=/opt/intel/openvino/runtime/cmake

Option B - PyPI wheel (installed on this machine, no sudo):
  export OPENVINO_DIR=~/intel/openvino_sdk/openvino/cmake
  # Version: 2026.1.0 at ~/intel/openvino_sdk/

TBB compatibility: symlinks at ~/intel/openvino_sdk/openvino/3rdparty/tbb/lib/cmake/TBB/
point to ~/intel/openvino_sdk/openvino/cmake/TBB*.cmake

## Build Commands

  export OPENVINO_DIR=~/intel/openvino_sdk/openvino/cmake
  make build-libs-openvino
  make build-embedded-openvino
  make dist-embedded-variant EMBEDDED_VARIANT=openvino
  OPENVINO_DEVICE=GPU ./scripts/build-variant-libs.sh openvino

## CMake flags
  -DGGML_OPENVINO=ON
  -DOpenVINO_DIR=<path-to-cmake-dir>
  -DGGML_OPENVINO_DEVICE=CPU|GPU|NPU

## Runtime
  export LD_LIBRARY_PATH=~/intel/openvino_sdk/openvino/libs:$LD_LIBRARY_PATH
  GGML_OPENVINO_DEVICE=GPU ./build/remembrances-mcp

## Hardware state (this machine)
  iGPU: Intel Arrow Lake-U [Intel Graphics] (00:02.0)
  NPU: Intel Arrow Lake NPU (00:0b.0) -> /dev/accel/accel0
  OpenVINO SDK installed: YES (wheel)
  Intel GPU OpenCL ICD: NOT installed (need: sudo apt-get install intel-opencl-icd)
  Level Zero: NOT installed (need: sudo apt-get install intel-level-zero-gpu level-zero)
  NPU user-space driver: NOT installed (need kernel module + linux-npu-driver package)

## sudo commands needed to complete GPU/NPU setup:
  sudo apt-get install intel-opencl-icd intel-level-zero-gpu level-zero
  sudo usermod -aG render $USER

## Files changed in repo:
  local-go-llama/Makefile - BUILD_TYPE=openvino block added
  scripts/build-variant-libs.sh - openvino variant + GO_LLAMA_DIR path fix
  Makefile (top) - build-libs-openvino, prepare-embedded-libs-openvino, build-embedded-openvino targets
  docs/OPENVINO_BUILD.md - this guide
  docs/MULTI_VARIANT_BUILD.md - openvino row added to variant table
