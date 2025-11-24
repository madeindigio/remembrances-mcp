# Remembrances MCP Website

Documentation website for Remembrances MCP, built with Hugo and the Docsy theme.

## Prerequisites

- Docker
- Make
- Git

## Quick Start

### Development Server

Start the Hugo development server with live reload:

```bash
make serve
```

The site will be available at: http://localhost:1313/remembrances-mcp/

Press `Ctrl+C` to stop the server.

### Build Static Site

Build the static site to the `public/` directory:

```bash
make build
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

## Troubleshooting

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
