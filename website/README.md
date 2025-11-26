# Remembrances MCP Website

Documentation website for Remembrances MCP, built with Hugo and the Docsy theme.

## 🎉 Multi-Language Fix Applied

**Status**: ✅ **FIXED** - All language routing issues resolved

A configuration fix has been applied to resolve all multi-language navigation issues:
- Changed `defaultContentLanguageInSubdir = true` in `hugo.toml`
- Both English and Spanish now use consistent URL structure (`/en/` and `/es/`)
- Language selector, menus, and navigation now work correctly

**See**: [LANGUAGE-ROUTING-FIX.md](LANGUAGE-ROUTING-FIX.md) for technical details

**Quick verification**:
```bash
./verify-lang-fix.sh
```

## Prerequisites

### Option 1: Docker-based (Recommended)

- Docker
- Make
- Git

### Option 2: Local Hugo Extended

- **Hugo Extended** v0.146.0 or later (required for SCSS compilation)
- Node.js and npm
- Git

**Important**: You MUST use Hugo Extended, not the standard Hugo version. The Docsy theme requires Hugo Extended to compile SCSS/SASS files.

#### Installing Hugo Extended

**Ubuntu/Debian:**
```bash
# Download Hugo Extended
wget https://github.com/gohugoio/hugo/releases/download/v0.152.2/hugo_extended_0.152.2_linux-amd64.tar.gz

# Extract and install
tar -xzf hugo_extended_0.152.2_linux-amd64.tar.gz
sudo mv hugo /usr/local/bin/

# Verify installation
hugo version  # Should show "extended" in the output
```

**macOS:**
```bash
brew install hugo
```

**Windows:**
Download the extended version from [Hugo Releases](https://github.com/gohugoio/hugo/releases) and add to PATH.

## Quick Start

### Development Server

Start the Hugo development server with live reload:

```bash
make serve
```

The site will be available at: http://localhost:1313/remembrances-mcp/

Or, if using local Hugo Extended:

```bash
hugo server
```

Press `Ctrl+C` to stop the server.

### Build Static Site

Build the static site to the `public/` directory:

```bash
make build
```

Or, if using local Hugo Extended:

```bash
npm install  # First time only
hugo --cleanDestinationDir
```

### Publish to GitHub Pages

Build and publish to the `gh-pages` branch:

```bash
make publish
```

## Available Commands

Run `make help` to see all available commands:

```bash
make help
```

### Main Commands

- `make serve` - Start Hugo development server (interactive)
- `make serve-detached` - Start server in background
- `make stop` - Stop background server
- `make build` - Build static site
- `make publish` - Build and publish to gh-pages
- `make clean` - Clean build artifacts
- `make logs` - Show server logs (if running detached)

### Utility Commands

- `make npm-install` - Install npm dependencies
- `make npm-update` - Update npm dependencies
- `make shell` - Open shell in Hugo container

## Project Structure

```
website/
├── content/
│   ├── en/          # English content
│   └── es/          # Spanish content
├── layouts/         # Custom layouts
├── static/          # Static assets
├── hugo.toml        # Hugo configuration
├── Makefile         # Build commands
└── package.json     # NPM dependencies
```

## Multi-language Support

The site supports two languages:

- **English** (primary): `/en/`
- **Spanish** (secondary): `/es/`

Content is organized by language in the `content/` directory.

## Sections

- **About**: Information about the project
- **Documentation**: Technical documentation
- **Blog**: News and updates

## Docker-based Workflow

All Hugo commands run inside Docker containers using the `hugomods/hugo:exts` image, which includes:

- Hugo Extended (latest version, compatible with Docsy)
- Dart Sass
- All necessary dependencies

This ensures consistent builds regardless of your local Hugo installation and avoids version compatibility issues.

## Common Issues

### SCSS Compilation Errors

If you see errors like:
```
ERROR TOCSS: failed to transform "/scss/main.scss"
```

This means you're using standard Hugo instead of Hugo Extended. You MUST install Hugo Extended to compile SCSS files.

**Solution:**
1. Check your Hugo version: `hugo version` (should include "extended")
2. If not extended, install Hugo Extended following the instructions above
3. Clean and rebuild: `make clean && make build`

### Styles Not Loading

If the website displays but styles are missing:
1. Ensure you're using Hugo Extended (see above)
2. Check that `public/scss/main.min.*.css` exists after building
3. Rebuild with: `hugo --cleanDestinationDir`

### Port 1313 already in use

If you get a port conflict, stop any running Hugo servers:

```bash
make stop
# or
docker stop remembrances-hugo-server
```

### Permission issues

If you encounter permission issues with Docker volumes, check your Docker user configuration.

### Build fails

Clean build artifacts and try again:

```bash
make clean
make build
```

## License

See the main project LICENSE file.
