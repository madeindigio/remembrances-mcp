.PHONY: help serve build publish clean

# Docker image configuration
DOCKER_IMAGE := remembrances-mcp-hugo
HUGO_IMAGE := hugomods/hugo:exts
CONTAINER_NAME := remembrances-hugo-server

# Directories
WEBSITE_DIR := $(shell pwd)
PUBLIC_DIR := public

help: ## Show this help message
	@echo "Remembrances MCP Website - Makefile"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

docker-build: ## Build the Docker image with Hugo extended
	@echo "Using Hugo image: $(HUGO_IMAGE)"

serve: ## Start Hugo development server (with live reload)
	@echo "Starting Hugo development server on http://localhost:1313/remembrances-mcp/"
	@echo "Press Ctrl+C to stop"
	docker run --rm -it \
		--name $(CONTAINER_NAME) \
		-v $(WEBSITE_DIR):/src \
		-p 1313:1313 \
		$(HUGO_IMAGE) \
		server --bind 0.0.0.0 --buildDrafts --buildFuture --disableFastRender

serve-detached: ## Start Hugo server in background
	@echo "Starting Hugo server in background..."
	docker run --rm -d \
		--name $(CONTAINER_NAME) \
		-v $(WEBSITE_DIR):/src \
		-p 1313:1313 \
		$(HUGO_IMAGE) \
		server --bind 0.0.0.0 --buildDrafts --buildFuture --disableFastRender
	@echo "Server running at http://localhost:1313/remembrances-mcp/"
	@echo "Stop with: make stop"

stop: ## Stop the background Hugo server
	@echo "Stopping Hugo server..."
	docker stop $(CONTAINER_NAME) || true

build: ## Build the static site
	@echo "Building static site..."
	docker run --rm \
		-v $(WEBSITE_DIR):/src \
		$(HUGO_IMAGE)
	@echo "Site built in $(PUBLIC_DIR)/"

publish: build ## Build and publish to gh-pages branch
	@echo "Publishing to gh-pages branch..."
	docker run --rm \
		-v $(WEBSITE_DIR):/src \
		-v $(HOME)/.gitconfig:/etc/gitconfig:ro \
		-v $(HOME)/.ssh:/root/.ssh:ro \
		-w /src \
		python:3.10-slim \
		bash -c "cd /src && python3 publish_gh_pages_docker.py"

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf $(PUBLIC_DIR) resources .hugo_build.lock

logs: ## Show Hugo server logs (if running detached)
	docker logs -f $(CONTAINER_NAME)

shell: ## Open a shell in Hugo container
	docker run --rm -it \
		-v $(WEBSITE_DIR):/src \
		$(HUGO_IMAGE) \
		sh

npm-install: ## Install npm dependencies in Docker
	@echo "Installing npm dependencies..."
	docker run --rm \
		-v $(WEBSITE_DIR):/src \
		-w /src \
		node:20-alpine \
		npm install

npm-update: ## Update npm dependencies
	@echo "Updating npm dependencies..."
	docker run --rm \
		-v $(WEBSITE_DIR):/src \
		-w /src \
		node:20-alpine \
		npm update
