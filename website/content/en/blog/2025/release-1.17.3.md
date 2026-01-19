---
title: "Release 1.17.3"
date: 2025-12-20
categories: [release]
tags: [release, update, bugfix, stability]
---

Version **1.17.3** of Remembrances-MCP is now available! This is a stability-focused release that addresses important compatibility and logging issues.

## What's Fixed

### Improved Session Management in Streamable HTTP Mode

We've enhanced the streamable HTTP mode with better session handling:

- **Better logging:** Lack of session issues are now properly logged in streamable HTTP mode, making it easier to diagnose connection problems
- **Improved debugging:** Enhanced visibility into session lifecycle helps troubleshoot integration issues

### Enhanced MCP Client Compatibility

Fixed a critical issue that affected some MCP clients:

- **Better schema handling:** Tools with no properties now correctly generate schemas when the tool type is "object"
- **Wider client support:** This fix ensures compatibility with a broader range of MCP client implementations
- **Fewer integration errors:** Eliminates errors that previously occurred in certain client configurations

## Why This Update Matters

While this release doesn't introduce new features, it significantly improves the reliability and compatibility of existing functionality:

- **Production-ready streaming:** The streamable HTTP mode is now more robust and ready for production use
- **Better troubleshooting:** Enhanced logging makes it easier to identify and resolve issues
- **Broader compatibility:** Works with more MCP client implementations out of the box

## Who Should Update?

This update is especially important if you:

- Use streamable HTTP mode in your deployments
- Experience client compatibility issues with certain MCP clients
- Need better logging for production environments

Download the new version here:

[Download Remembrances-MCP v1.17.3](https://github.com/madeindigio/remembrances-mcp/releases/tag/v1.17.3)

Update now for a more stable and compatible experience!
