package mcp_tools

import (
	"context"
	"log/slog"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
)

const defaultAlternativesLimit = 5

func (tm *ToolManager) userAlternatives(ctx context.Context, table string) []string {
	counts, err := tm.storage.CountByUserID(ctx, table)
	if err != nil {
		slog.Warn("failed to compute user alternatives", "table", table, "err", err)
		return nil
	}
	return TopAlternativesFromCounts(counts, defaultAlternativesLimit)
}

func projectAlternativesFromList(projects []storage.CodeProject) []string {
	counts := make(map[string]int)
	for _, p := range projects {
		counts[p.ProjectID] = 1
	}
	return TopAlternativesFromCounts(counts, defaultAlternativesLimit)
}

func (cstm *CodeSearchToolManager) projectAlternatives(ctx context.Context) []string {
	codeStorage, ok := cstm.storage.(interface {
		ListCodeProjects(ctx context.Context) ([]storage.CodeProject, error)
	})
	if !ok {
		slog.Warn("storage does not support listing code projects for alternatives")
		return nil
	}
	projects, err := codeStorage.ListCodeProjects(ctx)
	if err != nil {
		slog.Warn("failed to list code projects for alternatives", "err", err)
		return nil
	}
	return projectAlternativesFromList(projects)
}

func (ctm *CodeToolManager) projectAlternatives(ctx context.Context) []string {
	codeStorage, ok := ctm.storage.(interface {
		ListCodeProjects(ctx context.Context) ([]storage.CodeProject, error)
	})
	if !ok {
		slog.Warn("storage does not support listing code projects for alternatives")
		return nil
	}
	projects, err := codeStorage.ListCodeProjects(ctx)
	if err != nil {
		slog.Warn("failed to list code projects for alternatives", "err", err)
		return nil
	}
	return projectAlternativesFromList(projects)
}
