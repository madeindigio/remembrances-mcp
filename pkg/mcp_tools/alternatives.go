package mcp_tools

import (
	"context"
	"log/slog"

	"github.com/madeindigio/remembrances-mcp/internal/storage"
)

const defaultAlternativesLimit = 5
const defaultMaxDistance = 3

// AlternativeSuggestions aggregates similarity-based matches and the legacy
// count-based alternatives to provide richer guidance when a lookup fails.
type AlternativeSuggestions struct {
	SimilarNames []SimilarMatch `json:"similar_names,omitempty" toon:"similar_names,omitempty"`
	OtherIDs     []string       `json:"other_ids,omitempty" toon:"other_ids,omitempty"`
}

func (tm *ToolManager) userAlternatives(ctx context.Context, table string) []string {
	counts, err := tm.storage.CountByUserID(ctx, table)
	if err != nil {
		slog.Warn("failed to compute user alternatives", "table", table, "err", err)
		return nil
	}
	return TopAlternativesFromCounts(counts, defaultAlternativesLimit)
}

// FindUserAlternatives returns both similarity-based suggestions and
// popularity-based alternatives for a given user_id.
func (tm *ToolManager) FindUserAlternatives(ctx context.Context, table, queryUserID string) AlternativeSuggestions {
	suggestions := AlternativeSuggestions{}

	ids, err := tm.storage.ListUserIDs(ctx, table)
	if err != nil {
		slog.Warn("failed to list user ids", "table", table, "err", err)
	} else if queryUserID != "" {
		suggestions.SimilarNames = FindSimilarStrings(queryUserID, ids, defaultMaxDistance)
	}

	suggestions.OtherIDs = tm.userAlternatives(ctx, table)
	return suggestions
}

// FindKeyAlternatives suggests similar fact keys for a user and includes
// alternative user IDs based on existing data.
func (tm *ToolManager) FindKeyAlternatives(ctx context.Context, userID, table, queryKey string) AlternativeSuggestions {
	suggestions := AlternativeSuggestions{}

	keys, err := tm.storage.ListFactKeys(ctx, userID)
	if err != nil {
		slog.Warn("failed to list fact keys", "user_id", userID, "err", err)
	} else if queryKey != "" {
		suggestions.SimilarNames = FindSimilarStrings(queryKey, keys, defaultMaxDistance)
	}

	suggestions.OtherIDs = tm.userAlternatives(ctx, table)
	return suggestions
}

// FindEntityAlternatives suggests similar entity IDs and lists available IDs.
func (tm *ToolManager) FindEntityAlternatives(ctx context.Context, queryEntityID string) AlternativeSuggestions {
	suggestions := AlternativeSuggestions{}

	ids, err := tm.storage.ListEntityIDs(ctx)
	if err != nil {
		slog.Warn("failed to list entity ids", "err", err)
		return suggestions
	}

	suggestions.SimilarNames = FindSimilarStrings(queryEntityID, ids, defaultMaxDistance)
	suggestions.OtherIDs = ids
	return suggestions
}

// FindDocumentAlternatives suggests similar document paths and lists available
// documents.
func (tm *ToolManager) FindDocumentAlternatives(ctx context.Context, queryPath string) AlternativeSuggestions {
	suggestions := AlternativeSuggestions{}

	paths, err := tm.storage.ListDocumentPaths(ctx)
	if err != nil {
		slog.Warn("failed to list document paths", "err", err)
		return suggestions
	}

	suggestions.SimilarNames = FindSimilarStrings(queryPath, paths, defaultMaxDistance)
	suggestions.OtherIDs = paths
	return suggestions
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

// FindProjectAlternatives suggests similar project IDs and provides the legacy
// list of available project IDs with counts.
func (cstm *CodeSearchToolManager) FindProjectAlternatives(ctx context.Context, queryProjectID string) AlternativeSuggestions {
	suggestions := AlternativeSuggestions{}

	codeStorage, ok := cstm.storage.(interface {
		ListCodeProjects(ctx context.Context) ([]storage.CodeProject, error)
	})
	if !ok {
		slog.Warn("storage does not support listing code projects for alternatives")
		return suggestions
	}

	projects, err := codeStorage.ListCodeProjects(ctx)
	if err != nil {
		slog.Warn("failed to list code projects for alternatives", "err", err)
		return suggestions
	}

	ids := make([]string, 0, len(projects))
	for _, p := range projects {
		ids = append(ids, p.ProjectID)
	}

	suggestions.SimilarNames = FindSimilarStrings(queryProjectID, ids, defaultMaxDistance)
	suggestions.OtherIDs = projectAlternativesFromList(projects)
	return suggestions
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

// FindProjectAlternatives suggests similar project IDs using the CodeToolManager.
func (ctm *CodeToolManager) FindProjectAlternatives(ctx context.Context, queryProjectID string) AlternativeSuggestions {
	suggestions := AlternativeSuggestions{}

	codeStorage, ok := ctm.storage.(interface {
		ListCodeProjects(ctx context.Context) ([]storage.CodeProject, error)
	})
	if !ok {
		slog.Warn("storage does not support listing code projects for alternatives")
		return suggestions
	}

	projects, err := codeStorage.ListCodeProjects(ctx)
	if err != nil {
		slog.Warn("failed to list code projects for alternatives", "err", err)
		return suggestions
	}

	ids := make([]string, 0, len(projects))
	for _, p := range projects {
		ids = append(ids, p.ProjectID)
	}

	suggestions.SimilarNames = FindSimilarStrings(queryProjectID, ids, defaultMaxDistance)
	suggestions.OtherIDs = projectAlternativesFromList(projects)
	return suggestions
}
