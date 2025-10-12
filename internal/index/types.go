package index

import "github.com/igusev/glf/internal/types"

// DescriptionDocument represents an indexed project description
type DescriptionDocument struct {
	ProjectPath string // e.g., "numbuster/api/auth"
	ProjectName string // e.g., "line-login-initiator"
	Description string // Project description
}

// DescriptionMatch represents a search result from description index
type DescriptionMatch struct {
	Project types.Project // The matched project
	Score   float64       // Relevance score from bleve
	Snippet string        // Context snippet with highlighted match
}

// MatchSource indicates where the match was found
type MatchSource int

const (
	MatchSourceName        MatchSource = 1 << iota // Found in project name (fuzzy)
	MatchSourceDescription                         // Found in description (bleve)
)

// CombinedMatch represents a unified search result with score breakdown
type CombinedMatch struct {
	Project      types.Project
	SearchScore  float64     // Bleve relevance score
	HistoryScore int         // History boost (with exponential decay)
	TotalScore   float64     // Combined score (SearchScore + HistoryScore)
	Source       MatchSource // Bitflags: can be MatchSourceName | MatchSourceDescription
	Snippet      string      // Description snippet if found there
}
