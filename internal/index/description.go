// Package index provides full-text search indexing for project descriptions using Bleve
package index

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/standard"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/igusev/glf/internal/model"
)

const (
	// IndexVersion is the current version of the index schema
	// Increment this when making breaking changes to the index structure
	IndexVersion = 4 // Version 4: Added Member field

	// Version metadata document ID (reserved, never used for actual projects)
	versionDocID = "__index_version__"
)

// ErrIndexVersionMismatch indicates the index schema version is incompatible
var ErrIndexVersionMismatch = errors.New("index version mismatch")

// DescriptionIndex manages the bleve index for project descriptions
type DescriptionIndex struct {
	index bleve.Index
	path  string
}

// versionDocument stores the index schema version
type versionDocument struct {
	Version int `json:"version"`
}

// NewDescriptionIndex creates or opens a description index
// Returns ErrIndexVersionMismatch if existing index has incompatible version
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

		// Store version in new index
		versionDoc := versionDocument{Version: IndexVersion}
		if err := index.Index(versionDocID, versionDoc); err != nil {
			_ = index.Close() // Ignore close error on error path
			return nil, fmt.Errorf("failed to store index version: %w", err)
		}
	} else {
		// Open existing index
		index, err = bleve.Open(indexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open index: %w", err)
		}

		// Check version compatibility by searching for version document
		searchReq := bleve.NewSearchRequest(bleve.NewDocIDQuery([]string{versionDocID}))
		searchReq.Fields = []string{"version"}
		searchRes, err := index.Search(searchReq)
		if err != nil || len(searchRes.Hits) == 0 {
			// Old index without version metadata (version 1)
			_ = index.Close() // Ignore close error on error path
			return nil, fmt.Errorf("%w: index created before versioning was added", ErrIndexVersionMismatch)
		}

		// Extract version number from search result
		storedVersion := 0
		if versionField, ok := searchRes.Hits[0].Fields["version"].(float64); ok {
			storedVersion = int(versionField)
		}

		if storedVersion == 0 {
			// Couldn't determine version - assume old
			_ = index.Close() // Ignore close error on error path
			return nil, fmt.Errorf("%w: could not determine index version", ErrIndexVersionMismatch)
		}

		if storedVersion != IndexVersion {
			_ = index.Close() // Ignore close error on error path
			return nil, fmt.Errorf("%w: index version %d, current version %d",
				ErrIndexVersionMismatch, storedVersion, IndexVersion)
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

	// Starred: boolean field (not searchable, just stored)
	starredFieldMapping := bleve.NewBooleanFieldMapping()
	starredFieldMapping.Store = true
	starredFieldMapping.Index = false // No need to search by this
	descMapping.AddFieldMappingsAt("Starred", starredFieldMapping)

	// Archived: boolean field (not searchable, just stored)
	archivedFieldMapping := bleve.NewBooleanFieldMapping()
	archivedFieldMapping.Store = true
	archivedFieldMapping.Index = false // No need to search by this
	descMapping.AddFieldMappingsAt("Archived", archivedFieldMapping)

	// Member: boolean field (not searchable, just stored)
	memberFieldMapping := bleve.NewBooleanFieldMapping()
	memberFieldMapping.Store = true
	memberFieldMapping.Index = false // No need to search by this
	descMapping.AddFieldMappingsAt("Member", memberFieldMapping)

	indexMapping.DefaultMapping = descMapping

	return indexMapping
}

// Add indexes a description document
func (di *DescriptionIndex) Add(projectPath, projectName, description string, starred, archived bool) error {
	doc := DescriptionDocument{
		ProjectPath: projectPath,
		ProjectName: projectName,
		Description: description,
		Starred:     starred,
		Archived:    archived,
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
	searchRequest.Fields = []string{"ProjectPath", "ProjectName", "Description", "Starred", "Archived", "Member"}

	// Execute search
	searchResults, err := di.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results to DescriptionMatch
	matches := make([]DescriptionMatch, 0, len(searchResults.Hits))
	for _, hit := range searchResults.Hits {
		projectPath, ok := hit.Fields["ProjectPath"].(string)
		if !ok {
			projectPath = ""
		}
		projectName, ok := hit.Fields["ProjectName"].(string)
		if !ok {
			projectName = ""
		}
		description, ok := hit.Fields["Description"].(string)
		if !ok {
			description = ""
		}
		starred, ok := hit.Fields["Starred"].(bool)
		if !ok {
			starred = false
		}
		archived, ok := hit.Fields["Archived"].(bool)
		if !ok {
			archived = false
		}
		member, ok := hit.Fields["Member"].(bool)
		if !ok {
			member = false
		}

		// Extract snippet from highlight or description
		snippet := extractSnippet(hit)

		match := DescriptionMatch{
			Project: model.Project{
				Path:        projectPath,
				Name:        projectName,
				Description: description,
				Starred:     starred,
				Archived:    archived,
				Member:      member,
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

// NewDescriptionIndexWithAutoRecreate creates or opens a description index
// Automatically recreates the index if version mismatch is detected
func NewDescriptionIndexWithAutoRecreate(indexPath string) (*DescriptionIndex, bool, error) {
	descIndex, err := NewDescriptionIndex(indexPath)
	if err != nil {
		// Check if this is a version mismatch error
		if errors.Is(err, ErrIndexVersionMismatch) {
			// Delete old index
			if err := os.RemoveAll(indexPath); err != nil {
				return nil, false, fmt.Errorf("failed to remove old index: %w", err)
			}

			// Create new index with current version
			descIndex, err = NewDescriptionIndex(indexPath)
			if err != nil {
				return nil, false, fmt.Errorf("failed to create new index after version mismatch: %w", err)
			}

			// Return with recreated flag = true
			return descIndex, true, nil
		}

		// Other error - propagate
		return nil, false, err
	}

	// Successfully opened existing index
	return descIndex, false, nil
}

// GetAllProjects retrieves all projects from the index
// Returns all indexed projects (no pagination)
func (di *DescriptionIndex) GetAllProjects() ([]model.Project, error) {
	// Use match_all query to get everything
	query := bleve.NewMatchAllQuery()

	// Get doc count first to size the request appropriately
	count, err := di.Count()
	if err != nil {
		return nil, fmt.Errorf("failed to get document count: %w", err)
	}

	if count == 0 {
		return []model.Project{}, nil
	}

	// Request all documents with bounds checking for integer overflow
	size := int(count)
	if count > math.MaxInt {
		size = math.MaxInt
	}
	searchRequest := bleve.NewSearchRequestOptions(query, size, 0, false)
	searchRequest.Fields = []string{"ProjectPath", "ProjectName", "Description", "Starred", "Archived", "Member"}

	// Execute search
	searchResults, err := di.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Convert results to Project slice (filtering out version document)
	projects := make([]model.Project, 0, len(searchResults.Hits))
	for _, hit := range searchResults.Hits {
		// Skip version document (it has ID __index_version__ and no ProjectPath)
		if hit.ID == versionDocID {
			continue
		}

		projectPath, ok := hit.Fields["ProjectPath"].(string)
		if !ok {
			projectPath = ""
		}
		projectName, ok := hit.Fields["ProjectName"].(string)
		if !ok {
			projectName = ""
		}
		description, ok := hit.Fields["Description"].(string)
		if !ok {
			description = ""
		}
		starred, ok := hit.Fields["Starred"].(bool)
		if !ok {
			starred = false
		}
		archived, ok := hit.Fields["Archived"].(bool)
		if !ok {
			archived = false
		}
		member, ok := hit.Fields["Member"].(bool)
		if !ok {
			member = false
		}

		projects = append(projects, model.Project{
			Path:        projectPath,
			Name:        projectName,
			Description: description,
			Starred:     starred,
			Archived:    archived,
			Member:      member,
		})
	}

	return projects, nil
}
