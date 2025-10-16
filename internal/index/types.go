package index

import "github.com/igusev/glf/internal/types"

// DescriptionDocument represents an indexed project description
type DescriptionDocument struct {
	ProjectPath string // e.g., "backend/api/auth"
	ProjectName string // e.g., "login-service"
	Description string // Project description
	Starred     bool   // Whether the project is starred by the user
}

// DescriptionMatch represents a search result from description index
type DescriptionMatch struct {
	Project types.Project // The matched project
	Snippet string        // Context snippet with highlighted match
	Score   float64       // Relevance score from bleve
}

// MatchSource indicates where the match was found
type MatchSource int

const (
	// MatchSourceName indicates match found in project name (fuzzy)
	MatchSourceName MatchSource = 1 << iota
	// MatchSourceDescription indicates match found in description (bleve)
	MatchSourceDescription
)

// CombinedMatch represents a unified search result with score breakdown
type CombinedMatch struct {
	Project      types.Project
	Snippet      string      // Description snippet if found there
	SearchScore  float64     // Bleve relevance score
	TotalScore   float64     // Combined score (SearchScore + HistoryScore + StarredBonus)
	HistoryScore int         // History boost (with exponential decay)
	StarredBonus int         // Bonus for starred projects (+50 for starred)
	Source       MatchSource // Bitflags: can be MatchSourceName | MatchSourceDescription
}
