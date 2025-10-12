package index

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/igusev/glf/internal/types"
)

// DescriptionIndex manages the bleve index for project descriptions
type DescriptionIndex struct {
	index bleve.Index
	path  string
}

// NewDescriptionIndex creates or opens a description index
func NewDescriptionIndex(indexPath string) (*DescriptionIndex, error) {
	var index bleve.Index
	var err error

	// Check if index already exists
	if _, statErr := os.Stat(indexPath); os.IsNotExist(statErr) {
		// Create new index with custom mapping
		indexMapping := buildIndexMapping()
		index, err = bleve.New(indexPath, indexMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}
	} else {
		// Open existing index
		index, err = bleve.Open(indexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open index: %w", err)
		}
	}

	return &DescriptionIndex{
		index: index,
		path:  indexPath,
	}, nil
}

// buildFieldQuery creates a query for a specific field with multi-token support
// Combines MatchQuery (fuzzy, distance=1) + PrefixQuery for flexible matching
// For single token: returns DisjunctionQuery(MatchQuery OR PrefixQuery)
// For multiple tokens: returns ConjunctionQuery(AND) of DisjunctionQuery for each token
func buildFieldQuery(tokens []string, field string, boost float64) query.Query {
	if len(tokens) == 0 {
		// Empty query - return match nothing
		return bleve.NewMatchNoneQuery()
	}

	if len(tokens) == 1 {
		// Single token - combine fuzzy match + prefix match
		// MatchQuery: handles exact matches and typos (e.g., "tmeplate" → "template")
		matchQ := bleve.NewMatchQuery(tokens[0])
		matchQ.SetField(field)
		matchQ.SetFuzziness(1) // Allow 1 edit distance for typo tolerance

		// PrefixQuery: handles partial matches (e.g., "templa" → "template")
		prefixQ := bleve.NewPrefixQuery(tokens[0])
		prefixQ.SetField(field)

		// Combine with OR logic (either fuzzy or prefix match)
		disjunction := bleve.NewDisjunctionQuery(matchQ, prefixQ)
		disjunction.SetBoost(boost)
		return disjunction
	}

	// Multiple tokens - require ALL tokens (AND logic)
	tokenQueries := make([]query.Query, 0, len(tokens))
	for _, token := range tokens {
		// Each token gets fuzzy + prefix treatment
		matchQ := bleve.NewMatchQuery(token)
		matchQ.SetField(field)
		matchQ.SetFuzziness(1)

		prefixQ := bleve.NewPrefixQuery(token)
		prefixQ.SetField(field)

		// OR for this token
		tokenDisjunction := bleve.NewDisjunctionQuery(matchQ, prefixQ)
		tokenQueries = append(tokenQueries, tokenDisjunction)
	}

	// Combine with AND logic (all tokens must be present)
	conjunctionQuery := bleve.NewConjunctionQuery(tokenQueries...)
	conjunctionQuery.SetBoost(boost)
	return conjunctionQuery
}

// buildIndexMapping creates the index mapping for description documents
func buildIndexMapping() mapping.IndexMapping {
	indexMapping := bleve.NewIndexMapping()

	// Use standard analyzer (supports stemming and stop words)
	indexMapping.DefaultAnalyzer = standard.Name

	// Document mapping for project descriptions
	descMapping := bleve.NewDocumentMapping()

	// ProjectPath: searchable text field for fuzzy/partial matching
	pathFieldMapping := bleve.NewTextFieldMapping()
	pathFieldMapping.Analyzer = standard.Name
	pathFieldMapping.Store = true
	pathFieldMapping.Index = true
	descMapping.AddFieldMappingsAt("ProjectPath", pathFieldMapping)

	// ProjectName: text field (searchable)
	nameFieldMapping := bleve.NewTextFieldMapping()
	nameFieldMapping.Analyzer = standard.Name
	nameFieldMapping.Store = true
	nameFieldMapping.Index = true
	descMapping.AddFieldMappingsAt("ProjectName", nameFieldMapping)

	// Description: text field with full-text search
	descriptionFieldMapping := bleve.NewTextFieldMapping()
	descriptionFieldMapping.Analyzer = standard.Name
	descriptionFieldMapping.Store = true // Store for snippet extraction
	descriptionFieldMapping.Index = true
	descriptionFieldMapping.IncludeTermVectors = true // For better snippet highlighting
	descMapping.AddFieldMappingsAt("Description", descriptionFieldMapping)

	indexMapping.DefaultMapping = descMapping

	return indexMapping
}

// Add indexes a description document
func (di *DescriptionIndex) Add(projectPath, projectName, description string) error {
	doc := DescriptionDocument{
		ProjectPath: projectPath,
		ProjectName: projectName,
		Description: description,
	}

	return di.index.Index(projectPath, doc)
}

// AddBatch indexes multiple description documents in a batch
func (di *DescriptionIndex) AddBatch(docs []DescriptionDocument) error {
	batch := di.index.NewBatch()

	for _, doc := range docs {
		if err := batch.Index(doc.ProjectPath, doc); err != nil {
			return fmt.Errorf("failed to add document %s to batch: %w", doc.ProjectPath, err)
		}
	}

	return di.index.Batch(batch)
}

// Search performs a full-text search across ProjectName, ProjectPath, and Description
// Uses field boosting: ProjectName (5x), ProjectPath (2x), Description (1x)
// Supports multi-word queries with AND logic (all words must be present)
func (di *DescriptionIndex) Search(query string, maxResults int) ([]DescriptionMatch, error) {
	if query == "" {
		return []DescriptionMatch{}, nil
	}

	// Normalize query (lowercase for case-insensitive search)
	queryLower := strings.ToLower(query)

	// Split query into tokens for multi-word support
	tokens := strings.Fields(queryLower)

	// Build field queries with multi-token support
	// ProjectName: highest priority (10x boost)
	nameQuery := buildFieldQuery(tokens, "ProjectName", 10.0)

	// ProjectPath: medium priority (5x boost)
	pathQuery := buildFieldQuery(tokens, "ProjectPath", 5.0)

	// Description: lowest priority (1x boost)
	descQuery := buildFieldQuery(tokens, "Description", 1.0)

	// Add MatchQuery for description as fallback (tokenized full-text)
	descriptionMatch := bleve.NewMatchQuery(query)
	descriptionMatch.SetField("Description")
	descriptionMatch.SetBoost(1.0)

	// Combine with OR logic (disjunction)
	boolQuery := bleve.NewDisjunctionQuery(nameQuery, pathQuery, descQuery, descriptionMatch)

	searchRequest := bleve.NewSearchRequestOptions(boolQuery, maxResults, 0, false)

	// Request snippets for context
	searchRequest.Highlight = bleve.NewHighlight()
	searchRequest.Fields = []string{"ProjectPath", "ProjectName", "Description"}

	// Execute search
	searchResults, err := di.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results to DescriptionMatch
	matches := make([]DescriptionMatch, 0, len(searchResults.Hits))
	for _, hit := range searchResults.Hits {
		projectPath, _ := hit.Fields["ProjectPath"].(string)
		projectName, _ := hit.Fields["ProjectName"].(string)
		description, _ := hit.Fields["Description"].(string)

		// Extract snippet from highlight or description
		snippet := extractSnippet(hit)

		match := DescriptionMatch{
			Project: types.Project{
				Path:        projectPath,
				Name:        projectName,
				Description: description,
			},
			Score:   hit.Score,
			Snippet: snippet,
		}
		matches = append(matches, match)
	}

	return matches, nil
}

// extractSnippet extracts a relevant snippet from search hit
func extractSnippet(hit *search.DocumentMatch) string {
	// Try to get highlighted fragments first
	if len(hit.Fragments) > 0 && len(hit.Fragments["Description"]) > 0 {
		// Join first few fragments
		fragments := hit.Fragments["Description"]
		if len(fragments) > 2 {
			fragments = fragments[:2]
		}
		snippet := strings.Join(fragments, " ... ")
		// Strip HTML tags (Bleve adds <mark> tags for highlighting)
		return stripHTMLTags(snippet)
	}

	// Fallback: truncate description
	if description, ok := hit.Fields["Description"].(string); ok {
		if len(description) > 150 {
			return description[:150] + "..."
		}
		return description
	}

	return ""
}

// stripHTMLTags removes HTML tags from a string
func stripHTMLTags(s string) string {
	// Simple regex-free approach: remove everything between < and >
	var result strings.Builder
	inTag := false
	for _, ch := range s {
		if ch == '<' {
			inTag = true
		} else if ch == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// Delete removes a document from the index
func (di *DescriptionIndex) Delete(projectPath string) error {
	return di.index.Delete(projectPath)
}

// Count returns the number of indexed documents
func (di *DescriptionIndex) Count() (uint64, error) {
	return di.index.DocCount()
}

// Close closes the index
func (di *DescriptionIndex) Close() error {
	return di.index.Close()
}

// Exists checks if the index exists at the given path
func Exists(indexPath string) bool {
	_, err := os.Stat(indexPath)
	return !os.IsNotExist(err)
}

// GetAllProjects retrieves all projects from the index
// Returns all indexed projects (no pagination)
func (di *DescriptionIndex) GetAllProjects() ([]types.Project, error) {
	// Use match_all query to get everything
	query := bleve.NewMatchAllQuery()

	// Get doc count first to size the request appropriately
	count, err := di.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to get document count: %w", err)
	}

	if count == 0 {
		return []types.Project{}, nil
	}

	// Request all documents with bounds checking for integer overflow
	size := int(count)
	if count > math.MaxInt {
		size = math.MaxInt
	}
	searchRequest := bleve.NewSearchRequestOptions(query, size, 0, false)
	searchRequest.Fields = []string{"ProjectPath", "ProjectName", "Description"}

	// Execute search
	searchResults, err := di.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results to Project slice
	projects := make([]types.Project, 0, len(searchResults.Hits))
	for _, hit := range searchResults.Hits {
		projectPath, _ := hit.Fields["ProjectPath"].(string)
		projectName, _ := hit.Fields["ProjectName"].(string)
		description, _ := hit.Fields["Description"].(string)

		projects = append(projects, types.Project{
			Path:        projectPath,
			Name:        projectName,
			Description: description,
		})
	}

	return projects, nil
}
