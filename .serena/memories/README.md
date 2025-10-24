# Remembrances-MCP Documentation

Welcome to the technical documentation for Remembrances-MCP.

## 📚 Available Documentation

### Build System

- **[MULTIPLATFORM_BUILD_FIXES.md](MULTIPLATFORM_BUILD_FIXES.md)** - Complete technical documentation of multiplatform build system fixes
  - Detailed problem analysis and solutions
  - Step-by-step fixes for cross-compilation issues
  - Platform-specific configurations
  - Troubleshooting guide

- **[BUILD_FIXES_SUMMARY.md](BUILD_FIXES_SUMMARY.md)** - Executive summary of build fixes
  - Quick overview of changes
  - Results and metrics
  - Usage instructions
  - Known limitations

#### Windows Build Enablement

- **[WINDOWS_BUILD_ANALYSIS.md](WINDOWS_BUILD_ANALYSIS.md)** - Technical analysis for enabling Windows builds
  - Root cause analysis of MinGW threading issues
  - 4 solution options with detailed comparison
  - Risk assessment and cost-benefit analysis
  - Implementation recommendations

- **[WINDOWS_BUILD_IMPLEMENTATION.md](WINDOWS_BUILD_IMPLEMENTATION.md)** - Step-by-step implementation guide
  - Complete code changes with diffs
  - Testing and verification procedures
  - Troubleshooting common issues
  - Rollback procedures

### Project Structure

See the main [README.md](../README.md) in the project root for:
- Project overview
- Installation instructions
- Usage examples
- Configuration options

See [.github/copilot-instructions.md](../.github/copilot-instructions.md) for:
- Project architecture
- Development guidelines
- Code style conventions
- Recent changes log

## 🎯 Quick Links

### For Developers

- **Building for multiple platforms**: [MULTIPLATFORM_BUILD_FIXES.md](MULTIPLATFORM_BUILD_FIXES.md)
- **Quick build summary**: [BUILD_FIXES_SUMMARY.md](BUILD_FIXES_SUMMARY.md)
- **Enabling Windows builds**: [WINDOWS_BUILD_ANALYSIS.md](WINDOWS_BUILD_ANALYSIS.md)
- **Windows implementation guide**: [WINDOWS_BUILD_IMPLEMENTATION.md](WINDOWS_BUILD_IMPLEMENTATION.md)
- **Project instructions**: [../.github/copilot-instructions.md](../.github/copilot-instructions.md)

### For Users

- **Installation**: [../README.md](../README.md)
- **Configuration**: [../README.md#configuration](../README.md#configuration)
- **Usage examples**: [../README.md#usage](../README.md#usage)

## 📝 Documentation Status

| Document | Status | Last Updated |
|----------|--------|--------------|
| MULTIPLATFORM_BUILD_FIXES.md | ✅ Complete | Oct 24, 2024 |
| BUILD_FIXES_SUMMARY.md | ✅ Complete | Oct 24, 2024 |
| WINDOWS_BUILD_ANALYSIS.md | ✅ Complete | Oct 24, 2024 |
| WINDOWS_BUILD_IMPLEMENTATION.md | ✅ Complete | Oct 24, 2024 |

## 🔄 Contributing Documentation

When adding new documentation:

1. Place technical docs in this `docs/` directory
2. Add a reference in this README
3. Update the status table above
4. Link from main README if user-facing

### Documentation Standards

- Use clear, descriptive titles
- Include table of contents for long documents
- Add code examples where applicable
- Keep documentation up-to-date with code changes
- Use markdown formatting consistently

## 📞 Support

For questions or issues:
- Check existing documentation first
- Review troubleshooting sections
- Open an issue on GitHub
- Refer to [copilot-instructions.md](../.github/copilot-instructions.md) for development context

---

**Project**: Remembrances-MCP  
**Repository**: https://github.com/madeindigio/remembrances-mcp  
**Documentation Directory**: `/docs`
