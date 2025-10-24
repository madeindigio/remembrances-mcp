# Build Quick Start Guide

**Remembrances-MCP** - Fast Multiplatform Build System

---

## 🚀 Quick Start (30 seconds)

### Single Platform Build

```bash
# Build for your current platform (5-8 minutes)
make build
```

### All Platforms Build (Fast Method) ⚡

```bash
# Build for all 5 platforms in parallel (6-10 minutes)
make release-multi-snapshot-fast
```

**Output:** `dist/outputs/dist/` contains binaries for Linux, macOS, Windows

---

## 📦 What Gets Built

The build system produces:

| Platform | Architecture | Output File | Size |
|----------|--------------|-------------|------|
| Linux | x86_64 | `remembrances-mcp_*_Linux_x86_64.tar.gz` | ~30MB |
| Linux | ARM64 | `remembrances-mcp_*_Linux_arm64.tar.gz` | ~28MB |
| macOS | Intel | `remembrances-mcp_*_Darwin_x86_64.tar.gz` | ~32MB |
| macOS | Apple Silicon | `remembrances-mcp_*_Darwin_arm64.tar.gz` | ~30MB |
| Windows | x86_64 | `remembrances-mcp_*_Windows_x86_64.zip` | ~25MB |

Each archive includes:
- Static binary with embedded llama.cpp
- README.md
- LICENSE
- CHANGELOG.md
- SHA256 checksum

---

## ⚡ Performance Comparison

### Traditional Sequential Build
```bash
make release-multi-snapshot
```
- **Time:** 30-40 minutes
- **Method:** Builds libraries one at a time

### New Parallel Build (Recommended)
```bash
make release-multi-snapshot-fast
```
- **Time:** 6-10 minutes
- **Improvement:** 70-80% faster
- **Method:** Builds all libraries simultaneously

---

## 🛠️ Common Build Commands

### Development

```bash
# Quick development build (current platform only)
make dev

# Run with auto-reload during development
make run-dev

# Format and lint code
make format lint
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

### Static Libraries

```bash
# Build llama.cpp libraries in parallel (FAST)
make llama-deps-all-parallel

# Build with clean slate and verification
make llama-deps-all-parallel-clean

# Build for specific platform
make llama-deps-linux-amd64
make llama-deps-darwin-arm64
```

### Release

```bash
# Snapshot build (local testing)
make release-multi-snapshot-fast

# Official release (requires GITHUB_TOKEN)
export GITHUB_TOKEN=ghp_xxxxx
make release-multi-fast
```

---

## 📋 Requirements

### For Local Builds (Current Platform)

- **Go 1.21+**
- **CMake 3.15+**
- **GCC/Clang** (system compiler)
- **8GB RAM**
- **5GB disk space**

```bash
# Ubuntu/Debian
sudo apt-get install build-essential cmake

# macOS
brew install cmake

# Check versions
go version      # Should be 1.21+
cmake --version # Should be 3.15+
```

### For Multiplatform Builds

**Option 1: Docker (Recommended)**

```bash
# Install Docker
curl -fsSL https://get.docker.com | sh

# Verify
docker --version
docker info
```

**Option 2: Parallel Builds**

```bash
# Install GNU Parallel for faster builds
# Ubuntu/Debian
sudo apt-get install parallel

# macOS
brew install parallel

# Verify
parallel --version
```

---

## 🐛 Troubleshooting

### Build Fails with "No space left"

```bash
# Check disk space
df -h .

# Clean old builds
make clean

# Remove Docker images cache
docker system prune -a
```

### Build Fails with "Cannot find compiler"

```bash
# Install build tools
# Ubuntu/Debian
sudo apt-get install build-essential gcc g++ cmake

# macOS
xcode-select --install
brew install cmake
```

### Docker Permission Denied

```bash
# Add user to docker group
sudo usermod -aG docker $USER

# Logout and login, or:
newgrp docker
```

### Parallel Build Not Working

```bash
# Install GNU Parallel
sudo apt-get install parallel  # Ubuntu/Debian
brew install parallel          # macOS

# Or use sequential build
make llama-deps-all
```

### Build Logs Show Errors

```bash
# Check platform-specific logs
ls build-*.log

# View failed build log
cat build-linux-amd64.log

# Clean and retry
make clean
make llama-deps-all-parallel-clean
```

---

## 🔧 Advanced Usage

### Custom Parallel Jobs

```bash
# Use 3 parallel jobs instead of 5
MAX_PARALLEL_JOBS=3 make llama-deps-all-parallel
```

### Build Only Specific Platforms

```bash
# Edit scripts/build-llama-parallel.sh
# Change: PLATFORMS=("linux-amd64" "darwin-arm64")
```

### Skip Library Rebuild

```bash
# If libraries already exist and haven't changed
# Just build Go binaries
make build
```

### Debug Build Issues

```bash
# Verbose output
./scripts/build-llama-parallel.sh --clean --verify

# Check library verification
cd go-llama.cpp
for lib in libbinding-*.a; do
    echo "=== $lib ==="
    ar t "$lib" | head -10
    ls -lh "$lib"
done
```

---

## 📊 Build Time Breakdown

### Sequential Build (~35 minutes)
```
├─ llama.cpp linux-amd64   : 6 min
├─ llama.cpp linux-arm64   : 7 min
├─ llama.cpp darwin-amd64  : 6 min
├─ llama.cpp darwin-arm64  : 7 min
├─ llama.cpp windows-amd64 : 6 min
├─ Go binary compilation   : 2 min
└─ Packaging & checksums   : 1 min
```

### Parallel Build (~8 minutes)
```
├─ llama.cpp all platforms : 7 min (parallel)
├─ Go binary compilation   : 2 min
└─ Packaging & checksums   : 1 min
```

---

## 🎯 Best Practices

### For Development
1. Use `make dev` for quick iterations
2. Run `make test` before committing
3. Use `make format lint` to maintain code quality

### For Testing Releases
1. Always test with `make release-multi-snapshot-fast` first
2. Verify binaries in `dist/outputs/dist/`
3. Test on target platforms before official release

### For Official Releases
1. Ensure clean git state (no uncommitted changes)
2. Tag the release: `git tag -a v1.0.0 -m "Release v1.0.0"`
3. Set `GITHUB_TOKEN` environment variable
4. Run `make release-multi-fast`
5. Verify release on GitHub

---

## 🔍 Verification

### Verify Built Binaries

```bash
# Extract and test
cd dist/outputs/dist
tar xzf remembrances-mcp_*_Linux_x86_64.tar.gz
cd remembrances-mcp_*_Linux_x86_64
./remembrances-mcp --version
./remembrances-mcp --help

# Check binary size and dependencies
ls -lh remembrances-mcp
file remembrances-mcp
ldd remembrances-mcp  # Should show minimal dependencies
```

### Verify Static Linking

```bash
# Binary should NOT depend on external llama.cpp
ldd ./dist/remembrances-mcp | grep -i llama
# Expected: No output (static linking successful)

# Check symbols are present
nm ./dist/remembrances-mcp | grep llama_
# Expected: Many llama_ symbols (embedded)
```

---

## 🆘 Getting Help

### Documentation
- Full build guide: [CROSS_COMPILATION.md](CROSS_COMPILATION.md)
- Improvement plan: [MULTIPLATFORM_BUILD_REVIEW.md](MULTIPLATFORM_BUILD_REVIEW.md)
- Main README: [../README.md](../README.md)

### Commands Reference
```bash
# Show all available targets
make help

# Show build information
make info
```

### Common Issues
- **Slow builds?** → Use parallel builds: `make llama-deps-all-parallel`
- **Out of memory?** → Reduce jobs: `MAX_PARALLEL_JOBS=2 make ...`
- **Permission errors?** → Check Docker group or use `sudo` carefully
- **Corrupted libraries?** → Clean build: `make clean && make llama-deps-all-parallel-clean`

---

## 📝 Examples

### Example 1: First-Time Build

```bash
# Clone repository
git clone https://github.com/madeindigio/remembrances-mcp.git
cd remembrances-mcp

# Install dependencies
make deps

# Build for all platforms (fast)
make release-multi-snapshot-fast

# Verify output
ls -lh dist/outputs/dist/
```

### Example 2: Quick Development Build

```bash
# Make code changes...

# Quick build for testing
make dev

# Run locally
./dist/remembrances-mcp --help
```

### Example 3: Release to GitHub

```bash
# Ensure clean state
git status
git pull origin main

# Create and push tag
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0

# Set GitHub token
export GITHUB_TOKEN=ghp_your_token_here

# Build and release
make release-multi-fast

# Check GitHub releases page
```

### Example 4: Build Specific Platforms

```bash
# Build only Linux platforms
make llama-deps-linux-amd64
make llama-deps-linux-arm64

# Then use standard goreleaser for those platforms
# (Edit .goreleaser-multiplatform.yml to include only linux builds)
```

---

## ⏱️ Performance Tips

1. **Use Parallel Builds:** Always prefer `*-parallel` or `*-fast` targets
2. **Cache Libraries:** Keep `go-llama.cpp/libbinding-*.a` files between builds
3. **Increase Jobs:** Use `MAX_PARALLEL_JOBS=8` if you have 8+ cores
4. **Use Docker Cache:** Don't prune Docker images between builds
5. **SSD Storage:** Build on SSD for 30-40% faster compilation
6. **RAM:** Ensure 8GB+ RAM for parallel builds

---

## 🎉 Success Checklist

- [ ] All 5 platform libraries built without errors
- [ ] Go binaries compiled for all platforms
- [ ] Archives created with correct naming
- [ ] Checksums generated
- [ ] Binaries tested on at least one platform
- [ ] Build completed in <15 minutes
- [ ] No warnings in build logs
- [ ] File permissions correct (non-root)
- [ ] Git repository clean (for releases)
- [ ] Documentation updated (if needed)

---

**Ready to build?** → `make release-multi-snapshot-fast` 🚀