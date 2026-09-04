package transport

import (
	"encoding/json"
	"sort"
	"strings"
)

// The OpenAPI 3.1 and Postman documents built from the registered routes.
//
// The types are declared rather than assembled as free-form maps so that a
// field name typo is a compile error, and the few genuinely open parts (JSON
// Schema fragments) are json.RawMessage-free maps whose keys Go marshals in
// sorted order. Determinism matters here: the drift test compares bytes, so a
// document that reordered itself between runs would fail for no reason.

// Document is an OpenAPI 3.1 document.
type Document struct {
	OpenAPI    string              `json:"openapi"`
	Info       Info                `json:"info"`
	Servers    []Server            `json:"servers,omitempty"`
	Tags       []Tag               `json:"tags,omitempty"`
	Paths      map[string]PathItem `json:"paths"`
	Components Components          `json:"components"`
}

// Info is the document metadata.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	License     *struct {
		Name       string `json:"name"`
		Identifier string `json:"identifier,omitempty"`
	} `json:"license,omitempty"`
}

// Server is one server entry.
type Server struct {
	URL         string                    `json:"url"`
	Description string                    `json:"description,omitempty"`
	Variables   map[string]ServerVariable `json:"variables,omitempty"`
}

// ServerVariable is a templated part of a server URL.
type ServerVariable struct {
	Default     string `json:"default"`
	Description string `json:"description,omitempty"`
}

// Tag groups operations.
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// PathItem holds the operations of one path, keyed by lower case method.
type PathItem map[string]*Operation

// Operation is one documented operation.
type Operation struct {
	OperationID string                `json:"operationId"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	RequestBody *RequestBody          `json:"requestBody,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
}

// Parameter is one path or query parameter.
type Parameter struct {
	Name        string         `json:"name"`
	In          string         `json:"in"`
	Required    bool           `json:"required"`
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"schema"`
}

// RequestBody is a request body description.
type RequestBody struct {
	Required bool                 `json:"required"`
	Content  map[string]MediaType `json:"content"`
}

// Response is one response description.
type Response struct {
	Description string               `json:"description"`
	Content     map[string]MediaType `json:"content,omitempty"`
}

// MediaType carries the schema of one content type.
type MediaType struct {
	Schema map[string]any `json:"schema"`
}

// Components holds the reusable schemas and security schemes.
type Components struct {
	Schemas         map[string]map[string]any `json:"schemas"`
	SecuritySchemes map[string]map[string]any `json:"securitySchemes,omitempty"`
}

// bearerScheme is the name of the only security scheme the server accepts.
const bearerScheme = "controlJWT"

// documentDescription states what the document is and, as importantly, what
// it is not. A reader who believes a generated document describes more than
// the server serves is the failure this whole item exists to prevent.
const documentDescription = `HTTP surface of the Remembrances MCP server.

This document is generated from the route registrations (` + "`internal/transport`" + `),
not written by hand, and a test fails the build when a registered route has no
operation or an operation has no registered route.

Three listeners exist and they are separate:

- the HTTP JSON API (` + "`--http`" + `, :8080 by default) serves /health and the
  /mcp/tools placeholder endpoints of this document;
- the MCP Streamable HTTP transport (` + "`--mcp-http`" + `, :3000 by default) serves the
  /mcp endpoint, which is the real MCP surface and is described here as an
  external operation because an embedded library serves it;
- module routes are registered on the HTTP JSON API listener, under the base
  path each module declares, and appear here only in a build that includes
  those modules.

Authentication: module routes are behind the module authentication layer when a
module provides one. Alchemai Mem provides ` + "`Authorization: Bearer <Control JWT>`" + `,
verified against Control's JWKS with iss, aud=alchemai and exp checks; the
claims become the storage principal (see RM-2 and RM-10). The core routes of
this document carry no authentication of their own.`

// BuildDocument turns the registered routes into an OpenAPI 3.1 document.
func BuildDocument(routes []APIRoute, version string) (*Document, error) {
	if err := validateRoutes(routes); err != nil {
		return nil, err
	}

	doc := &Document{
		OpenAPI: "3.1.0",
		Info: Info{
			Title:       "Remembrances MCP HTTP API",
			Version:     version,
			Description: documentDescription,
		},
		Servers: []Server{
			{
				URL:         "http://{host}:{port}",
				Description: "HTTP JSON API listener and module routes",
				Variables: map[string]ServerVariable{
					"host": {Default: "localhost"},
					"port": {Default: "8080", Description: "8080 for the JSON API, 3000 for the MCP Streamable HTTP endpoint"},
				},
			},
		},
		Paths:      map[string]PathItem{},
		Components: components(),
	}

	tags := map[string]bool{}
	for _, route := range routes {
		item, ok := doc.Paths[route.Path]
		if !ok {
			item = PathItem{}
			doc.Paths[route.Path] = item
		}
		item[strings.ToLower(route.Method)] = operationFor(route)
		if route.Tag != "" {
			tags[route.Tag] = true
		}
	}

	names := make([]string, 0, len(tags))
	for name := range tags {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		doc.Tags = append(doc.Tags, Tag{Name: name, Description: tagDescription(name)})
	}

	return doc, nil
}

// tagDescription explains a tag, so a reader of the document does not have to
// guess what a module id means.
func tagDescription(name string) string {
	switch name {
	case "core":
		return "Endpoints the core HTTP JSON API transport serves itself"
	case "mcp":
		return "The MCP Streamable HTTP endpoint, served by the embedded go-mcp transport on its own listener"
	default:
		return "Routes provided by module " + name
	}
}

// operationFor renders one route as an OpenAPI operation.
func operationFor(route APIRoute) *Operation {
	op := &Operation{
		OperationID: route.OperationID,
		Summary:     route.Summary,
		Description: route.Description,
		Responses:   map[string]Response{},
	}
	if route.Tag != "" {
		op.Tags = []string{route.Tag}
	}
	for _, p := range route.Parameters {
		schema := map[string]any{"type": p.Schema}
		if p.Schema == "" {
			schema = map[string]any{"type": "string"}
		}
		op.Parameters = append(op.Parameters, Parameter{
			Name:        p.Name,
			In:          p.In,
			Required:    p.Required,
			Description: p.Description,
			Schema:      schema,
		})
	}
	if route.RequestSchema != "" {
		op.RequestBody = &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {Schema: ref(route.RequestSchema)},
			},
		}
	}

	success := Response{Description: "Success"}
	if schema := responseSchema(route); schema != nil {
		success.Content = map[string]MediaType{"application/json": {Schema: schema}}
	}
	op.Responses["200"] = success
	if route.Authenticated {
		op.Security = []map[string][]string{{bearerScheme: {}}}
		op.Responses["401"] = Response{Description: "No credentials, or credentials no provider accepted"}
		op.Responses["403"] = Response{Description: "Authenticated, but the credentials carry no usable principal"}
	}
	if route.RequestSchema != "" {
		op.Responses["400"] = Response{Description: "The request body is not valid JSON for this operation"}
	}
	if !route.External {
		op.Responses["405"] = Response{Description: "The path is served but not for this method; the Allow header lists the methods that are"}
	}
	return op
}

// responseSchema names the component a route answers with. A core route with
// no schema answers with no body (the CORS preflight); a module route with no
// declared schema answers with a shape only that module knows, and saying so
// is more honest than inventing one.
func responseSchema(route APIRoute) map[string]any {
	if route.ResponseSchema != "" {
		return ref(route.ResponseSchema)
	}
	if route.Source == SourceCore {
		return nil
	}
	return map[string]any{"type": "object", "description": "Response shape declared by the module that provides the route"}
}

// ref is a JSON Schema reference to a component.
func ref(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

// components declares the schemas the core operations use. They mirror the
// handlers in http.go; a schema without a handler is as wrong as a route
// without an operation.
func components() Components {
	return Components{
		Schemas: map[string]map[string]any{
			"HealthResponse": {
				"type":       "object",
				"required":   []string{"status"},
				"properties": map[string]any{"status": map[string]any{"type": "string", "const": "ok"}},
			},
			"ToolList": {
				"type":     "object",
				"required": []string{"tools"},
				"properties": map[string]any{
					"tools": map[string]any{"type": "array", "items": ref("Tool")},
				},
			},
			"Tool": {
				"type":     "object",
				"required": []string{"name"},
				"properties": map[string]any{
					"name":        map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"inputSchema": map[string]any{"type": "object", "description": "JSON Schema of the tool arguments"},
				},
			},
			"CallToolRequest": {
				"type":     "object",
				"required": []string{"name"},
				"properties": map[string]any{
					"name":      map[string]any{"type": "string", "description": "Tool name"},
					"arguments": map[string]any{"type": "object", "description": "Tool arguments, validated against the tool input schema"},
				},
			},
			"CallToolResult": {
				"type":     "object",
				"required": []string{"content"},
				"properties": map[string]any{
					"content": map[string]any{
						"type":  "array",
						"items": ref("ContentBlock"),
					},
					"isError": map[string]any{"type": "boolean"},
				},
			},
			"ContentBlock": {
				"type":     "object",
				"required": []string{"type"},
				"properties": map[string]any{
					"type": map[string]any{"type": "string"},
					"text": map[string]any{"type": "string"},
				},
			},
		},
		SecuritySchemes: map[string]map[string]any{
			bearerScheme: {
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
				"description":  "Control-issued JWT, verified against Control's JWKS with iss, aud=alchemai and exp checks",
			},
		},
	}
}

// PostmanCollection is a Postman v2.1.0 collection.
type PostmanCollection struct {
	Info     PostmanInfo       `json:"info"`
	Item     []PostmanItem     `json:"item"`
	Auth     *PostmanAuth      `json:"auth,omitempty"`
	Variable []PostmanVariable `json:"variable,omitempty"`
}

// PostmanInfo names the collection.
type PostmanInfo struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Schema      string `json:"schema"`
}

// PostmanItem is one folder or one request.
type PostmanItem struct {
	Name    string          `json:"name"`
	Item    []PostmanItem   `json:"item,omitempty"`
	Request *PostmanRequest `json:"request,omitempty"`
}

// PostmanRequest is one HTTP request.
type PostmanRequest struct {
	Method      string          `json:"method"`
	Header      []PostmanHeader `json:"header,omitempty"`
	URL         PostmanURL      `json:"url"`
	Body        *PostmanBody    `json:"body,omitempty"`
	Description string          `json:"description,omitempty"`
}

// PostmanHeader is one request header.
type PostmanHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PostmanURL is the structured URL Postman expects.
type PostmanURL struct {
	Raw      string            `json:"raw"`
	Host     []string          `json:"host"`
	Path     []string          `json:"path"`
	Variable []PostmanVariable `json:"variable,omitempty"`
}

// PostmanVariable is a collection or path variable.
type PostmanVariable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// PostmanBody is a raw JSON request body.
type PostmanBody struct {
	Mode    string         `json:"mode"`
	Raw     string         `json:"raw"`
	Options map[string]any `json:"options,omitempty"`
}

// PostmanAuth is the collection level auth.
type PostmanAuth struct {
	Type   string            `json:"type"`
	Bearer []PostmanVariable `json:"bearer,omitempty"`
}

// BuildPostmanCollection renders the same routes as a runnable collection, one
// request per documented operation, grouped by tag.
//
// It is built from the routes rather than from the rendered OpenAPI document
// so that the two artefacts cannot describe different things; the drift test
// compares both against the registrations.
func BuildPostmanCollection(routes []APIRoute, version string) (*PostmanCollection, error) {
	if err := validateRoutes(routes); err != nil {
		return nil, err
	}

	collection := &PostmanCollection{
		Info: PostmanInfo{
			Name:        "Remembrances MCP " + version,
			Description: documentDescription,
			Schema:      "https://schema.getpostman.com/json/collection/v2.1.0/collection.json",
		},
		Auth: &PostmanAuth{
			Type:   "bearer",
			Bearer: []PostmanVariable{{Key: "token", Value: "{{token}}"}},
		},
		Variable: []PostmanVariable{
			{Key: "base", Value: "http://localhost:8080", Description: "HTTP JSON API listener, where module routes are served"},
			{Key: "mcpBase", Value: "http://localhost:3000", Description: "MCP Streamable HTTP listener"},
			{Key: "token", Value: "", Description: "Control-issued JWT for authenticated module routes"},
		},
	}

	byTag := map[string][]APIRoute{}
	for _, route := range routes {
		byTag[route.Tag] = append(byTag[route.Tag], route)
	}
	tags := make([]string, 0, len(byTag))
	for tag := range byTag {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	for _, tag := range tags {
		group := byTag[tag]
		sort.Slice(group, func(i, j int) bool {
			if group[i].Path != group[j].Path {
				return group[i].Path < group[j].Path
			}
			return group[i].Method < group[j].Method
		})
		folder := PostmanItem{Name: tag}
		for _, route := range group {
			folder.Item = append(folder.Item, postmanRequest(route))
		}
		collection.Item = append(collection.Item, folder)
	}
	return collection, nil
}

// postmanRequest renders one operation as a request against the right base
// variable: the MCP endpoint lives on its own listener.
func postmanRequest(route APIRoute) PostmanItem {
	base := "{{base}}"
	if route.External {
		base = "{{mcpBase}}"
	}

	segments := make([]string, 0, 4)
	for _, segment := range strings.Split(strings.TrimPrefix(route.Path, "/"), "/") {
		if segment == "" {
			continue
		}
		if strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}") {
			segments = append(segments, ":"+strings.Trim(segment, "{}"))
			continue
		}
		segments = append(segments, segment)
	}

	item := PostmanItem{
		Name: route.Summary,
		Request: &PostmanRequest{
			Method:      route.Method,
			Description: route.Description,
			URL: PostmanURL{
				Raw:  base + route.Path,
				Host: []string{base},
				Path: segments,
			},
		},
	}
	for _, p := range route.Parameters {
		if p.In == "path" {
			item.Request.URL.Variable = append(item.Request.URL.Variable, PostmanVariable{
				Key:         p.Name,
				Value:       "",
				Description: p.Description,
			})
		}
	}
	if route.RequestSchema != "" {
		item.Request.Header = append(item.Request.Header, PostmanHeader{Key: "Content-Type", Value: "application/json"})
		item.Request.Body = &PostmanBody{
			Mode:    "raw",
			Raw:     exampleBody(route.RequestSchema),
			Options: map[string]any{"raw": map[string]any{"language": "json"}},
		}
	}
	if route.External {
		item.Request.Header = append(item.Request.Header,
			PostmanHeader{Key: "Accept", Value: "application/json, text/event-stream"})
	}
	return item
}

// exampleBody returns a request body a reviewer can send as it stands. The
// grant evidence is a collection that runs, so an empty body would defeat it.
func exampleBody(schema string) string {
	switch schema {
	case "CallToolRequest":
		body, _ := json.MarshalIndent(map[string]any{
			"name":      "list_available_tools",
			"arguments": map[string]any{},
		}, "", "  ")
		return string(body)
	default:
		return "{}"
	}
}
