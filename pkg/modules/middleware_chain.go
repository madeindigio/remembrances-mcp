package modules

import (
	"context"
	"sort"
	"strings"

	"github.com/ThinkInAIXYZ/go-mcp/protocol"
)

// MiddlewareChain manages tool middleware execution.
type MiddlewareChain struct {
	middlewares  []ToolMiddleware
	transformers []ToolTransformer
	validators   []ToolValidator
	enrichers    []ResponseEnricher
}

// NewMiddlewareChain creates a new middleware chain.
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{}
}

// AddMiddleware registers a ToolMiddleware.
func (mc *MiddlewareChain) AddMiddleware(m ToolMiddleware) {
	mc.middlewares = append(mc.middlewares, m)
	sort.Slice(mc.middlewares, func(i, j int) bool {
		return mc.middlewares[i].Priority() < mc.middlewares[j].Priority()
	})
}

// AddTransformer registers a ToolTransformer.
func (mc *MiddlewareChain) AddTransformer(t ToolTransformer) {
	mc.transformers = append(mc.transformers, t)
}

// AddValidator registers a ToolValidator.
func (mc *MiddlewareChain) AddValidator(v ToolValidator) {
	mc.validators = append(mc.validators, v)
}

// AddEnricher registers a ResponseEnricher.
func (mc *MiddlewareChain) AddEnricher(e ResponseEnricher) {
	mc.enrichers = append(mc.enrichers, e)
}

// Wrap wraps a handler with validation, middleware, transforms, and enrichers.
func (mc *MiddlewareChain) Wrap(toolName string, handler ToolHandler) ToolHandler {
	wrapped := handler

	for i := len(mc.middlewares) - 1; i >= 0; i-- {
		m := mc.middlewares[i]
		if mc.matchesFilter(toolName, m.ToolFilter()) {
			wrapped = m.Wrap(wrapped)
		}
	}

	withValidation := func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		for _, v := range mc.validators {
			if mc.matchesFilter(toolName, v.ToolFilter()) {
				if err := v.Validate(ctx, req); err != nil {
					return &protocol.CallToolResult{
						IsError: true,
						Content: []protocol.Content{&protocol.TextContent{Type: "text", Text: err.Error()}},
					}, nil
				}
			}
		}
		return wrapped(ctx, req)
	}

	withPostProcess := func(ctx context.Context, req *protocol.CallToolRequest) (*protocol.CallToolResult, error) {
		res, err := withValidation(ctx, req)
		if err != nil {
			return res, err
		}

		for _, t := range mc.transformers {
			if mc.matchesFilter(toolName, t.ToolFilter()) {
				res, err = t.Transform(ctx, req, res)
				if err != nil {
					return res, err
				}
			}
		}

		for _, e := range mc.enrichers {
			if mc.matchesFilter(toolName, e.ToolFilter()) {
				res, err = e.Enrich(ctx, req, res)
				if err != nil {
					return res, err
				}
			}
		}

		return res, nil
	}

	return withPostProcess
}

func (mc *MiddlewareChain) matchesFilter(toolName string, filter []string) bool {
	if filter == nil || len(filter) == 0 {
		return true
	}
	for _, pattern := range filter {
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			if strings.HasPrefix(toolName, prefix) {
				return true
			}
		} else if pattern == toolName {
			return true
		}
	}
	return false
}
