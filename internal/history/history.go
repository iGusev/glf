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
	Timestamps []time.Time // All selection timestamps (for accurate decay calculation)
}

// historyData is the serializable representation of history
type historyData struct {
	Selections      map[string]SelectionInfo
	QuerySelections map[string]map[string]SelectionInfo
}

// History manages selection frequency tracking
type History struct {
	mu              sync.RWMutex
	selections      map[string]SelectionInfo            // Global history: projectPath -> info
	querySelections map[string]map[string]SelectionInfo // Query-specific: queryHash -> projectPath -> info
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

// oldSelectionInfo is the previous format for migration
type oldSelectionInfo struct {
	LastUsed time.Time
	Count    int
}

// migrateOldSelection converts old format to new format
func migrateOldSelection(old oldSelectionInfo) SelectionInfo {
	// Create timestamps array with all clicks at LastUsed time
	// This is a best-effort migration - we don't have actual individual timestamps
	timestamps := make([]time.Time, old.Count)
	for i := 0; i < old.Count; i++ {
		timestamps[i] = old.LastUsed
	}
	return SelectionInfo{Timestamps: timestamps}
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
		defer func() {
			if err := file.Close(); err != nil {
				// Ignore close error in async load
				_ = err
			}
		}()

		decoder := gob.NewDecoder(file)

		h.mu.Lock()
		defer h.mu.Unlock()

		// Try to decode new format first
		var data historyData
		if err := decoder.Decode(&data); err != nil {
			// Failed - might be old format, try decoding
			if _, seekErr := file.Seek(0, 0); seekErr != nil {
				// Can't seek - corrupt file, start fresh
				h.selections = make(map[string]SelectionInfo)
				h.querySelections = make(map[string]map[string]SelectionInfo)
				h.dirty = true
				errCh <- nil
				return
			}
			decoder = gob.NewDecoder(file)

			// Try old historyData format
			type oldHistoryData struct {
				Selections      map[string]oldSelectionInfo
				QuerySelections map[string]map[string]oldSelectionInfo
			}

			var oldData oldHistoryData
			if err := decoder.Decode(&oldData); err != nil {
				// Try even older format (just map)
				if _, seekErr := file.Seek(0, 0); seekErr != nil {
					// Can't seek - corrupt file, start fresh
					h.selections = make(map[string]SelectionInfo)
					h.querySelections = make(map[string]map[string]SelectionInfo)
					h.dirty = true
					errCh <- nil
					return
				}
				decoder = gob.NewDecoder(file)

				var veryOldSelections map[string]oldSelectionInfo
				if err := decoder.Decode(&veryOldSelections); err != nil {
					// All formats failed - corrupt file, start fresh
					h.selections = make(map[string]SelectionInfo)
					h.querySelections = make(map[string]map[string]SelectionInfo)
					h.dirty = true
					errCh <- nil
					return
				}

				// Migrate very old format to new
				h.selections = make(map[string]SelectionInfo)
				for item, oldInfo := range veryOldSelections {
					h.selections[item] = migrateOldSelection(oldInfo)
				}
				h.querySelections = make(map[string]map[string]SelectionInfo)
			} else {
				// Migrate old historyData format to new
				h.selections = make(map[string]SelectionInfo)
				for item, oldInfo := range oldData.Selections {
					h.selections[item] = migrateOldSelection(oldInfo)
				}
				h.querySelections = make(map[string]map[string]SelectionInfo)
				for queryHash, oldQuerySelections := range oldData.QuerySelections {
					h.querySelections[queryHash] = make(map[string]SelectionInfo)
					for item, oldInfo := range oldQuerySelections {
						h.querySelections[queryHash][item] = migrateOldSelection(oldInfo)
					}
				}
			}
			h.dirty = true // Mark dirty to trigger save with new format
		} else {
			// New format loaded successfully
			h.selections = data.Selections
			if data.QuerySelections != nil {
				h.querySelections = data.QuerySelections
			} else {
				h.querySelections = make(map[string]map[string]SelectionInfo)
			}
			h.dirty = false
		}

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
					_ = err // explicitly ignore error
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
	info.Timestamps = append(info.Timestamps, time.Now())
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
// Each timestamp contributes independently to the score
// Entries older than 100 days return 0
func (h *History) GetScore(item string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	info, exists := h.selections[item]
	if !exists {
		return 0
	}

	score := 0.0
	now := time.Now()

	// Sum decay-adjusted scores for each timestamp
	for _, timestamp := range info.Timestamps {
		daysSinceUse := now.Sub(timestamp).Hours() / 24
		decayMultiplier := calculateDecayMultiplier(daysSinceUse)
		if decayMultiplier > 0 {
			score += 1.0 * decayMultiplier
		}
	}

	// Cap at 30 to prevent extreme dominance
	const maxHistoryScore = 30
	if score > maxHistoryScore {
		score = maxHistoryScore
	}

	return int(score)
}

// GetAllScores returns a map of all items to their scores with exponential decay
func (h *History) GetAllScores() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	scores := make(map[string]int, len(h.selections))
	now := time.Now()

	for item, info := range h.selections {
		score := 0.0

		// Sum decay-adjusted scores for each timestamp
		for _, timestamp := range info.Timestamps {
			daysSinceUse := now.Sub(timestamp).Hours() / 24
			decayMultiplier := calculateDecayMultiplier(daysSinceUse)
			if decayMultiplier > 0 {
				score += 1.0 * decayMultiplier
			}
		}

		// Skip if score is 0 (all timestamps too old)
		if score == 0 {
			continue
		}

		// Cap at 30 to prevent extreme dominance
		const maxHistoryScore = 30
		if score > maxHistoryScore {
			score = maxHistoryScore
		}

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
		if closeErr := file.Close(); closeErr != nil {
			// Ignore close error on error path
			_ = closeErr
		}
		if removeErr := os.Remove(tempPath); removeErr != nil {
			// Ignore remove error on error path
			_ = removeErr
		}
		return fmt.Errorf("failed to encode history: %w", err)
	}

	if err := file.Close(); err != nil {
		if removeErr := os.Remove(tempPath); removeErr != nil {
			// Ignore remove error on error path
			_ = removeErr
		}
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, cleanPath); err != nil {
		if removeErr := os.Remove(tempPath); removeErr != nil {
			// Ignore remove error on error path
			_ = removeErr
		}
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
		totalSelections += len(info.Timestamps)
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
		// Filter out old timestamps
		validTimestamps := make([]time.Time, 0, len(info.Timestamps))
		for _, timestamp := range info.Timestamps {
			daysSinceUse := now.Sub(timestamp).Hours() / 24
			if daysSinceUse <= maxAgeDays {
				validTimestamps = append(validTimestamps, timestamp)
			} else {
				removed++
			}
		}

		// If no valid timestamps left, remove the item entirely
		if len(validTimestamps) == 0 {
			delete(h.selections, item)
		} else if len(validTimestamps) < len(info.Timestamps) {
			// Update with filtered timestamps
			h.selections[item] = SelectionInfo{Timestamps: validTimestamps}
		}
	}

	// Clean query-specific selections
	for queryHash, querySelections := range h.querySelections {
		for item, info := range querySelections {
			// Filter out old timestamps
			validTimestamps := make([]time.Time, 0, len(info.Timestamps))
			for _, timestamp := range info.Timestamps {
				daysSinceUse := now.Sub(timestamp).Hours() / 24
				if daysSinceUse <= maxAgeDays {
					validTimestamps = append(validTimestamps, timestamp)
				} else {
					removed++
				}
			}

			// If no valid timestamps left, remove the item entirely
			if len(validTimestamps) == 0 {
				delete(querySelections, item)
			} else if len(validTimestamps) < len(info.Timestamps) {
				// Update with filtered timestamps
				querySelections[item] = SelectionInfo{Timestamps: validTimestamps}
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
	globalInfo.Timestamps = append(globalInfo.Timestamps, now)
	h.selections[item] = globalInfo

	// Update query-specific history
	if query != "" {
		queryHash := normalizeQuery(query)

		if h.querySelections[queryHash] == nil {
			h.querySelections[queryHash] = make(map[string]SelectionInfo)
		}

		queryInfo := h.querySelections[queryHash][item]
		queryInfo.Timestamps = append(queryInfo.Timestamps, now)
		h.querySelections[queryHash][item] = queryInfo
	}

	h.dirty = true
}

// GetScoreForQuery returns the score for an item considering query-specific history with exponential decay
// Query-specific selections get a moderate boost (2.5x multiplier over global)
func (h *History) GetScoreForQuery(query, item string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	totalScore := 0.0
	now := time.Now()

	// Base global score with exponential decay
	if info, exists := h.selections[item]; exists {
		for _, timestamp := range info.Timestamps {
			daysSinceUse := now.Sub(timestamp).Hours() / 24
			decayMultiplier := calculateDecayMultiplier(daysSinceUse)
			if decayMultiplier > 0 {
				totalScore += 1.0 * decayMultiplier
			}
		}
	}

	// Query-specific boost (2.5x multiplier) with exponential decay
	if query != "" {
		queryHash := normalizeQuery(query)
		if querySelections, exists := h.querySelections[queryHash]; exists {
			if info, exists := querySelections[item]; exists {
				for _, timestamp := range info.Timestamps {
					daysSinceUse := now.Sub(timestamp).Hours() / 24
					decayMultiplier := calculateDecayMultiplier(daysSinceUse)
					if decayMultiplier > 0 {
						totalScore += 2.5 * decayMultiplier
					}
				}
			}
		}
	}

	// Cap at 30 to prevent extreme dominance
	const maxHistoryScore = 30
	if totalScore > maxHistoryScore {
		totalScore = maxHistoryScore
	}

	return int(totalScore)
}

// GetAllScoresForQuery returns scores for all items, boosted by query-specific history with exponential decay
func (h *History) GetAllScoresForQuery(query string) map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	scores := make(map[string]float64)
	now := time.Now()

	// Add global scores with exponential decay
	for item, info := range h.selections {
		for _, timestamp := range info.Timestamps {
			daysSinceUse := now.Sub(timestamp).Hours() / 24
			decayMultiplier := calculateDecayMultiplier(daysSinceUse)
			if decayMultiplier > 0 {
				scores[item] += 1.0 * decayMultiplier
			}
		}
	}

	// Add query-specific boosts with exponential decay
	if query != "" {
		queryHash := normalizeQuery(query)
		if querySelections, exists := h.querySelections[queryHash]; exists {
			for item, info := range querySelections {
				for _, timestamp := range info.Timestamps {
					daysSinceUse := now.Sub(timestamp).Hours() / 24
					decayMultiplier := calculateDecayMultiplier(daysSinceUse)
					if decayMultiplier > 0 {
						scores[item] += 2.5 * decayMultiplier
					}
				}
			}
		}
	}

	// Cap at 30 to prevent extreme dominance
	const maxHistoryScore = 30
	for item, score := range scores {
		if score > maxHistoryScore {
			scores[item] = maxHistoryScore
		}
	}

	// Convert to int map
	intScores := make(map[string]int, len(scores))
	for item, score := range scores {
		intScores[item] = int(score)
	}

	return intScores
}

// Entry represents a single history entry for display
type Entry struct {
	ProjectPath string
	Count       int
	LastUsed    time.Time
	Score       int
}

// GetAllEntries returns all history entries sorted by score (highest first)
func (h *History) GetAllEntries() []Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	entries := make([]Entry, 0, len(h.selections))
	now := time.Now()

	for item, info := range h.selections {
		if len(info.Timestamps) == 0 {
			continue
		}

		// Calculate score
		score := 0.0
		for _, timestamp := range info.Timestamps {
			daysSinceUse := now.Sub(timestamp).Hours() / 24
			decayMultiplier := calculateDecayMultiplier(daysSinceUse)
			if decayMultiplier > 0 {
				score += 1.0 * decayMultiplier
			}
		}

		// Skip if score is 0 (all timestamps too old)
		if score == 0 {
			continue
		}

		// Cap at 30
		const maxHistoryScore = 30
		if score > maxHistoryScore {
			score = maxHistoryScore
		}

		// Find last used time (most recent timestamp)
		lastUsed := info.Timestamps[0]
		for _, t := range info.Timestamps {
			if t.After(lastUsed) {
				lastUsed = t
			}
		}

		entries = append(entries, Entry{
			ProjectPath: item,
			Count:       len(info.Timestamps),
			LastUsed:    lastUsed,
			Score:       int(score),
		})
	}

	// Sort by score descending (highest first)
	// Using simple bubble sort for small datasets
	for i := 0; i < len(entries)-1; i++ {
		for j := 0; j < len(entries)-i-1; j++ {
			if entries[j].Score < entries[j+1].Score {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			}
		}
	}

	return entries
}
