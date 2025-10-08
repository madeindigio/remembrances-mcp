# Multi-stage for copy surrealdb binary
FROM surrealdb/surrealdb:latest AS surrealdb-builder

FROM ollama/ollama:latest
COPY --from=surrealdb-builder /surreal /usr/local/bin/surreal

RUN apt-get update && apt-get install -y xz-utils && apt-get clean && rm -rf /var/lib/apt/lists/*

# Install s6-overlay
ARG S6_OVERLAY_VERSION=v3.2.1.0

ADD https://github.com/just-containers/s6-overlay/releases/download/v3.2.1.0/s6-overlay-noarch.tar.xz /tmp/
ADD https://github.com/just-containers/s6-overlay/releases/download/v3.2.1.0/s6-overlay-x86_64.tar.xz /tmp/
RUN tar -C / -Jxvf /tmp/s6-overlay-noarch.tar.xz
RUN tar -C / -Jxvf /tmp/s6-overlay-x86_64.tar.xz
RUN rm -rf /tmp/*

# Copy Remembrances-MCP binary
COPY dist/remembrances-mcp /usr/local/bin/remembrances-mcp

# Copy configuration and s6 scripts
COPY docker-scripts/ /

# Permissions for s6 scripts
RUN chmod +x /etc/cont-init.d/01-ollama-pull && \
    chmod +x /etc/services.d/ollama/run && \
    chmod +x /etc/services.d/remembrances/run

# Expose necessary ports (as remembrances has SSE, HTTP, unicorn beyond stdio)
EXPOSE 11434 3000 8080 8000

# ENTRYPOINT s6-init to manage processes
ENTRYPOINT ["/init"]
