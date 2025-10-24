# Windows Build - Current Blocker

**Date**: October 24, 2024  
**Status**: ⚠️ BLOCKED  
**Issue**: MinGW Windows API compatibility

---

## Problem Summary

Windows builds are currently blocked by a compatibility issue between the MinGW headers in the Docker container and the Windows API used by llama.cpp.

### Error

```
/go/src/github.com/madeindigio/remembrances-mcp/go-llama.cpp/llama.cpp/llama.cpp:694:71: 
error: 'PWIN32_MEMORY_RANGE_ENTRY' has not been declared

/go/src/github.com/madeindigio/remembrances-mcp/go-llama.cpp/llama.cpp/llama.cpp:702:17: 
error: 'WIN32_MEMORY_RANGE_ENTRY' was not declared in this scope
```

### Root Cause

1. **MinGW Version**: GCC 10.0.0-posix (2021) with old Windows SDK headers
2. **API Requirement**: llama.cpp uses `WIN32_MEMORY_RANGE_ENTRY` (Windows 8+ API)
3. **Header Gap**: MinGW headers in container don't include this newer API

The structure `WIN32_MEMORY_RANGE_ENTRY` was introduced in Windows 8 SDK but the MinGW headers in `goreleaser-cross:v1.21` are from an older SDK version.

---

## What We Tried

### ✅ Attempt 1: Use POSIX-threaded MinGW (SUCCESSFUL for threading)
**Result**: Threading issues resolved, but hit Windows API compatibility issue

```bash
CC="x86_64-w64-mingw32-gcc-posix"
CXX="x86_64-w64-mingw32-g++-posix"
AR="x86_64-w64-mingw32-gcc-ar-posix"
LDFLAGS="-static -pthread -static-libgcc -static-libstdc++"
```

### ❌ Attempt 2: Target Windows 7 API
**Result**: Didn't prevent the code path from compiling

```bash
CFLAGS="-D_WIN32_WINNT=0x0601"  # Windows 7
CXXFLAGS="-D_WIN32_WINNT=0x0601"
```

### ❌ Attempt 3: Disable MMAP
**Result**: Code path still compiled due to unconditional `#ifdef _WIN32`

```bash
CXXFLAGS="-DGGML_USE_MMAP=0"
```

---

## Solutions Available

### Option 1: Define Missing Structure (QUICK FIX)

Create a patch file to define the missing Windows API structure.

**Complexity**: Low  
**Time**: 30 minutes  
**Risk**: Low  

**Implementation**:

Create `go-llama.cpp/windows-api-compat.patch`:

```cpp
// Add before llama.cpp includes
#ifdef _WIN32
#ifndef WIN32_MEMORY_RANGE_ENTRY
typedef struct _WIN32_MEMORY_RANGE_ENTRY {
    PVOID  VirtualAddress;
    SIZE_T NumberOfBytes;
} WIN32_MEMORY_RANGE_ENTRY, *PWIN32_MEMORY_RANGE_ENTRY;
#endif
#endif
```

Apply in build script before compilation.

### Option 2: Update llama.cpp Version (MEDIUM TERM)

Update to a newer version of llama.cpp that may have better Windows compatibility or doesn't use this specific API.

**Complexity**: Medium  
**Time**: 2-4 hours (includes testing all platforms)  
**Risk**: Medium (may break Linux/macOS builds)

**Steps**:
1. Check llama.cpp releases for Windows compatibility improvements
2. Test newer version in isolated branch
3. Verify all 5 platforms still build correctly

### Option 3: Use Newer Docker Image (LONG TERM)

Wait for or create a goreleaser-cross image with updated MinGW headers.

**Complexity**: High  
**Time**: Unknown (depends on upstream)  
**Risk**: Low  

**Considerations**:
- May require custom Docker image
- Could affect other projects using same image
- Need to maintain custom image

### Option 4: Native Windows Build (ALTERNATIVE)

Build Windows binaries natively on Windows using GitHub Actions Windows runners.

**Complexity**: Medium  
**Time**: 4-6 hours  
**Risk**: Low  

**Pros**:
- Native Windows SDK (latest headers)
- Potentially better performance
- No cross-compilation issues

**Cons**:
- Separate build pipeline
- Longer CI/CD time
- Two build systems to maintain

---

## Recommended Solution

**Option 1: Define Missing Structure** - Quick patch approach

This is the fastest way to unblock Windows builds:

1. Add structure definition in build script or source
2. Test build succeeds
3. Verify Windows binary works
4. Document as temporary workaround
5. Plan migration to Option 2 or 4 long-term

---

## Current Status

### Working Platforms (4/5)

- ✅ Linux x86_64 (amd64) - 3.8M
- ✅ Linux ARM64 - 3.6M
- ✅ macOS x86_64 (Intel) - 3.9M
- ✅ macOS ARM64 (Apple Silicon) - 3.6M

### Blocked Platform (1/5)

- ⚠️ Windows x86_64 (amd64) - **BLOCKED** by MinGW API compatibility

### Progress Made

1. ✅ Fixed GoReleaser version compatibility
2. ✅ Enabled non-interactive builds
3. ✅ Fixed BUILD_NUMBER issues
4. ✅ Disabled Apple frameworks for cross-compilation
5. ✅ Fixed macOS archiver tools
6. ✅ Resolved CGO linking conflicts
7. ✅ Switched to POSIX-threaded MinGW (threading now works!)
8. ⚠️ **BLOCKED**: Windows API headers incompatibility

---

## Impact Assessment

### Without Windows Support

- **Platform Coverage**: 80% (4/5)
- **User Base**: ~80% covered (Windows is ~20% of potential users)
- **Build System**: Fully functional for all other platforms

### With Windows Support

- **Platform Coverage**: 100% (5/5)
- **User Base**: 100% covered
- **Completeness**: Full cross-platform solution

---

## Next Steps

### Immediate (Choose One)

1. **Quick Win**: Implement Option 1 (patch missing structure)
2. **Document & Ship**: Release 4-platform build, mark Windows as "coming soon"
3. **Native Build**: Start implementing Option 4 (GitHub Actions Windows)

### Recommended Path

1. **Now**: Document current state (this file)
2. **Short-term**: Implement Option 1 patch (if feasible)
3. **Medium-term**: Evaluate Option 2 (llama.cpp update) or Option 4 (native builds)
4. **Long-term**: Migrate to most sustainable solution

---

## Code Changes Made

### Files Modified

1. `go-llama.cpp/scripts/build-static-multi.sh`
   - Changed to POSIX-threaded compilers ✅
   - Added Windows 7 target flag
   - Added MMAP disable flag (didn't help)

2. `.goreleaser-multiplatform.yml`
   - Re-enabled Windows build hooks
   - Re-enabled Windows binary build section
   - Re-enabled Windows in UPX

### Changes to Revert (if giving up on cross-compile)

If deciding to use native Windows builds instead:

```bash
# Revert .goreleaser-multiplatform.yml
git diff .goreleaser-multiplatform.yml

# Keep the POSIX compiler changes (they work!)
# Only revert the Windows build enablement
```

---

## Alternative: Ship Without Windows

If Windows support is not critical:

### Option A: Document as Known Limitation

Update documentation to state:
- 4 platforms fully supported
- Windows builds coming in future release
- Users can build locally on Windows

### Option B: Provide Build Instructions

Create `docs/BUILD_WINDOWS_LOCALLY.md` with:
- Native Windows build instructions
- Required tools (Go, MinGW-w64, CMake)
- Step-by-step compilation guide

---

## Technical Details

### MinGW Version in Container

```
x86_64-w64-mingw32-gcc-posix (GCC) 10-posix 20210110
```

### Missing API Definition

```c
// Required but missing from old MinGW headers
typedef struct _WIN32_MEMORY_RANGE_ENTRY {
    PVOID  VirtualAddress;
    SIZE_T NumberOfBytes;
} WIN32_MEMORY_RANGE_ENTRY, *PWIN32_MEMORY_RANGE_ENTRY;
```

### llama.cpp Code Location

File: `go-llama.cpp/llama.cpp/llama.cpp`  
Lines: 694-703  
Function: `llama_mmap::llama_mmap` constructor

The code uses `PrefetchVirtualMemory` Windows API which requires `WIN32_MEMORY_RANGE_ENTRY`.

---

## Conclusion

We've successfully resolved the original MinGW threading issues by switching to POSIX-threaded compilers. However, we've hit a secondary blocker: outdated Windows SDK headers in the MinGW distribution.

**Current State**: 80% complete (4/5 platforms working)

**Recommendation**: 
1. Ship current 4-platform build as v0.31.0
2. Implement Option 1 (structure patch) in parallel
3. Release Windows support as v0.31.1 or v0.32.0

---

**Last Updated**: October 24, 2024  
**Blocked By**: MinGW Windows SDK headers (2021 vintage)  
**Solution Complexity**: Low (30 min patch) to Medium (4-6 hr native build)