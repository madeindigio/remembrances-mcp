---
title: "Release 1.17.1"
date: 2025-12-20
categories: [release]
tags: [release, update, changelog, http, streaming]
---

Version **1.17.1** of Remembrances-MCP is now available! This update introduces a major new feature for HTTP communication and several improvements to the installation process.

## What's New

### Streamable HTTP Mode Support

The star feature of this release is **streamable HTTP mode**, which allows real-time streaming communication over HTTP protocol. This enables:

- **Real-time responses:** Get progressive results as they're generated instead of waiting for complete responses
- **Better integration:** Easier integration with web applications and HTTP-based clients
- **Improved user experience:** See results streaming in, perfect for long-running operations

To use streamable HTTP mode, simply configure the appropriate port in your settings. The system will automatically handle streaming without the ':' character prefix issue that existed in early implementations.

### Enhanced Installation Scripts

We've significantly improved the installation experience across all platforms:

- **AVX-512 CPU detection:** The installer now automatically detects if your CPU supports AVX-512 instructions and downloads the optimized version for maximum performance
- **Improved macOS support:** Fixed compilation issues with embedded flavor on macOS systems
- **Better CUDA detection:** Enhanced CUDA libraries detection for Linux systems
- **More reliable binary detection:** Fixed errors when the installer tries to detect existing binaries

### Bug Fixes

- Fixed MCP client compatibility issues when llama.cpp outputs debug information
- Resolved shared library location problems in both embedded and non-embedded versions
- Improved macOS embedded build options

## Why Update?

- **Stay current with HTTP streaming:** Take advantage of modern real-time communication protocols
- **Easier installation:** Let the installer automatically detect and configure the best version for your hardware
- **Better stability:** Multiple bug fixes ensure smoother operation across all platforms

Download the new version here:

[Download Remembrances-MCP v1.17.1](https://github.com/madeindigio/remembrances-mcp/releases/tag/v1.17.1)

Update now and experience real-time streaming!
