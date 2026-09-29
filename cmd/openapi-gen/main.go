// Command openapi-gen writes the API document and the Postman collection from
// the routes the server actually registers (RM-5).
//
// It builds a real HTTPTransport, registers the core routes on it and then the
// routes of every module compiled into this binary, and describes what that
// transport is serving. Nothing is written by hand, so a route added without a
// document entry cannot exist.
//
// The build tags decide which document is produced: without tags the core
// document, with `-tags commercial` the document that also carries the
// commercial module routes. Both are checked in under docs/api and the drift
// test in this package fails when either is stale.
//
//	go run ./cmd/openapi-gen -out docs/api
//	go run -tags commercial ./cmd/openapi-gen -out docs/api
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/madeindigio/remembrances-mcp/v2/internal/transport"
	"github.com/madeindigio/remembrances-mcp/v2/pkg/modules"
)

func main() {
	out := flag.String("out", "docs/api", "Directory the documents are written to")
	flag.Parse()

	artefacts, err := Generate()
	if err != nil {
		fmt.Fprintln(os.Stderr, "openapi-gen:", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "openapi-gen:", err)
		os.Exit(1)
	}
	for name, content := range artefacts {
		path := filepath.Join(*out, name)
		if err := os.WriteFile(path, content, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "openapi-gen:", err)
			os.Exit(1)
		}
		fmt.Println("wrote", path)
	}
}

// documentVersion is the version stamped into the artefacts.
//
// It is the API contract version, not the binary version. The binary version
// is injected with -ldflags at release time, so using it would make the
// checked in documents differ between a developer's machine and a release
// build, and the drift test would fail for a reason that has nothing to do
// with the API. Bump transport.APIVersion when the surface changes
// incompatibly.
func documentVersion() string { return transport.APIVersion }

// Generate produces the artefacts for the build it is compiled into, keyed by
// file name. It is exported to the test in this package, which regenerates in
// memory and compares against the checked in files.
func Generate() (map[string][]byte, error) {
	routes, err := registeredRoutes()
	if err != nil {
		return nil, err
	}

	doc, err := transport.BuildDocument(routes, documentVersion())
	if err != nil {
		return nil, fmt.Errorf("building the OpenAPI document: %w", err)
	}
	collection, err := transport.BuildPostmanCollection(routes, documentVersion())
	if err != nil {
		return nil, fmt.Errorf("building the Postman collection: %w", err)
	}

	openapiJSON, err := marshal(doc)
	if err != nil {
		return nil, err
	}
	postmanJSON, err := marshal(collection)
	if err != nil {
		return nil, err
	}

	return map[string][]byte{
		OpenAPIFileName: openapiJSON,
		PostmanFileName: postmanJSON,
	}, nil
}

// marshal renders a document deterministically, with a trailing newline so the
// files are ordinary text files.
func marshal(v any) ([]byte, error) {
	body, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(body, '\n'), nil
}

// registeredRoutes drives a real transport exactly as the server does, then
// asks it what it serves.
func registeredRoutes() ([]transport.APIRoute, error) {
	h := transport.NewHTTPTransport(":0")
	if err := h.SetupMCPRoutes(nil); err != nil {
		return nil, fmt.Errorf("registering the core routes: %w", err)
	}

	providers, authProviders := moduleProviders()
	if len(providers) > 0 {
		h.RegisterModuleRoutes(providers, authProviders...)
	}

	routes := h.RegisteredRoutes()
	// The MCP Streamable HTTP endpoint is served by the embedded go-mcp
	// transport on its own listener, so it is not in this mux; it is part of
	// the API a client talks to and is documented as an external operation.
	return append(routes, transport.MCPEndpointRoutes(transport.DefaultMCPEndpoint)...), nil
}

// moduleProviders instantiates every module registered in this build and keeps
// the ones that contribute HTTP routes. Routes() is a declaration, so it is
// answered by an unprovisioned instance; nothing here touches storage.
func moduleProviders() ([]modules.HTTPEndpointProvider, []modules.HTTPAuthProvider) {
	var endpoints []modules.HTTPEndpointProvider
	var auth []modules.HTTPAuthProvider

	for _, info := range sortedModules() {
		if info.New == nil {
			continue
		}
		instance := info.New()
		if provider, ok := instance.(modules.HTTPEndpointProvider); ok {
			endpoints = append(endpoints, provider)
		}
		if provider, ok := instance.(modules.HTTPAuthProvider); ok {
			auth = append(auth, provider)
		}
	}
	return endpoints, auth
}
