// Package search combines fuzzy name search with full-text description search
package search

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/igusev/glf/internal/index"
	"github.com/igusev/glf/internal/types"
)

// CombinedSearch performs unified search using Bleve across project names, paths, and descriptions
// For empty queries, returns all projects sorted by history
// If descIndex is provided, it will be used; otherwise a new index will be opened
func CombinedSearch(query string, projects []types.Project, historyScores map[string]int, cacheDir string) ([]index.CombinedMatch, error) {
	return CombinedSearchWithIndex(query, projects, historyScores, cacheDir, nil)
}

// CombinedSearchWithIndex is like CombinedSearch but accepts an already-open index
func CombinedSearchWithIndex(query string, projects []types.Project, historyScores map[string]int, cacheDir string, descIndex *index.DescriptionIndex) ([]index.CombinedMatch, error) {
	if query == "" {
		// Empty query: return all projects sorted by history
		return allProjectsSortedByHistory(projects, historyScores), nil
	}

	// Non-empty query: use Bleve unified search
	var needClose bool
	if descIndex == nil {
		// No index provided, open it ourselves
		indexPath := filepath.Join(cacheDir, "description.bleve")
		if !index.Exists(indexPath) {
			// Index doesn't exist yet - return empty results
			// User should run 'glf sync' to build it
			return nil, fmt.Errorf("search index not found, run 'glf sync' to build it")
		}

		var err error
		descIndex, err = index.NewDescriptionIndex(indexPath)
		if err != nil {
			// Failed to open index
			return nil, fmt.Errorf("failed to open search index: %w", err)
		}
		needClose = true
	}

	if needClose {
		defer func() {
			if err := descIndex.Close(); err != nil {
				// Log error but don't fail the search operation
				_ = err // Error in deferred close is logged if needed
			}
		}()
	}

	// Search across all fields (ProjectName, ProjectPath, Description) with boosting
	bleveMatches, err := descIndex.Search(query, 100)
	if err != nil {
		// Search failed
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Create project lookup map to get full project details
	projectMap := make(map[string]types.Project)
	for _, p := range projects {
		projectMap[p.Path] = p
	}

	// Convert Bleve matches to CombinedMatch with history boost
	results := make([]index.CombinedMatch, 0, len(bleveMatches))
	for _, match := range bleveMatches {
		fullProject, ok := projectMap[match.Project.Path]
		if !ok {
			// Skip if project not found in original list
			continue
		}

		// Get history boost for this project
		historyScore := 0
		if score, exists := historyScores[fullProject.Path]; exists {
			historyScore = score
		}

		// Calculate total score (search + history)
		// History scores are already reduced by 10x in history.go
		totalScore := match.Score + float64(historyScore)

		results = append(results, index.CombinedMatch{
			Project:      fullProject,
			SearchScore:  match.Score,
			HistoryScore: historyScore,
			TotalScore:   totalScore,
			// Bleve searches all fields, so consider it as both name and description match
			Source:  index.MatchSourceName | index.MatchSourceDescription,
			Snippet: match.Snippet,
		})
	}

	// Sort by total score (search + history), highest first
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalScore > results[j].TotalScore
	})

	return results, nil
}

// allProjectsSortedByHistory returns all projects sorted by history scores
// Used for empty queries to show recently/frequently used projects first
func allProjectsSortedByHistory(projects []types.Project, historyScores map[string]int) []index.CombinedMatch {
	results := make([]index.CombinedMatch, len(projects))

	for i, p := range projects {
		historyScore := 0
		if score, exists := historyScores[p.Path]; exists {
			historyScore = score
		}

		results[i] = index.CombinedMatch{
			Project:      p,
			SearchScore:  0.0, // No search for empty query
			HistoryScore: historyScore,
			TotalScore:   float64(historyScore),
			Source:       index.MatchSourceName,
			Snippet:      "",
		}
	}

	// Sort by total score (history only for empty query) descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalScore > results[j].TotalScore
	})

	return results
}
