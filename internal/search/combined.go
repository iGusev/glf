// Package search combines fuzzy name search with full-text description search
package search

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/igusev/glf/internal/index"
	"github.com/igusev/glf/internal/model"
)

// calculateRelevanceMultiplier returns a multiplier [0.0, 1.0] based on search relevance
// This prevents history/starred bonuses from overwhelming irrelevant search results
//
// Logic (based on real data showing max scores ~1.4-1.6):
//   - searchScore < 0.1:  multiplier = 0.0 (too irrelevant, no boost)
//   - searchScore >= 1.4: multiplier = 1.0 (sufficiently relevant, full boost)
//   - 0.1 <= searchScore < 1.4: non-linear curve with 6 gradations
//
// Gradations provide very smooth transitions (6 levels):
//   - 0.10-0.30: very slow ramp (0.00 → 0.15) - minimal boost for very weak matches
//   - 0.30-0.50: slow ramp (0.15 → 0.30) - small boost for weak matches
//   - 0.50-0.70: moderate ramp (0.30 → 0.50) - moderate boost for decent matches
//   - 0.70-0.90: medium-fast ramp (0.50 → 0.70) - good boost for solid matches
//   - 0.90-1.15: fast ramp (0.70 → 0.85) - strong boost for very good matches
//   - 1.15-1.40: very fast ramp (0.85 → 1.00) - full boost for excellent matches
//
// Example: if searchScore = 0.012 (very low relevance), multiplier = 0.0
//
//	History boost of 57 and starred boost of 50 would be zeroed out,
//	preventing them from overwhelming the search score
func calculateRelevanceMultiplier(searchScore float64) float64 {
	// Minimum threshold: scores below this get no history/starred boost
	const minRelevanceThreshold = 0.1

	// Full boost threshold: scores at or above this get full history/starred boost
	// Based on real data showing max scores around 1.4-1.6
	const fullBoostThreshold = 1.4

	if searchScore < minRelevanceThreshold {
		// Too irrelevant - no history/starred boost
		return 0.0
	}

	if searchScore >= fullBoostThreshold {
		// Sufficiently relevant - full history/starred boost
		return 1.0
	}

	// Non-linear curve with 6 gradations for very smooth transitions
	// Normalize score to [0, 1] range
	normalized := (searchScore - minRelevanceThreshold) / (fullBoostThreshold - minRelevanceThreshold)

	// Apply piece-wise function with 6 different slopes for fine-grained gradations
	// This creates a very nuanced curve with smoother transitions
	switch {
	case normalized < 0.154: // 0.10 to 0.30 range (very slow ramp)
		// Minimal boost for very weak matches
		return normalized * 0.974 // 0.00 → 0.15
	case normalized < 0.308: // 0.30 to 0.50 range (slow ramp)
		// Small boost for weak matches
		return 0.15 + (normalized-0.154)*0.974 // 0.15 → 0.30
	case normalized < 0.462: // 0.50 to 0.70 range (moderate ramp)
		// Moderate boost for decent matches
		return 0.30 + (normalized-0.308)*1.299 // 0.30 → 0.50
	case normalized < 0.615: // 0.70 to 0.90 range (medium-fast ramp)
		// Good boost for solid matches
		return 0.50 + (normalized-0.462)*1.303 // 0.50 → 0.70
	case normalized < 0.808: // 0.90 to 1.15 range (fast ramp)
		// Strong boost for very good matches
		return 0.70 + (normalized-0.615)*0.777 // 0.70 → 0.85
	default: // 1.15 to 1.40 range (very fast ramp)
		// Full boost for excellent matches
		return 0.85 + (normalized-0.808)*0.781 // 0.85 → 1.00
	}
}

// CombinedSearch performs unified search using Bleve across project names, paths, and descriptions
// For empty queries, returns all projects sorted by history
// If descIndex is provided, it will be used; otherwise a new index will be opened
func CombinedSearch(query string, projects []model.Project, historyScores map[string]int, cacheDir string) ([]index.CombinedMatch, error) {
	return CombinedSearchWithIndex(query, projects, historyScores, cacheDir, nil)
}

// CombinedSearchWithIndex is like CombinedSearch but accepts an already-open index
func CombinedSearchWithIndex(query string, projects []model.Project, historyScores map[string]int, cacheDir string, descIndex *index.DescriptionIndex) ([]index.CombinedMatch, error) {
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
		descIndex, _, err = index.NewDescriptionIndexWithAutoRecreate(indexPath)
		if err != nil {
			// Failed to open index
			return nil, fmt.Errorf("failed to open search index: %w", err)
		}
		// Note: recreated flag ignored here - if index was recreated, it will be empty
		// and search will return no results, prompting user to run 'glf sync'
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
	projectMap := make(map[string]model.Project)
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

		// Calculate starred bonus
		starredBonus := 0
		if fullProject.Starred {
			starredBonus += 3
		}

		// Apply context-dependent scaling based on search relevance
		// This prevents history/starred from dominating when search relevance is low
		relevanceMultiplier := calculateRelevanceMultiplier(match.Score)
		adjustedHistoryScore := float64(historyScore) * relevanceMultiplier
		adjustedStarredBonus := float64(starredBonus) * relevanceMultiplier

		// Calculate total score (search + context-adjusted history + starred)
		// Example: searchScore=0.012 (too low) -> multiplier=0.0 -> no history/starred boost
		//          searchScore=0.5 (moderate) -> multiplier≈0.34 -> partial boost
		//          searchScore=1.2 (good) -> multiplier≈0.92 -> strong boost
		//          searchScore=1.4+ (high) -> multiplier=1.0 -> full boost
		totalScore := match.Score + adjustedHistoryScore + adjustedStarredBonus

		results = append(results, index.CombinedMatch{
			Project:      fullProject,
			SearchScore:  match.Score,
			HistoryScore: historyScore,
			StarredBonus: starredBonus,
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
func allProjectsSortedByHistory(projects []model.Project, historyScores map[string]int) []index.CombinedMatch {
	results := make([]index.CombinedMatch, len(projects))

	for i, p := range projects {
		historyScore := 0
		if score, exists := historyScores[p.Path]; exists {
			historyScore = score
		}

		// Calculate starred bonus
		starredBonus := 0
		if p.Starred {
			starredBonus += 3
		}

		results[i] = index.CombinedMatch{
			Project:      p,
			SearchScore:  0.0, // No search for empty query
			HistoryScore: historyScore,
			StarredBonus: starredBonus,
			TotalScore:   float64(historyScore) + float64(starredBonus),
			Source:       index.MatchSourceName,
			Snippet:      p.Description, // Show full description for empty query
		}
	}

	// Sort by total score (history only for empty query) descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalScore > results[j].TotalScore
	})

	return results
}
