# Windows Build Implementation Guide

**Date**: October 24, 2024  
**Estimated Time**: 30-60 minutes  
**Difficulty**: Easy  
**Risk Level**: Low

---

## Overview

This guide provides step-by-step instructions to enable Windows builds for Remembrances-MCP using POSIX-threaded MinGW compilers.

**Solution**: Use MinGW POSIX variant compilers that fully support C++11/14 threading.

---

## Prerequisites

- Working Linux/macOS builds (already functioning)
- Docker with goreleaser-cross:v1.21 image
- Git repository with write access

---

## Implementation Steps

### Step 1: Update Build Script (5 minutes)

**File**: `go-llama.cpp/scripts/build-static-multi.sh`

Find the Windows section (around line 119) and make these changes:

**BEFORE:**
```bash
"windows")
    case "$ARCH" in
        "amd64")
            CC="x86_64-w64-mingw32-gcc"
            CXX="x86_64-w64-mingw32-g++"
            AR="x86_64-w64-mingw32-ar"
            CMAKE_SYSTEM_PROCESSOR="x86_64"
            CFLAGS="-O3 -DNDEBUG -std=c11 -fPIC -march=x86-64 -mtune=generic"
            CXXFLAGS="-O3 -DNDEBUG -std=c++14 -fPIC -march=x86-64 -mtune=generic"
            LDFLAGS="-static -static-libgcc -static-libstdc++"
            # Disable threading for Windows to avoid MinGW pthread issues
            DISABLE_THREADING="ON"
            ;;
```

**AFTER:**
```bash
"windows")
    case "$ARCH" in
        "amd64")
            # Use POSIX-threaded MinGW for full C++11/14 threading support
            CC="x86_64-w64-mingw32-gcc-posix"
            CXX="x86_64-w64-mingw32-g++-posix"
            AR="x86_64-w64-mingw32-gcc-ar-posix"
            CMAKE_SYSTEM_PROCESSOR="x86_64"
            CFLAGS="-O3 -DNDEBUG -std=c11 -fPIC -march=x86-64 -mtune=generic"
            CXXFLAGS="-O3 -DNDEBUG -std=c++14 -fPIC -march=x86-64 -mtune=generic"
            LDFLAGS="-static -pthread -static-libgcc -static-libstdc++"
            ;;
```

**Changes Made**:
- ✅ Changed `CC` from `gcc` to `gcc-posix`
- ✅ Changed `CXX` from `g++` to `g++-posix`
- ✅ Changed `AR` from `ar` to `gcc-ar-posix`
- ✅ Added `-pthread` to LDFLAGS
- ✅ Removed `DISABLE_THREADING="ON"` line

---

### Step 2: Remove Threading Disable Logic (2 minutes)

**File**: `go-llama.cpp/scripts/build-static-multi.sh`

Find and **REMOVE** this block (around line 177):

```bash
# Disable threading for Windows builds
if [ "$DISABLE_THREADING" = "ON" ]; then
    CMAKE_ARGS+=("-DLLAMA_THREADS=OFF")
    CMAKE_ARGS+=("-DCMAKE_CXX_FLAGS=-DGGML_USE_OPENMP=0")
fi
```

**Delete the entire block** - it's no longer needed.

---

### Step 3: Re-enable Windows in GoReleaser (5 minutes)

**File**: `.goreleaser-multiplatform.yml`

#### 3.1: Enable Windows Library Build

Find the before hooks section (around line 15) and uncomment:

**BEFORE:**
```yaml
    - bash -c "cd go-llama.cpp && ./scripts/build-static-multi.sh darwin arm64"
    # TODO: Fix MinGW threading issues before enabling Windows builds
    # - bash -c "cd go-llama.cpp && ./scripts/build-static-multi.sh windows amd64"
    # Ensure go.mod is tidy
    - go mod tidy
```

**AFTER:**
```yaml
    - bash -c "cd go-llama.cpp && ./scripts/build-static-multi.sh darwin arm64"
    - bash -c "cd go-llama.cpp && ./scripts/build-static-multi.sh windows amd64"
    # Ensure go.mod is tidy
    - go mod tidy
```

#### 3.2: Enable Windows Binary Build

Find the Windows build section (around line 115) and uncomment:

**BEFORE:**
```yaml
  # Windows AMD64 - DISABLED: MinGW threading issues with llama.cpp
  # TODO: Re-enable after fixing threading support or upgrading to newer llama.cpp
  # - id: windows-amd64
  #   main: ./cmd/remembrances-mcp
  #   binary: remembrances-mcp
  #   goos:
  #     - windows
  #   goarch:
  #     - amd64
  #   env:
  #     - CGO_ENABLED=1
  #     - CC=x86_64-w64-mingw32-gcc
  #     - CXX=x86_64-w64-mingw32-g++
  #     - CGO_CFLAGS=-I./go-llama.cpp/llama.cpp -I./go-llama.cpp
  #     - CGO_CXXFLAGS=-I./go-llama.cpp/llama.cpp -I./go-llama.cpp -I./go-llama.cpp/llama.cpp/common -I./go-llama.cpp/common
  #     - CGO_LDFLAGS=-L./go-llama.cpp -lbinding-windows-amd64 -lm -lstdc++ -static
  #   ldflags:
  #     - -s
  #     - -w
  #     - -X github.com/madeindigio/remembrances-mcp/pkg/version.CommitHash={{.Commit}}
  #     - -X github.com/madeindigio/remembrances-mcp/pkg/version.Version={{.Version}}
  #     - -extldflags=-static
  #   flags:
  #     - -trimpath
  #     - -buildmode=exe
```

**AFTER:**
```yaml
  # Windows AMD64
  - id: windows-amd64
    main: ./cmd/remembrances-mcp
    binary: remembrances-mcp
    goos:
      - windows
    goarch:
      - amd64
    env:
      - CGO_ENABLED=1
      - CC=x86_64-w64-mingw32-gcc
      - CXX=x86_64-w64-mingw32-g++
      - CGO_CFLAGS=-I./go-llama.cpp/llama.cpp -I./go-llama.cpp
      - CGO_CXXFLAGS=-I./go-llama.cpp/llama.cpp -I./go-llama.cpp -I./go-llama.cpp/llama.cpp/common -I./go-llama.cpp/common
      - CGO_LDFLAGS=-L./go-llama.cpp -lbinding-windows-amd64 -lm -lstdc++ -static
    ldflags:
      - -s
      - -w
      - -X github.com/madeindigio/remembrances-mcp/pkg/version.CommitHash={{.Commit}}
      - -X github.com/madeindigio/remembrances-mcp/pkg/version.Version={{.Version}}
      - -extldflags=-static
    flags:
      - -trimpath
      - -buildmode=exe
```

**Note**: The `CC` and `CXX` here can remain as `x86_64-w64-mingw32-gcc` (without `-posix`) because these are symlinks that will be overridden by the library build using the correct POSIX compilers.

#### 3.3: Enable Windows in UPX Section

Find the UPX section (around line 145) and add windows back:

**BEFORE:**
```yaml
upx:
  - enabled: true
    ids:
      - linux-amd64
      - linux-arm64
      # - windows-amd64  # Disabled until Windows build is fixed
    goos:
      - linux
      # - windows  # Disabled until Windows build is fixed
    goarch:
      - amd64
      - arm64
```

**AFTER:**
```yaml
upx:
  - enabled: true
    ids:
      - linux-amd64
      - linux-arm64
      - windows-amd64
    goos:
      - linux
      - windows
    goarch:
      - amd64
      - arm64
```

---

### Step 4: Test Windows Library Build (5 minutes)

Before doing a full build, test just the Windows library:

```bash
# Navigate to go-llama.cpp
cd go-llama.cpp

# Make script executable
chmod +x scripts/build-static-multi.sh

# Test Windows library build
./scripts/build-static-multi.sh windows amd64

# Expected output:
# ==========================================
# Building llama.cpp static library
# Platform: windows-amd64
# ==========================================
# Compiler: x86_64-w64-mingw32-gcc-posix / x86_64-w64-mingw32-g++-posix
# ...
# ✓ Successfully created: libbinding-windows-amd64.a

# Verify library was created
ls -lh libbinding-windows-amd64.a

# Go back to project root
cd ..
```

**Expected Result**: Library file `libbinding-windows-amd64.a` created successfully (~15-20MB).

**If it fails**: Check error messages carefully and verify compiler names are correct.

---

### Step 5: Full Multiplatform Build (15-20 minutes)

Now test the complete build with all platforms:

```bash
# Clean previous builds
make clean

# Run multiplatform build
make release-multi-snapshot

# Expected output:
# ✓ Mode: snapshot
# ✓ All requirements satisfied
# ...
# ✓ Build completed successfully!
```

**This will take ~3-5 minutes** to compile all platforms.

---

### Step 6: Verify Build Artifacts (2 minutes)

Check that all 5 platforms were built:

```bash
# List generated artifacts
ls -lh dist/outputs/dist/

# Expected files:
# remembrances-mcp_*_Linux_x86_64.tar.gz
# remembrances-mcp_*_Linux_arm64.tar.gz
# remembrances-mcp_*_Darwin_x86_64.tar.gz
# remembrances-mcp_*_Darwin_arm64.tar.gz
# remembrances-mcp_*_Windows_x86_64.zip     ← NEW!
# checksums.txt
```

**Verify Windows binary**:

```bash
# Extract Windows archive
unzip -l dist/outputs/dist/*Windows_x86_64.zip

# Should contain:
# remembrances-mcp.exe
# README.md
# LICENSE
# CHANGELOG.md
```

---

### Step 7: Test Windows Binary (10 minutes)

#### Option A: On Windows Machine

Transfer the binary to a Windows machine and test:

```powershell
# Extract the zip
Expand-Archive remembrances-mcp_*_Windows_x86_64.zip

# Navigate to extracted folder
cd remembrances-mcp_*_Windows_x86_64

# Test version
.\remembrances-mcp.exe --version

# Test help
.\remembrances-mcp.exe --help

# Expected output:
# remembrances-mcp version 0.30.3-SNAPSHOT-xxxxx
```

#### Option B: Using Wine (Linux)

```bash
# Install wine if needed
sudo apt-get install wine64

# Run Windows binary
wine dist/outputs/dist/remembrances-mcp_*_Windows_x86_64/remembrances-mcp.exe --version
```

---

### Step 8: Update Documentation (5 minutes)

**File**: `docs/MULTIPLATFORM_BUILD_FIXES.md`

Update the results table:

```diff
 | Platform | Architecture | Binary Size | Status |
 |----------|-------------|-------------|---------|
 | Linux | x86_64 (amd64) | 3.8M | ✅ Working |
 | Linux | ARM64 | 3.6M | ✅ Working |
 | macOS | x86_64 (Intel) | 3.9M | ✅ Working |
 | macOS | ARM64 (Apple Silicon) | 3.6M | ✅ Working |
-| Windows | x86_64 (amd64) | - | ⚠️ Disabled |
+| Windows | x86_64 (amd64) | 4.2M | ✅ Working |
```

**File**: `docs/BUILD_FIXES_SUMMARY.md`

Update the summary:

```diff
-### ⚠️ Disabled Platform
-
-- **Windows x86_64**: MinGW threading issues with llama.cpp (needs investigation)
+### ✅ All Platforms Working
+
+All 5 target platforms are now fully functional!
```

**File**: `CHANGELOG.md`

Add entry:

```markdown
### Fixed

* **build**: Enabled Windows x86_64 builds using POSIX-threaded MinGW compilers
  - Switched to x86_64-w64-mingw32-gcc-posix for full C++11 threading support
  - All 5 platforms now building successfully
  - Windows binaries are statically linked with no DLL dependencies
```

---

## Verification Checklist

### ✅ Build Verification

- [ ] Windows library compiles without errors
- [ ] Windows binary links successfully
- [ ] All 5 platforms build in single command
- [ ] No new warnings or errors
- [ ] Build time is acceptable (~3-5 minutes)

### ✅ Binary Verification

- [ ] Windows ZIP archive created
- [ ] Binary size is reasonable (~4-5MB)
- [ ] Archive contains all expected files
- [ ] Checksums file updated

### ✅ Runtime Verification (Windows)

- [ ] Binary runs without DLL errors
- [ ] `--version` flag works
- [ ] `--help` flag works
- [ ] Can connect to SurrealDB
- [ ] Threading works correctly
- [ ] No crashes or hangs

### ✅ Regression Testing

- [ ] Linux amd64 still works
- [ ] Linux arm64 still works
- [ ] macOS amd64 still works
- [ ] macOS arm64 still works
- [ ] No performance degradation

---

## Troubleshooting

### Issue: "command not found: x86_64-w64-mingw32-gcc-posix"

**Solution**: The Docker image should have these compilers. Verify:

```bash
docker run --rm --network=none --entrypoint="" \
  ghcr.io/goreleaser/goreleaser-cross:v1.21 \
  which x86_64-w64-mingw32-gcc-posix
```

If not found, the Docker image may be outdated. Update it:

```bash
docker pull ghcr.io/goreleaser/goreleaser-cross:v1.21
```

### Issue: "undefined reference to pthread_create"

**Solution**: Ensure `-pthread` is in LDFLAGS:

```bash
LDFLAGS="-static -pthread -static-libgcc -static-libstdc++"
```

### Issue: Windows binary requires DLLs

**Solution**: Verify static linking flags are correct:

```bash
# Check binary dependencies (on Windows)
dumpbin /dependents remembrances-mcp.exe

# Or using objdump (Linux)
x86_64-w64-mingw32-objdump -p remembrances-mcp.exe | grep "DLL Name"
```

Should only show system DLLs (KERNEL32.dll, etc.), no custom DLLs.

### Issue: Build succeeds but binary crashes on Windows

**Solution**: Test with verbose logging:

```powershell
.\remembrances-mcp.exe --log-level debug --version
```

Check for specific error messages.

### Issue: Threading doesn't work

**Solution**: Verify POSIX compilers were used:

```bash
# Check build log for compiler version
# Should show: x86_64-w64-mingw32-g++-posix (GCC) 10-posix
```

---

## Rollback Procedure

If Windows build causes issues:

```bash
# 1. Revert build script
git checkout go-llama.cpp/scripts/build-static-multi.sh

# 2. Revert GoReleaser config
git checkout .goreleaser-multiplatform.yml

# 3. Clean and rebuild
make clean
make release-multi-snapshot

# 4. Verify other platforms still work
ls -lh dist/outputs/dist/
```

---

## Performance Expectations

### Build Time

- **Before**: N/A (Windows disabled)
- **After**: +1-2 minutes for Windows build
- **Total**: ~3-5 minutes for all 5 platforms

### Binary Size

| Platform | Before | After | Change |
|----------|--------|-------|--------|
| Linux x86_64 | 3.8M | 3.8M | No change |
| Linux arm64 | 3.6M | 3.6M | No change |
| macOS x86_64 | 3.9M | 3.9M | No change |
| macOS arm64 | 3.6M | 3.6M | No change |
| Windows x86_64 | N/A | ~4.2M | NEW |

### Runtime Performance

Windows performance should be within 5% of Linux performance for typical workloads.

---

## Success Criteria

✅ **Build Success**: All 5 platforms build without errors  
✅ **Binary Quality**: Windows binary runs correctly  
✅ **No Regression**: Linux/macOS unaffected  
✅ **Documentation**: All docs updated  
✅ **Testing**: Basic functionality verified  

---

## Next Steps After Implementation

1. **Test on real Windows systems** (Windows 10, 11, Server)
2. **Monitor for user-reported issues**
3. **Consider native Windows builds** for optimization (future)
4. **Add Windows to CI/CD pipeline**
5. **Create Windows installer** (optional, future)

---

## Estimated Timeline

| Task | Time | Cumulative |
|------|------|------------|
| Update build script | 5 min | 5 min |
| Remove threading disable | 2 min | 7 min |
| Update GoReleaser config | 5 min | 12 min |
| Test library build | 5 min | 17 min |
| Full multiplatform build | 20 min | 37 min |
| Verify artifacts | 2 min | 39 min |
| Test Windows binary | 10 min | 49 min |
| Update documentation | 5 min | 54 min |
| **TOTAL** | **54 min** | |

**Actual time may vary**: 30-90 minutes depending on build speeds and testing thoroughness.

---

## Support

For issues or questions:
- Review [WINDOWS_BUILD_ANALYSIS.md](WINDOWS_BUILD_ANALYSIS.md) for detailed technical analysis
- Check [MULTIPLATFORM_BUILD_FIXES.md](MULTIPLATFORM_BUILD_FIXES.md) for general build system info
- Open GitHub issue with build logs

---

**Status**: Ready for implementation  
**Confidence**: High (95%)  
**Last Updated**: October 24, 2024