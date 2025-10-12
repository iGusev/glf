// Package history manages selection frequency tracking with exponential decay
package history

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// halfLifeDays is the number of days for score to decay to 50%
	halfLifeDays = 30.0
	// maxAgeDays is the maximum age for history entries (older entries are ignored/cleaned)
	maxAgeDays = 100.0
	// decayLambda is the decay constant: ln(2) / half_life
	decayLambda = 0.693147 / halfLifeDays // ≈ 0.0231
)

// SelectionInfo tracks information about a selected item
type SelectionInfo struct {
	Count    int       // Number of times selected
	LastUsed time.Time // Last time selected
}

// historyData is the serializable representation of history
type historyData struct {
	Selections      map[string]SelectionInfo
	QuerySelections map[string]map[string]SelectionInfo
}

// History manages selection frequency tracking
type History struct {
	selections      map[string]SelectionInfo            // Global history: projectPath -> info
	querySelections map[string]map[string]SelectionInfo // Query-specific: queryHash -> projectPath -> info
	mu              sync.RWMutex
	filePath        string
	dirty           bool // Indicates if there are unsaved changes
}

// New creates a new History instance with the given file path
func New(filePath string) *History {
	return &History{
		selections:      make(map[string]SelectionInfo),
		querySelections: make(map[string]map[string]SelectionInfo),
		filePath:        filePath,
		dirty:           false,
	}
}

// LoadAsync loads history from disk asynchronously
// Returns a channel that will receive an error (or nil on success)
func (h *History) LoadAsync() <-chan error {
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		// Clean path to prevent directory traversal
		cleanPath := filepath.Clean(h.filePath)
		file, err := os.Open(cleanPath)
		if err != nil {
			if os.IsNotExist(err) {
				// First run - no history file yet, not an error
				errCh <- nil
				return
			}
			errCh <- fmt.Errorf("failed to open history file: %w", err)
			return
		}
		defer file.Close() //nolint:errcheck // Deferred close on read-only file

		decoder := gob.NewDecoder(file)

		h.mu.Lock()
		defer h.mu.Unlock()

		// Try to decode new format first
		var data historyData
		if err := decoder.Decode(&data); err != nil {
			// Failed - might be old format, try decoding just selections map
			_, _ = file.Seek(0, 0) //nolint:errcheck // Reset to beginning; ignore error (best effort)
			decoder = gob.NewDecoder(file)

			var oldSelections map[string]SelectionInfo
			if err := decoder.Decode(&oldSelections); err != nil {
				errCh <- fmt.Errorf("failed to decode history: %w", err)
				return
			}

			// Migrate old format to new
			h.selections = oldSelections
			h.querySelections = make(map[string]map[string]SelectionInfo)
		} else {
			// New format loaded successfully
			h.selections = data.Selections
			if data.QuerySelections != nil {
				h.querySelections = data.QuerySelections
			} else {
				h.querySelections = make(map[string]map[string]SelectionInfo)
			}
		}

		h.dirty = false

		// Cleanup old entries (older than maxAgeDays)
		// This is done in the loading goroutine to avoid blocking
		h.mu.Unlock()
		removed := h.CleanupOldEntries()
		h.mu.Lock()

		if removed > 0 {
			// Save after cleanup to persist the changes
			go func() {
				if err := h.Save(); err != nil {
					// Can't use logger here as it may not be initialized
					// Silently fail - this is best-effort background cleanup
				}
			}()
		}

		errCh <- nil
	}()

	return errCh
}

// RecordSelection records a selection of the given item
func (h *History) RecordSelection(item string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	info := h.selections[item]
	info.Count++
	info.LastUsed = time.Now()
	h.selections[item] = info
	h.dirty = true
}

// calculateDecayMultiplier returns the exponential decay multiplier for the given age
// Uses formula: e^(-λt) where λ = ln(2) / half_life
// Returns 0 for entries older than maxAgeDays
func calculateDecayMultiplier(daysSinceLastUse float64) float64 {
	if daysSinceLastUse > maxAgeDays {
		return 0.0 // Ignore very old entries
	}
	// Exponential decay: e^(-λt)
	return math.Exp(-decayLambda * daysSinceLastUse)
}

// GetScore returns the frequency score for an item with exponential decay
// Higher score = more frequently selected and more recent
// Entries older than 100 days return 0
func (h *History) GetScore(item string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	info, exists := h.selections[item]
	if !exists {
		return 0
	}

	daysSinceLastUse := time.Since(info.LastUsed).Hours() / 24

	// Apply exponential decay
	decayMultiplier := calculateDecayMultiplier(daysSinceLastUse)
	if decayMultiplier == 0 {
		return 0
	}

	// Base score from frequency with exponential decay
	// Reduced by 10x to prevent dominating search relevance
	score := float64(info.Count*10) * decayMultiplier

	return int(score)
}

// GetAllScores returns a map of all items to their scores with exponential decay
func (h *History) GetAllScores() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	scores := make(map[string]int, len(h.selections))
	for item := range h.selections {
		info := h.selections[item]
		daysSinceLastUse := time.Since(info.LastUsed).Hours() / 24

		// Apply exponential decay
		decayMultiplier := calculateDecayMultiplier(daysSinceLastUse)
		if decayMultiplier == 0 {
			continue // Skip very old entries
		}

		// Base score from frequency with exponential decay
		// Reduced by 10x to prevent dominating search relevance
		score := float64(info.Count*10) * decayMultiplier
		scores[item] = int(score)
	}

	return scores
}

// Save saves the history to disk
func (h *History) Save() error {
	h.mu.RLock()
	if !h.dirty {
		h.mu.RUnlock()
		return nil // No changes to save
	}
	h.mu.RUnlock()

	// Clean path to prevent directory traversal
	cleanPath := filepath.Clean(h.filePath)

	// Create directory if it doesn't exist
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create history directory: %w", err)
	}

	// Create temporary file for atomic write
	tempPath := cleanPath + ".tmp"
	// #nosec G304 -- Path constructed with filepath.Clean(configPath) + ".tmp"
	// User controls config dir in their own config file - not a security issue:
	// 1. Base path is cleaned with filepath.Clean to prevent traversal
	// 2. Only ".tmp" extension is appended (fixed suffix, not user-controlled)
	// 3. No privilege escalation (runs with user's own permissions)
	// 4. Used for atomic write pattern (temp file + rename)
	file, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	encoder := gob.NewEncoder(file)

	h.mu.RLock()
	data := historyData{
		Selections:      h.selections,
		QuerySelections: h.querySelections,
	}
	err = encoder.Encode(data)
	h.mu.RUnlock()

	if err != nil {
		_ = file.Close()        //nolint:errcheck // Cleanup on error; ignore Close error
		_ = os.Remove(tempPath) //nolint:errcheck // Cleanup temp file; ignore Remove error
		return fmt.Errorf("failed to encode history: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tempPath) //nolint:errcheck // Cleanup temp file; ignore Remove error
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, cleanPath); err != nil {
		_ = os.Remove(tempPath) //nolint:errcheck // Cleanup temp file; ignore Remove error
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	h.mu.Lock()
	h.dirty = false
	h.mu.Unlock()

	return nil
}

// Stats returns statistics about the history
func (h *History) Stats() (totalSelections int, uniqueItems int) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	uniqueItems = len(h.selections)
	for _, info := range h.selections {
		totalSelections += info.Count
	}

	return totalSelections, uniqueItems
}

// Clear removes all history
func (h *History) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.selections = make(map[string]SelectionInfo)
	h.querySelections = make(map[string]map[string]SelectionInfo)
	h.dirty = true
}

// CleanupOldEntries removes history entries older than maxAgeDays
// This helps keep the history file size manageable and removes stale data
func (h *History) CleanupOldEntries() int {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()
	removed := 0

	// Clean global selections
	for item, info := range h.selections {
		daysSinceLastUse := now.Sub(info.LastUsed).Hours() / 24
		if daysSinceLastUse > maxAgeDays {
			delete(h.selections, item)
			removed++
		}
	}

	// Clean query-specific selections
	for queryHash, querySelections := range h.querySelections {
		for item, info := range querySelections {
			daysSinceLastUse := now.Sub(info.LastUsed).Hours() / 24
			if daysSinceLastUse > maxAgeDays {
				delete(querySelections, item)
				removed++
			}
		}
		// Remove empty query hashes
		if len(querySelections) == 0 {
			delete(h.querySelections, queryHash)
		}
	}

	if removed > 0 {
		h.dirty = true
	}

	return removed
}

// normalizeQuery normalizes a query string for consistent history tracking
func normalizeQuery(query string) string {
	// Lowercase, trim whitespace, collapse multiple spaces
	normalized := strings.ToLower(strings.TrimSpace(query))
	normalized = strings.Join(strings.Fields(normalized), " ")

	// Hash the normalized query for compact storage
	hash := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes (16 hex chars)
}

// RecordSelectionWithQuery records a selection with query context
func (h *History) RecordSelectionWithQuery(query, item string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	now := time.Now()

	// Update global history
	globalInfo := h.selections[item]
	globalInfo.Count++
	globalInfo.LastUsed = now
	h.selections[item] = globalInfo

	// Update query-specific history
	if query != "" {
		queryHash := normalizeQuery(query)

		if h.querySelections[queryHash] == nil {
			h.querySelections[queryHash] = make(map[string]SelectionInfo)
		}

		queryInfo := h.querySelections[queryHash][item]
		queryInfo.Count++
		queryInfo.LastUsed = now
		h.querySelections[queryHash][item] = queryInfo
	}

	h.dirty = true
}

// GetScoreForQuery returns the score for an item considering query-specific history with exponential decay
// Query-specific selections get a significant boost (3x multiplier)
func (h *History) GetScoreForQuery(query, item string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	totalScore := 0.0

	// Base global score with exponential decay
	// Reduced by 10x to prevent dominating search relevance
	if info, exists := h.selections[item]; exists {
		daysSinceLastUse := time.Since(info.LastUsed).Hours() / 24
		decayMultiplier := calculateDecayMultiplier(daysSinceLastUse)
		if decayMultiplier > 0 {
			totalScore += float64(info.Count*10) * decayMultiplier
		}
	}

	// Query-specific boost (3x multiplier) with exponential decay
	// Reduced by 10x to prevent dominating search relevance
	if query != "" {
		queryHash := normalizeQuery(query)
		if querySelections, exists := h.querySelections[queryHash]; exists {
			if info, exists := querySelections[item]; exists {
				daysSinceLastUse := time.Since(info.LastUsed).Hours() / 24
				decayMultiplier := calculateDecayMultiplier(daysSinceLastUse)
				if decayMultiplier > 0 {
					totalScore += float64(info.Count*30) * decayMultiplier
				}
			}
		}
	}

	return int(totalScore)
}

// GetAllScoresForQuery returns scores for all items, boosted by query-specific history with exponential decay
func (h *History) GetAllScoresForQuery(query string) map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	scores := make(map[string]float64)

	// Add global scores with exponential decay
	// Reduced by 10x to prevent dominating search relevance
	for item := range h.selections {
		info := h.selections[item]
		daysSinceLastUse := time.Since(info.LastUsed).Hours() / 24
		decayMultiplier := calculateDecayMultiplier(daysSinceLastUse)
		if decayMultiplier > 0 {
			scores[item] = float64(info.Count*10) * decayMultiplier
		}
	}

	// Add query-specific boosts with exponential decay
	// Reduced by 10x to prevent dominating search relevance
	if query != "" {
		queryHash := normalizeQuery(query)
		if querySelections, exists := h.querySelections[queryHash]; exists {
			for item, info := range querySelections {
				daysSinceLastUse := time.Since(info.LastUsed).Hours() / 24
				decayMultiplier := calculateDecayMultiplier(daysSinceLastUse)
				if decayMultiplier > 0 {
					scores[item] += float64(info.Count*30) * decayMultiplier
				}
			}
		}
	}

	// Convert to int map
	intScores := make(map[string]int, len(scores))
	for item, score := range scores {
		intScores[item] = int(score)
	}

	return intScores
}
