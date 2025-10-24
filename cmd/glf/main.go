package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/igusev/glf/internal/cache"
	"github.com/igusev/glf/internal/config"
	"github.com/igusev/glf/internal/gitlab"
	"github.com/igusev/glf/internal/history"
	"github.com/igusev/glf/internal/index"
	"github.com/igusev/glf/internal/logger"
	"github.com/igusev/glf/internal/model"
	"github.com/igusev/glf/internal/search"
	"github.com/igusev/glf/internal/tui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Build-time variables (set via ldflags)
var (
	version   = "dev"     // Version from git tag or "dev"
	commit    = "unknown" // Git commit hash (used in version output)
	buildTime = "unknown" // Build timestamp (used in version output)
)

// Sync mode constants
const (
	syncModeFull        = "full"
	syncModeIncremental = "incremental"
	responseYes         = "yes"
)

// Platform constants for runtime.GOOS
const (
	platformDarwin  = "darwin"
	platformLinux   = "linux"
	platformWindows = "windows"
)

// JSON output structures for API integrations
type (
	// JSONSearchResult represents the complete search response in JSON mode
	JSONSearchResult struct {
		Query   string        `json:"query"`   // Search query that was executed
		Results []JSONProject `json:"results"` // Matching projects
		Total   int           `json:"total"`   // Total number of results
		Limit   int           `json:"limit"`   // Maximum results returned
	}

	// JSONProject represents a single project in JSON output
	JSONProject struct {
		Path        string  `json:"path"`            // Project path (e.g., "group/project")
		Name        string  `json:"name"`            // Project name
		Description string  `json:"description"`     // Project description
		URL         string  `json:"url"`             // Full project URL
		Starred     bool    `json:"starred"`         // Whether the project is starred by the user
		Excluded    bool    `json:"excluded"`        // Whether the project is excluded via config
		Archived    bool    `json:"archived"`        // Whether the project is archived
		Member      bool    `json:"member"`          // Whether the user is a member of this project
		Score       float64 `json:"score,omitempty"` // Relevance score (optional, with --scores)
	}

	// JSONError represents an error response in JSON mode
	JSONError struct {
		Error string `json:"error"` // Error message
	}
)

var (
	verbose      bool   // Flag to enable verbose logging
	showScores   bool   // Flag to show score breakdown (search + history)
	autoGo       bool   // Flag to automatically select first result and open in browser
	doSync       bool   // Flag to perform sync instead of search
	forceFull    bool   // Flag to force full sync (ignore incremental)
	doInit       bool   // Flag to run interactive configuration wizard
	resetFlag    bool   // Flag to reset configuration and start from scratch
	jsonOutput   bool   // Flag to enable JSON output mode for API integrations
	limitResults int    // Flag to limit number of results in JSON mode
	showHistory  bool   // Flag to display search history
	clearHistory bool   // Flag to clear search history
	showHidden   bool   // Flag to show hidden projects (excluded, archived, non-member) - affects TUI initial state and JSON output
	jsonRecord   string // Flag to record project selection in history (for JSON integrations like Raycast)
	queryContext string // Flag to provide query context when recording selection
)

var rootCmd = &cobra.Command{
	Use:   "glf [flags] [query...]",
	Short: "GitLab Fuzzy Finder - Fast project search for self-hosted GitLab",
	Long: `glf is a CLI tool that provides instant fuzzy search across your GitLab projects.
It uses a local cache for blazing-fast performance.

Getting Started:
  1. Create config: ~/.config/glf/config.yaml
  2. Run: glf --sync (to fetch projects)
  3. Run: glf (interactive mode) or glf <query> (direct search)

Examples:
  glf                  # Interactive fuzzy finder
  glf backend          # Direct search for "backend"
  glf api ingress      # Multi-word search for "api ingress"
  glf .                # Open current Git repository in browser
  glf sync             # Search for "sync" (not a command!)
  glf --sync           # Synchronize projects cache
  glf --sync --full    # Force full sync
  glf -g api           # Auto-select first result and open in browser

Configuration:
  Set your GitLab URL and token in ~/.config/glf/config.yaml or via environment:
    GLF_GITLAB_URL=https://gitlab.example.com
    GLF_GITLAB_TOKEN=your-token-here`,
	RunE: runSearch,
	// Accept any number of arguments as search query
	Args: cobra.ArbitraryArgs,
	// Don't suggest commands when args don't match subcommands
	SuggestionsMinimumDistance: 2,
}

// runSearch handles the default search behavior
func runSearch(cmd *cobra.Command, args []string) error {
	// Handle --init flag first (before loading config)
	if doInit {
		return runConfigWizard()
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Handle --history flag (show history and exit)
	if showHistory {
		return runShowHistory(cfg)
	}

	// Handle --clear-history flag (clear history and exit)
	if clearHistory {
		return runClearHistory(cfg)
	}

	// Handle --json-record flag (record selection in history and exit)
	if jsonRecord != "" {
		return runRecordSelection(cfg, jsonRecord, queryContext)
	}

	// Handle "glf ." - open current Git repository
	if len(args) == 1 && args[0] == "." {
		return runOpenCurrent(cfg)
	}

	// Handle sync mode
	if doSync {
		return performSyncInternal(cfg, false, forceFull)
	}

	// Open description index
	indexPath := filepath.Join(cfg.Cache.Dir, "description.bleve")

	descIndex, recreated, err := index.NewDescriptionIndexWithAutoRecreate(indexPath)
	if err != nil {
		return fmt.Errorf("failed to open index: %w", err)
	}

	// If index was recreated due to version mismatch, trigger full sync
	if recreated {
		logger.Info("Index schema updated, performing full sync to rebuild cache...")
		if err := descIndex.Close(); err != nil {
			logger.Debug("Failed to close index: %v", err)
		}
		if err := performSyncInternal(cfg, false, true); err != nil {
			return fmt.Errorf("failed to rebuild index after schema update: %w", err)
		}
		// Reopen the index after sync
		descIndex, _, err = index.NewDescriptionIndexWithAutoRecreate(indexPath)
		if err != nil {
			return fmt.Errorf("failed to reopen index after sync: %w", err)
		}
	}

	// Ensure index is closed on all return paths
	// Set this to false before interactive mode to allow explicit close there
	shouldCloseIndex := true
	defer func() {
		if shouldCloseIndex {
			if err := descIndex.Close(); err != nil {
				logger.Debug("Failed to close index: %v", err)
			}
		}
	}()

	allProjects, err := descIndex.GetAllProjects()
	if err != nil {
		return fmt.Errorf("failed to load projects: %w", err)
	}

	// Check if we have any projects - if not, run sync
	if len(allProjects) == 0 {
		logger.Debug("No projects in index, running sync...")
		shouldCloseIndex = false
		if err := descIndex.Close(); err != nil {
			logger.Debug("Failed to close index: %v", err)
		}

		fmt.Println("First run detected - synchronizing projects from GitLab...")
		fmt.Println()
		if err := performSyncInternal(cfg, false, true); err != nil {
			return fmt.Errorf("sync failed: %w\n\nYou can try running 'glf --sync' manually", err)
		}
		fmt.Println()

		// Reopen index and reload projects
		descIndex, _, err = index.NewDescriptionIndexWithAutoRecreate(indexPath)
		if err != nil {
			return fmt.Errorf("failed to reopen index after sync: %w", err)
		}
		shouldCloseIndex = true

		allProjects, err = descIndex.GetAllProjects()
		if err != nil {
			return fmt.Errorf("failed to load projects after sync: %w", err)
		}
	}

	// Decide mode: interactive or direct search
	// Join all args to support multi-word queries: "glf api ingress"
	query := strings.TrimSpace(strings.Join(args, " "))

	// JSON output mode: return results in JSON format (for integrations like Raycast)
	if jsonOutput {
		return runJSONMode(allProjects, query, cfg, descIndex)
	}

	// Auto-go mode: select first result and open in browser
	if autoGo {
		if query == "" {
			return fmt.Errorf("-g/--go requires a search query")
		}
		return runAutoGo(allProjects, query, cfg, descIndex)
	}

	// Going to interactive mode - close index explicitly before launching TUI
	// (TUI will open it again when needed)
	shouldCloseIndex = false
	if err := descIndex.Close(); err != nil {
		logger.Debug("Failed to close index: %v", err)
	}
	// Launch interactive TUI with optional initial query
	return runInteractive(allProjects, query, cfg)
}

// runJSONMode outputs search results in JSON format for API integrations
func runJSONMode(projects []model.Project, query string, cfg *config.Config, descIndex *index.DescriptionIndex) error {
	if len(projects) == 0 {
		return outputJSONError("no projects in cache")
	}

	// Load history for score boosting (used for both empty and non-empty queries)
	historyPath := filepath.Join(cfg.Cache.Dir, "history.gob")
	hist := history.New(historyPath)

	// Load history synchronously
	errCh := hist.LoadAsync()
	if err := <-errCh; err != nil {
		logger.Debug("Failed to load history: %v", err)
	}

	var matches []index.CombinedMatch
	var err error

	// If query is provided, perform search
	if query != "" {
		// Get query-specific history scores
		historyScores := hist.GetAllScoresForQuery(query)

		// Perform search
		matches, err = search.CombinedSearchWithIndex(query, projects, historyScores, cfg.Cache.Dir, descIndex)
		if err != nil {
			return outputJSONError(fmt.Sprintf("search failed: %v", err))
		}
	} else {
		// No query - rank by history scores and starred status
		// Get ALL history scores (not query-specific)
		allHistoryScores := hist.GetAllScores()

		// Create matches with history scores
		matches = make([]index.CombinedMatch, len(projects))
		for i, proj := range projects {
			histScore := float64(allHistoryScores[proj.Path])

			// Give starred projects a bonus
			starBonus := 0.0
			if proj.Starred {
				starBonus = 50.0 // Same bonus as in search
			}

			matches[i] = index.CombinedMatch{
				Project:    proj,
				TotalScore: histScore + starBonus,
			}
		}

		// Sort by score descending (highest first)
		// Projects with higher history scores + starred bonus come first
		for i := 0; i < len(matches)-1; i++ {
			for j := i + 1; j < len(matches); j++ {
				if matches[j].TotalScore > matches[i].TotalScore {
					matches[i], matches[j] = matches[j], matches[i]
				}
			}
		}
	}

	// JSON mode: Include ALL projects with status fields (excluded, archived, member)
	// API consumers (like Raycast) can implement their own filtering based on these fields
	// The --show-hidden flag is more relevant for TUI where we control display

	// Apply limit
	if limitResults > 0 && len(matches) > limitResults {
		matches = matches[:limitResults]
	}

	// Convert to JSON format
	gitlabURL := strings.TrimSuffix(cfg.GitLab.URL, "/")
	jsonProjects := make([]JSONProject, len(matches))
	for i, match := range matches {
		projectPath := strings.TrimPrefix(match.Project.Path, "/")
		projectURL := fmt.Sprintf("%s/%s", gitlabURL, projectPath)

		// Check if project is excluded via config
		isExcluded := cfg != nil && cfg.IsExcluded(match.Project.Path)

		jsonProjects[i] = JSONProject{
			Path:        match.Project.Path,
			Name:        match.Project.Name,
			Description: match.Project.Description,
			URL:         projectURL,
			Starred:     match.Project.Starred,
			Excluded:    isExcluded,
			Archived:    match.Project.Archived,
			Member:      match.Project.Member,
		}

		// Include score if --scores flag is set
		if showScores {
			jsonProjects[i].Score = match.TotalScore
		}
	}

	// Create result
	result := JSONSearchResult{
		Query:   query,
		Results: jsonProjects,
		Total:   len(matches),
		Limit:   limitResults,
	}

	return outputJSON(result)
}

// outputJSON outputs a value as JSON to stdout
func outputJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(v); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}
	return nil
}

// outputJSONError outputs an error in JSON format and returns nil
// (so the program can exit cleanly with JSON output)
func outputJSONError(message string) error {
	if err := outputJSON(JSONError{Error: message}); err != nil {
		// If JSON encoding fails, fall back to stderr
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	}
	os.Exit(1)
	return nil
}

// runAutoGo automatically selects first result and opens it in browser
func runAutoGo(projects []model.Project, query string, cfg *config.Config, descIndex *index.DescriptionIndex) error {
	// Default sync function that calls performSyncInternal
	syncFunc := func() error {
		return performSyncInternal(cfg, true, false)
	}
	return runAutoGoWithSync(projects, query, cfg, descIndex, syncFunc)
}

// runAutoGoWithSync is the testable version that accepts a sync function
func runAutoGoWithSync(projects []model.Project, query string, cfg *config.Config, descIndex *index.DescriptionIndex, syncFunc func() error) error {
	if len(projects) == 0 {
		return fmt.Errorf("no projects in cache")
	}

	// Load history for score boosting
	historyPath := filepath.Join(cfg.Cache.Dir, "history.gob")
	hist := history.New(historyPath)

	// Load history synchronously
	errCh := hist.LoadAsync()
	if err := <-errCh; err != nil {
		logger.Debug("Failed to load history: %v", err)
	}

	// Get query-specific history scores
	historyScores := hist.GetAllScoresForQuery(query)

	// Perform search
	matches, err := search.CombinedSearchWithIndex(query, projects, historyScores, cfg.Cache.Dir, descIndex)
	if err != nil {
		return fmt.Errorf("search failed: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no projects found for query: %s", query)
	}

	// Take first result
	firstProject := matches[0].Project

	// Record selection in history
	if hist != nil {
		hist.RecordSelectionWithQuery(query, firstProject.Path)
		if err := hist.Save(); err != nil {
			logger.Debug("Failed to save history: %v", err)
		}
	}

	// Construct URL
	gitlabURL := strings.TrimSuffix(cfg.GitLab.URL, "/")
	projectPath := strings.TrimPrefix(firstProject.Path, "/")
	projectURL := fmt.Sprintf("%s/%s", gitlabURL, projectPath)

	// Always open in browser (that's the point of -g/--go)
	// IMMEDIATE USER FEEDBACK - open browser first
	logger.Debug("Opening browser with URL: %s", projectURL)
	if err := openBrowser(projectURL); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open browser: %v\n", err)
		logger.Debug("Browser open error: %v", err)
	} else {
		logger.Debug("Browser command executed successfully")
	}

	// Output URL immediately (don't wait for sync)
	fmt.Println(projectURL)

	// Start background sync to update cache for next time
	// User already has browser open, so sync happens completely in background
	// No waiting - auto-go mode prioritizes speed over cache freshness
	logger.Debug("Starting background sync...")
	go func() {
		if err := syncFunc(); err != nil {
			logger.Debug("Background sync failed: %v", err)
		} else {
			logger.Debug("Background sync completed successfully")
		}
	}()

	return nil
}

// openBrowser opens the given URL in the default browser (cross-platform)
func openBrowser(url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case platformDarwin: // macOS
		cmd = exec.CommandContext(ctx, "open", url)
	case platformLinux:
		cmd = exec.CommandContext(ctx, "xdg-open", url)
	case platformWindows:
		// Empty string before URL is important: start interprets first quoted arg as window title
		cmd = exec.CommandContext(ctx, "cmd", "/c", "start", "", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	// Use Run() instead of Start() to wait for the command to complete
	// This ensures the browser actually opens before we return
	return cmd.Run()
}

// getGitRemoteURL gets the Git remote origin URL for the given directory
func getGitRemoteURL(dir string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", dir, "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("not a git repository or no remote origin configured: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("failed to get git remote URL: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// extractProjectPath extracts the project path from a Git remote URL
// Returns: projectPath, baseURL, error
// baseURL is either the configured GitLab URL or the public repository host URL
func extractProjectPath(remoteURL, gitlabURL string) (string, string, error) {
	// Known public Git hosting services
	publicHosts := map[string]string{
		"github.com":    "https://github.com",
		"gitlab.com":    "https://gitlab.com",
		"bitbucket.org": "https://bitbucket.org",
	}

	gitlabURL = strings.TrimSuffix(gitlabURL, "/")
	var gitlabHost string

	// Parse GitLab URL to extract host (including port)
	if strings.HasPrefix(gitlabURL, "https://") || strings.HasPrefix(gitlabURL, "http://") {
		parsed, err := url.Parse(gitlabURL)
		if err != nil {
			return "", "", fmt.Errorf("invalid GitLab URL format: %s", gitlabURL)
		}
		gitlabHost = parsed.Host // Host includes port if present (e.g., "gitlab.example.com:8443")
	} else {
		return "", "", fmt.Errorf("invalid GitLab URL format: %s", gitlabURL)
	}

	var projectPath string
	var remoteHost string

	// Handle SSH with ssh:// prefix and port: ssh://git@gitlab.com:port/namespace/project.git
	if strings.HasPrefix(remoteURL, "ssh://") {
		rest := strings.TrimPrefix(remoteURL, "ssh://")
		rest = strings.TrimPrefix(rest, "git@") // Remove git@ if present

		// Split by first slash to separate host:port from path
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH remote URL format: %s", remoteURL)
		}

		remoteHost = parts[0] // Includes port if present
		projectPath = strings.TrimSuffix(parts[1], ".git")
	} else if strings.HasPrefix(remoteURL, "git@") {
		// Handle SSH format: git@gitlab.com:namespace/project.git (no port in this format)
		parts := strings.SplitN(remoteURL, ":", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid SSH remote URL format: %s", remoteURL)
		}

		remoteHost = strings.TrimPrefix(parts[0], "git@")
		projectPath = strings.TrimSuffix(parts[1], ".git")
	} else if strings.HasPrefix(remoteURL, "https://") || strings.HasPrefix(remoteURL, "http://") {
		// Handle HTTPS/HTTP format: https://gitlab.com:8443/namespace/project.git
		parsed, err := url.Parse(remoteURL)
		if err != nil {
			return "", "", fmt.Errorf("invalid remote URL format: %s", remoteURL)
		}

		remoteHost = parsed.Host // Host includes port if present
		pathPart := strings.TrimPrefix(parsed.Path, "/")

		if pathPart == "" {
			return "", "", fmt.Errorf("invalid remote URL format: no path found in %s", remoteURL)
		}

		projectPath = strings.TrimSuffix(pathPart, ".git")
	} else {
		return "", "", fmt.Errorf("unsupported git remote URL format: %s (expected SSH or HTTPS)", remoteURL)
	}

	// Ensure project path doesn't start with /
	projectPath = strings.TrimPrefix(projectPath, "/")

	if projectPath == "" {
		return "", "", fmt.Errorf("could not extract project path from remote URL: %s", remoteURL)
	}

	// Extract hostname without port for comparison
	// remoteHost might be "gitlab.com" or "gitlab.com:8443"
	// gitlabHost might be "gitlab.com" or "gitlab.com:8443"
	remoteHostname := remoteHost
	gitlabHostname := gitlabHost

	// Strip port from remote host if present
	if idx := strings.Index(remoteHost, ":"); idx != -1 {
		remoteHostname = remoteHost[:idx]
	}

	// Strip port from gitlab host if present
	if idx := strings.Index(gitlabHost, ":"); idx != -1 {
		gitlabHostname = gitlabHost[:idx]
	}

	// Check if remote matches configured GitLab (compare both full host and hostname)
	if remoteHost == gitlabHost || remoteHostname == gitlabHostname {
		return projectPath, gitlabURL, nil
	}

	// Check if it's a known public repository host
	if publicBaseURL, isPublic := publicHosts[remoteHostname]; isPublic {
		return projectPath, publicBaseURL, nil
	}

	// Not a match - return error
	return "", "", fmt.Errorf("git remote '%s' does not match configured GitLab '%s' and is not a known public repository (github.com, gitlab.com, bitbucket.org)", remoteHost, gitlabHost)
}

// runOpenCurrent opens the current directory's Git repository in the browser
func runOpenCurrent(cfg *config.Config) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Get Git remote URL
	remoteURL, err := getGitRemoteURL(cwd)
	if err != nil {
		return fmt.Errorf("failed to get git remote URL: %w", err)
	}

	logger.Debug("Git remote URL: %s", remoteURL)

	// Extract project path and base URL (either configured GitLab or public host)
	projectPath, baseURL, err := extractProjectPath(remoteURL, cfg.GitLab.URL)
	if err != nil {
		return fmt.Errorf("failed to extract project path: %w", err)
	}

	logger.Debug("Extracted project path: %s", projectPath)
	logger.Debug("Base URL: %s", baseURL)

	// Construct project URL using the base URL from extraction
	projectURL := fmt.Sprintf("%s/%s", baseURL, projectPath)

	// Open in browser
	logger.Debug("Opening browser with URL: %s", projectURL)
	if err := openBrowser(projectURL); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open browser: %v\n", err)
		logger.Debug("Browser open error: %v", err)
	} else {
		logger.Debug("Browser command executed successfully")
	}

	// Output URL to stdout
	fmt.Println(projectURL)

	return nil
}

// runShowHistory displays search history with scores
func runShowHistory(cfg *config.Config) error {
	historyPath := filepath.Join(cfg.Cache.Dir, "history.gob")
	hist := history.New(historyPath)

	// Load history synchronously
	errCh := hist.LoadAsync()
	if err := <-errCh; err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	// Get all history entries sorted by score
	entries := hist.GetAllEntries()

	if len(entries) == 0 {
		fmt.Println("No history yet. Use glf to search and select projects.")
		return nil
	}

	// Display history
	fmt.Printf("Search History (%d projects)\n\n", len(entries))
	fmt.Println("Project Path                                              Count  Last Used         Score")
	fmt.Println("─────────────────────────────────────────────────────── ────── ───────────────── ─────")

	for _, entry := range entries {
		// Format last used time
		lastUsed := entry.LastUsed.Format("2006-01-02 15:04")

		// Truncate long paths
		path := entry.ProjectPath
		if len(path) > 55 {
			path = path[:52] + "..."
		}

		fmt.Printf("%-55s %6d %17s %5d\n", path, entry.Count, lastUsed, entry.Score)
	}

	// Show stats
	totalSelections, uniqueProjects := hist.Stats()
	fmt.Printf("\nTotal selections: %d | Unique projects: %d\n", totalSelections, uniqueProjects)

	return nil
}

// runClearHistory clears the search history
func runClearHistory(cfg *config.Config) error {
	historyPath := filepath.Join(cfg.Cache.Dir, "history.gob")
	hist := history.New(historyPath)

	// Load history synchronously
	errCh := hist.LoadAsync()
	if err := <-errCh; err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	// Get stats before clearing
	totalSelections, uniqueProjects := hist.Stats()

	if totalSelections == 0 {
		fmt.Println("History is already empty.")
		return nil
	}

	// Clear history
	hist.Clear()

	// Save empty history
	if err := hist.Save(); err != nil {
		return fmt.Errorf("failed to save cleared history: %w", err)
	}

	fmt.Printf("✓ History cleared: %d selections from %d projects removed\n", totalSelections, uniqueProjects)

	return nil
}

// runRecordSelection records a project selection in the history (for JSON integrations)
func runRecordSelection(cfg *config.Config, projectPath, query string) error {
	historyPath := filepath.Join(cfg.Cache.Dir, "history.gob")
	hist := history.New(historyPath)

	// Load history synchronously
	errCh := hist.LoadAsync()
	if err := <-errCh; err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	// Record selection with or without query context
	if query != "" {
		hist.RecordSelectionWithQuery(query, projectPath)
		logger.Debug("Recorded selection: %s (query: %s)", projectPath, query)
	} else {
		hist.RecordSelection(projectPath)
		logger.Debug("Recorded selection: %s (no query)", projectPath)
	}

	// Save history
	if err := hist.Save(); err != nil {
		return fmt.Errorf("failed to save history: %w", err)
	}

	logger.Debug("History saved successfully")
	return nil
}

// runInteractive launches the interactive TUI with optional initial query
func runInteractive(projects []model.Project, initialQuery string, cfg *config.Config) error {
	if len(projects) == 0 {
		fmt.Println("No projects in cache. Run 'glf --sync' to fetch projects.")
		return nil
	}

	// Fetch current username for display in header
	// Try to load from cache first
	cacheManager := cache.New(cfg.Cache.Dir)
	username, err := cacheManager.LoadUsername()
	if err != nil {
		logger.Debug("Failed to load cached username: %v", err)
		username = ""
	}

	// If no cached username, try to fetch from API with reduced timeout
	if username == "" {
		// Use 10-second timeout for username fetch (faster fail on network issues)
		shortTimeout := 10 * time.Second
		client, err := gitlab.New(cfg.GitLab.URL, cfg.GitLab.Token, shortTimeout)
		if err != nil {
			logger.Debug("Failed to create GitLab client for username fetch: %v", err)
		} else {
			fetchedUsername, err := client.GetCurrentUsername()
			if err != nil {
				// Don't fail on username fetch error, just use empty string
				logger.Debug("Failed to fetch username: %v", err)
			} else {
				username = fetchedUsername
				// Save to cache for next time
				if err := cacheManager.SaveUsername(username); err != nil {
					logger.Debug("Failed to save username to cache: %v", err)
				} else {
					logger.Debug("Username cached: @%s", username)
				}
			}
		}
	} else {
		logger.Debug("Using cached username: @%s", username)
	}

	// Create sync callback
	syncCallback := func() tea.Cmd {
		return func() tea.Msg {
			// Perform sync in background
			indexPath := filepath.Join(cfg.Cache.Dir, "description.bleve")

			// Create GitLab client
			client, err := gitlab.New(cfg.GitLab.URL, cfg.GitLab.Token, cfg.GitLab.GetTimeout())
			if err != nil {
				return tui.SyncCompleteMsg{Err: err}
			}

			// Check for incremental sync
			cacheManager := cache.New(cfg.Cache.Dir)
			lastSyncTime, err := cacheManager.LoadLastSyncTime()
			lastFullSyncTime, fullSyncErr := cacheManager.LoadLastFullSyncTime()
			if fullSyncErr != nil {
				logger.Debug("Failed to load last full sync time: %v", fullSyncErr)
			}

			var sincePtr *time.Time
			var syncMode string
			const fullSyncInterval = 7 * 24 * time.Hour

			// Decide sync mode (same logic as sync command)
			if err != nil {
				// Error loading timestamp - fall back to full sync
				logger.Debug("TUI sync: could not load last sync time: %v, performing full sync", err)
				syncMode = syncModeFull
			} else if lastSyncTime.IsZero() {
				// First sync ever
				logger.Debug("TUI sync: first sync detected, performing full sync")
				syncMode = syncModeFull
			} else if !lastFullSyncTime.IsZero() && time.Since(lastFullSyncTime) > fullSyncInterval {
				// Last full sync was >7 days ago - auto full sync to remove deleted projects
				daysSinceFullSync := int(time.Since(lastFullSyncTime).Hours() / 24)
				logger.Debug("TUI sync: auto full sync (last full sync was %d days ago, removes deleted projects)", daysSinceFullSync)
				syncMode = syncModeFull
			} else {
				// Incremental sync possible
				sincePtr = &lastSyncTime
				logger.Debug("TUI sync: incremental (since %v ago)", time.Since(lastSyncTime).Round(time.Second))
				syncMode = syncModeIncremental
			}

			// Fetch projects (incremental or full)
			// Always fetch ALL projects (membership=false) - filtering happens at display time
			newProjects, err := client.FetchAllProjects(sincePtr, false)
			if err != nil {
				return tui.SyncCompleteMsg{Err: err}
			}

			// Open or create description index
			descIndex, recreated, err := index.NewDescriptionIndexWithAutoRecreate(indexPath)
			if err != nil {
				return tui.SyncCompleteMsg{Err: err}
			}

			// If index was recreated due to version mismatch, trigger full sync in TUI context
			if recreated {
				logger.Debug("TUI sync: index schema updated, switching to full sync mode")
				syncMode = syncModeFull

				// Re-fetch all projects for full sync
				// Always fetch ALL projects (membership=false) - filtering happens at display time
				newProjects, err = client.FetchAllProjects(nil, false)
				if err != nil {
					return tui.SyncCompleteMsg{Err: err}
				}
				logger.Debug("TUI sync: re-fetched %d projects for full sync after index recreation", len(newProjects))
			}
			defer func() {
				if err := descIndex.Close(); err != nil {
					logger.Debug("Failed to close index: %v", err)
				}
			}()

			// Prepare documents for batch indexing
			batchDocs := make([]index.DescriptionDocument, 0, len(newProjects))
			for _, proj := range newProjects {
				// Index all projects, even those without descriptions
				batchDocs = append(batchDocs, index.DescriptionDocument{
					ProjectPath: proj.Path,
					ProjectName: proj.Name,
					Description: proj.Description,
					Starred:     proj.Starred,
					Archived:    proj.Archived,
					Member:      proj.Member,
				})
			}

			// Index all projects in batches
			if len(batchDocs) > 0 {
				// Index in batches of 100
				for i := 0; i < len(batchDocs); i += 100 {
					end := i + 100
					if end > len(batchDocs) {
						end = len(batchDocs)
					}
					if err := descIndex.AddBatch(batchDocs[i:end]); err != nil {
						return tui.SyncCompleteMsg{Err: err}
					}
				}
			}

			// Save timestamp for successful sync
			syncCompletedAt := time.Now()
			if err := cacheManager.SaveLastSyncTime(syncCompletedAt); err != nil {
				logger.Debug("Failed to save TUI sync timestamp: %v", err)
			}

			// Save last full sync time only if this was a full sync
			if syncMode == syncModeFull {
				if err := cacheManager.SaveLastFullSyncTime(syncCompletedAt); err != nil {
					logger.Debug("Failed to save TUI full sync timestamp: %v", err)
				} else {
					logger.Debug("TUI full sync timestamp saved: %s", syncCompletedAt.Format(time.RFC3339))
				}
			}

			// CRITICAL: For incremental sync, we fetched only CHANGED projects
			// But TUI needs ALL projects, so load complete list from index
			allProjects, err := descIndex.GetAllProjects()
			if err != nil {
				return tui.SyncCompleteMsg{Err: fmt.Errorf("failed to load all projects after sync: %w", err)}
			}

			return tui.SyncCompleteMsg{Projects: allProjects, Err: nil}
		}
	}

	// Create and run the TUI with initial query, sync callback, cache dir for history, config, showScores flag, showHidden flag, username, and version
	m := tui.New(projects, initialQuery, syncCallback, cfg.Cache.Dir, cfg, showScores, showHidden, username, version)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	// Check if user selected a project
	if model, ok := finalModel.(tui.Model); ok {
		selected := model.Selected()
		if selected != "" {
			// Construct GitLab project URL
			gitlabURL := strings.TrimSuffix(cfg.GitLab.URL, "/")
			projectPath := strings.TrimPrefix(selected, "/")
			projectURL := fmt.Sprintf("%s/%s", gitlabURL, projectPath)

			// Open in browser
			logger.Debug("Opening browser with URL: %s", projectURL)
			if err := openBrowser(projectURL); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to open browser: %v\n", err)
				logger.Debug("Browser open error: %v", err)
			} else {
				logger.Debug("Browser command executed successfully")
			}

			// Output URL to stdout (for copying or script usage)
			fmt.Println(projectURL)
		}
	}

	return nil
}

// performSyncInternal performs the actual sync logic
// silent=true suppresses Info/Success messages (for background sync)
// forceFullSync=true forces full sync regardless of timestamps
func performSyncInternal(cfg *config.Config, silent bool, forceFullSync bool) error {
	logInfo := logger.Info
	if silent {
		logInfo = logger.Debug
	}

	// Create GitLab client with timeout
	logInfo("Connecting to GitLab at %s (timeout: %ds)...", cfg.GitLab.URL, cfg.GitLab.Timeout)
	client, err := gitlab.New(cfg.GitLab.URL, cfg.GitLab.Token, cfg.GitLab.GetTimeout())
	if err != nil {
		logger.Error("Failed to create GitLab client")
		return fmt.Errorf("GitLab client error: %w", err)
	}

	return performSyncInternalWithClient(cfg, client, silent, forceFullSync)
}

// performSyncInternalWithClient performs sync with an injected GitLab client (testable version)
func performSyncInternalWithClient(cfg *config.Config, client gitlab.GitLabClient, silent bool, forceFullSync bool) error {
	logInfo := logger.Info
	logSuccess := logger.Success
	if silent {
		logInfo = logger.Debug
		logSuccess = logger.Debug
	}

	// Test connection
	logger.Debug("Testing GitLab connection...")
	if err := client.TestConnection(); err != nil {
		logger.Error("Connection test failed")
		logInfo("Please check:")
		logInfo("  - GitLab URL is correct: %s", cfg.GitLab.URL)
		logInfo("  - Personal Access Token is valid")
		logInfo("  - Network connection is available")
		logInfo("  - GitLab server is accessible")
		return fmt.Errorf("connection test failed: %w", err)
	}
	logSuccess("Connected successfully")

	// Check for incremental sync capability
	cacheManager := cache.New(cfg.Cache.Dir)
	lastSyncTime, err := cacheManager.LoadLastSyncTime()
	lastFullSyncTime, fullSyncErr := cacheManager.LoadLastFullSyncTime()
	if fullSyncErr != nil {
		logger.Debug("Failed to load last full sync time: %v", fullSyncErr)
	}

	var projects []model.Project
	var syncMode string
	const fullSyncInterval = 7 * 24 * time.Hour // 7 days

	// Decide sync mode: full vs incremental
	if forceFullSync {
		// User explicitly requested full sync
		logInfo("Full sync requested (--full flag)")
		syncMode = syncModeFull
	} else if err != nil {
		// Error loading timestamp - fall back to full sync
		logger.Debug("Could not load last sync time: %v, performing full sync", err)
		syncMode = syncModeFull
	} else if lastSyncTime.IsZero() {
		// First sync ever
		logInfo("First sync detected")
		syncMode = syncModeFull
	} else if !lastFullSyncTime.IsZero() && time.Since(lastFullSyncTime) > fullSyncInterval {
		// Last full sync was >7 days ago - auto full sync to remove deleted projects
		daysSinceFullSync := int(time.Since(lastFullSyncTime).Hours() / 24)
		logInfo("Auto full sync: last full sync was %d days ago (removes deleted projects)", daysSinceFullSync)
		syncMode = syncModeFull
	} else {
		// Incremental sync possible
		timeSinceLastSync := time.Since(lastSyncTime)
		logInfo("Incremental sync: fetching projects changed since %v ago", timeSinceLastSync.Round(time.Second))
		syncMode = syncModeIncremental
	}

	// Fetch projects (full or incremental)
	logInfo("Fetching projects...")
	start := time.Now()

	var sincePtr *time.Time
	if syncMode == syncModeIncremental {
		sincePtr = &lastSyncTime
	}

	// Always fetch ALL projects (membership=false) - filtering happens at display time
	projects, err = client.FetchAllProjects(sincePtr, false)
	if err != nil {
		logger.Error("Failed to fetch projects")
		return fmt.Errorf("fetch error: %w", err)
	}
	elapsed := time.Since(start)

	if syncMode == syncModeIncremental {
		logSuccess("Fetched %d changed projects in %v", len(projects), elapsed)
		if len(projects) == 0 {
			logInfo("No projects changed since last sync")
			return nil // Early return - nothing to index
		}
	} else {
		logSuccess("Fetched %d projects in %v", len(projects), elapsed)
		if len(projects) == 0 {
			logger.Warn("No projects found. Check if your token has sufficient permissions.")
			return nil
		}
	}

	// Index project descriptions
	if err := indexDescriptions(projects, cfg.Cache.Dir, silent); err != nil {
		logger.Warn("Description indexing failed: %v", err)
		logInfo("Search will work without description content. Run 'glf --sync' again to retry.")
		// Don't fail the entire sync if indexing fails
	}

	// Save timestamps for successful sync
	syncCompletedAt := time.Now()

	// Always save last sync time (for incremental)
	if err := cacheManager.SaveLastSyncTime(syncCompletedAt); err != nil {
		logger.Warn("Failed to save sync timestamp: %v (incremental sync won't work next time)", err)
	} else {
		logger.Debug("Sync timestamp saved: %s", syncCompletedAt.Format(time.RFC3339))
	}

	// Save last full sync time only if this was a full sync
	if syncMode == syncModeFull {
		if err := cacheManager.SaveLastFullSyncTime(syncCompletedAt); err != nil {
			logger.Warn("Failed to save full sync timestamp: %v", err)
		} else {
			logger.Debug("Full sync timestamp saved: %s", syncCompletedAt.Format(time.RFC3339))
		}
	}

	if !silent {
		logInfo("\nRun 'glf' to search projects interactively")
	}

	return nil
}

// indexDescriptions indexes project descriptions for full-text search
func indexDescriptions(projects []model.Project, cacheDir string, silent bool) error {
	logInfo := logger.Info
	logSuccess := logger.Success
	if silent {
		logInfo = logger.Debug
		logSuccess = logger.Debug
	}

	logInfo("Indexing project descriptions...")
	start := time.Now()

	// Create or open index
	indexPath := filepath.Join(cacheDir, "description.bleve")
	descriptionIndex, recreated, err := index.NewDescriptionIndexWithAutoRecreate(indexPath)
	if err != nil {
		return fmt.Errorf("failed to create description index: %w", err)
	}

	// If index was recreated, we're already in a full sync context, so just log it
	if recreated {
		logger.Debug("Index schema updated during indexing, new index created with current version")
	}
	defer func() {
		if err := descriptionIndex.Close(); err != nil {
			logger.Debug("Failed to close index: %v", err)
		}
	}()

	// Get current document count
	docCount, countErr := descriptionIndex.Count()
	if countErr != nil {
		logger.Debug("Failed to get document count: %v", countErr)
	} else if docCount > 0 {
		logger.Debug("Existing index has %d documents", docCount)
	}

	// Prepare documents for batch indexing
	var indexed int
	batchDocs := make([]index.DescriptionDocument, 0, 100)

	for _, proj := range projects {
		// Index all projects, even those without descriptions
		batchDocs = append(batchDocs, index.DescriptionDocument{
			ProjectPath: proj.Path,
			ProjectName: proj.Name,
			Description: proj.Description,
			Starred:     proj.Starred,
			Archived:    proj.Archived,
			Member:      proj.Member,
		})

		// Index batch when it reaches 100 docs
		if len(batchDocs) >= 100 {
			if err := descriptionIndex.AddBatch(batchDocs); err != nil {
				logger.Debug("Failed to index batch: %v", err)
				return fmt.Errorf("failed to index batch: %w", err)
			}
			indexed += len(batchDocs)
			batchDocs = batchDocs[:0] // Clear batch

			// Show progress
			if indexed%50 == 0 {
				logger.Debug("Progress: %d/%d (%d%%)", indexed, len(projects), (indexed*100)/len(projects))
			}
		}
	}

	// Index remaining documents
	if len(batchDocs) > 0 {
		if err := descriptionIndex.AddBatch(batchDocs); err != nil {
			logger.Debug("Failed to index final batch: %v", err)
			return fmt.Errorf("failed to index final batch: %w", err)
		}
		indexed += len(batchDocs)
	}

	elapsed := time.Since(start)
	logSuccess("Description indexing complete in %v", elapsed)
	logInfo("  Indexed: %d projects", indexed)

	return nil
}

// runConfigWizard runs the interactive configuration wizard
func runConfigWizard() error {
	reader := bufio.NewReader(os.Stdin)

	// Check if config exists
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "glf", "config.yaml")
	configExists := false
	if _, err := os.Stat(configPath); err == nil {
		configExists = true
	}

	// Handle reset flag
	if resetFlag {
		if !configExists {
			fmt.Println("No existing configuration found.")
			fmt.Println()
			// Continue to wizard
		} else {
			confirmed, err := confirmReset(reader)
			if err != nil {
				return err
			}
			if !confirmed {
				fmt.Println("Configuration reset cancelled.")
				return nil
			}
			// Delete config file
			if err := os.Remove(configPath); err != nil {
				return fmt.Errorf("failed to remove config: %w", err)
			}
			fmt.Println("✓ Configuration deleted")
			fmt.Println()
			configExists = false // Treat as first-time setup
		}
	}

	// Show welcome message
	showWelcomeMessage(configExists && !resetFlag)

	// Load existing config if available (for defaults)
	existingCfg, err := config.Load()
	if err != nil {
		// If no config exists, create empty config for defaults
		existingCfg = &config.Config{}
	}

	// Step 1: Get and validate GitLab URL
	var gitlabURL string
	for {
		urlInput, err := promptForURL(reader, existingCfg.GitLab.URL)
		if err != nil {
			return err
		}

		normalizedURL, err := parseGitLabURL(urlInput)
		if err != nil {
			fmt.Printf("   ❌ Invalid URL: %v\n", err)
			fmt.Println("   Please try again.")
			fmt.Println()
			continue
		}

		gitlabURL = normalizedURL
		break
	}

	// Step 2: Show smart token helper
	showTokenHelper(gitlabURL)

	// Step 3: Get and validate token
	var token string
	for {
		tokenInput, err := promptForToken(reader, existingCfg.GitLab.Token)
		if err != nil {
			return err
		}

		if err := validateToken(tokenInput); err != nil {
			fmt.Printf("   ⚠️  %v\n", err)
			fmt.Print("   Use this token anyway? [y/N]: ")
			response, readErr := reader.ReadString('\n')
			if readErr != nil {
				return err
			}
			response = strings.ToLower(strings.TrimSpace(response))
			if response == "y" || response == responseYes {
				token = tokenInput
				break
			}
			fmt.Println()
			continue
		}

		token = tokenInput
		break
	}

	// Step 4: Create config and test connection (use default timeout)
	cfg := &config.Config{
		GitLab: config.GitLabConfig{
			URL:     gitlabURL,
			Token:   token,
			Timeout: 30, // Default timeout
		},
		Cache:         existingCfg.Cache,
		ExcludedPaths: existingCfg.ExcludedPaths,
	}

	if err := testConnectionWithRetry(cfg, reader); err != nil {
		return err
	}

	// Step 6: Save configuration
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "glf")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("✓ Configuration saved to %s\n", configPath)

	// Step 7: Automatically perform full sync and launch
	fmt.Println()
	fmt.Println("🎉 Configuration Complete!")
	fmt.Println()
	fmt.Println("🔄 Syncing all projects from GitLab...")
	fmt.Println()

	// Perform full sync (force=true to get all projects)
	if err := performSyncInternal(cfg, false, true); err != nil {
		fmt.Printf("\n⚠️  Sync failed: %v\n", err)
		fmt.Println("You can run 'glf --sync' manually later.")
		fmt.Println()
		return nil
	}

	// Launch interactive TUI
	fmt.Println()
	fmt.Println("🚀 Launching GLF...")
	fmt.Println()

	// Load projects from index
	indexPath := filepath.Join(cfg.Cache.Dir, "description.bleve")
	descIndex, _, err := index.NewDescriptionIndexWithAutoRecreate(indexPath)
	if err != nil {
		fmt.Printf("⚠️  Failed to open index: %v\n", err)
		fmt.Println("Run 'glf' to start searching.")
		return nil
	}
	// Note: recreated flag ignored here - wizard already ran full sync above

	// Use flag to control defer close behavior
	shouldCloseIndex := true
	defer func() {
		if shouldCloseIndex {
			_ = descIndex.Close() // Silent close - we're in interactive mode
		}
	}()

	allProjects, err := descIndex.GetAllProjects()
	if err != nil {
		fmt.Printf("⚠️  Failed to load projects: %v\n", err)
		fmt.Println("Run 'glf' to start searching.")
		return nil
	}

	if len(allProjects) == 0 {
		fmt.Println("No projects found. Check your GitLab permissions.")
		return nil
	}

	// Close index before launching TUI (TUI will reopen it)
	shouldCloseIndex = false
	_ = descIndex.Close() // Silent close - we're in interactive mode

	// Launch interactive TUI
	return runInteractive(allProjects, "", cfg)
}

// maskToken masks a token for display, showing only first and last 4 characters
func maskToken(token string) string {
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "****" + token[len(token)-4:]
}

// parseGitLabURL normalizes and validates a GitLab URL
func parseGitLabURL(rawURL string) (string, error) {
	// Trim spaces
	rawURL = strings.TrimSpace(rawURL)

	// Check if URL starts with http:// or https://
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return "", fmt.Errorf("URL must start with http:// or https://")
	}

	// Parse URL to validate format
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	// Validate host is not empty
	if parsed.Host == "" {
		return "", fmt.Errorf("URL must include a host")
	}

	// Reconstruct normalized URL (removes trailing slashes, normalizes path)
	normalized := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	if parsed.Path != "" && parsed.Path != "/" {
		normalized += strings.TrimSuffix(parsed.Path, "/")
	}

	return normalized, nil
}

// generateTokenURL creates the token creation URL for a GitLab instance with pre-filled parameters
func generateTokenURL(gitlabURL string) string {
	baseURL := strings.TrimSuffix(gitlabURL, "/")

	// Build URL with query parameters to pre-fill token name and scopes
	// GitLab expects: ?name=...&scopes=scope1,scope2,scope3
	params := url.Values{}
	params.Set("name", "glf-cli-token")
	params.Set("scopes", "read_api,read_repository")

	tokenURL := baseURL + "/-/profile/personal_access_tokens?" + params.Encode()
	return tokenURL
}

// validateToken performs basic token format validation
func validateToken(token string) error {
	token = strings.TrimSpace(token)

	if len(token) == 0 {
		return fmt.Errorf("token cannot be empty")
	}

	// GitLab tokens are typically 20+ characters
	if len(token) < 20 {
		return fmt.Errorf("token seems too short (expected at least 20 characters)")
	}

	// Basic format check - no spaces allowed
	if strings.Contains(token, " ") {
		return fmt.Errorf("token should not contain spaces")
	}

	return nil
}

// confirmReset prompts user to confirm configuration reset
func confirmReset(reader *bufio.Reader) (bool, error) {
	fmt.Println()
	fmt.Println("⚠️  WARNING: This will delete your existing configuration.")
	fmt.Println("   Your project cache and history will be preserved.")
	fmt.Print("   Continue? [y/N]: ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == responseYes, nil
}

// showWelcomeMessage displays contextual welcome message with styled logo
func showWelcomeMessage(isReconfiguration bool) {
	if isReconfiguration {
		printLogo(version)
		printTitle("Configuration Update")
		printMuted("Update your GitLab connection settings.")
		printMuted("Press Enter to keep existing values.")
	} else {
		printLogo(version)
		printTitle("Initial Setup")
		printMuted("Let's connect to your GitLab instance.")
		printMuted("This will only take a minute!")
	}
	fmt.Println()
}

// showTokenHelper displays smart token creation guidance with better formatting
func showTokenHelper(gitlabURL string) {
	tokenURL := generateTokenURL(gitlabURL)

	fmt.Println()
	printSection("📋", "Personal Access Token Setup")
	fmt.Println()
	printMuted("To create a token, open this URL in your browser:")
	fmt.Println()
	printURL(tokenURL)
	fmt.Println()
	printMuted("The form will be pre-filled with:")
	printBullet("Token name: glf-cli-token")
	printBullet("Scopes: read_api, read_repository")
	fmt.Println()
}

// promptForURL prompts for GitLab URL with better formatting
func promptForURL(reader *bufio.Reader, existingURL string) (string, error) {
	printSection("🌐", "GitLab Instance URL")

	if existingURL != "" {
		fmt.Println()
		printMuted(fmt.Sprintf("Current: %s", existingURL))
		fmt.Println()
		printPrompt("New URL [Enter to keep]: ")
	} else {
		fmt.Println()
		printExample("Example: https://gitlab.company.com")
		fmt.Println()
		printPrompt("URL: ")
	}

	urlInput, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	urlInput = strings.TrimSpace(urlInput)

	if urlInput == "" && existingURL != "" {
		return existingURL, nil
	}

	return urlInput, nil
}

// promptForToken prompts for GitLab Personal Access Token with better formatting
func promptForToken(reader *bufio.Reader, existingToken string) (string, error) {
	fmt.Println()
	printSection("🔑", "Personal Access Token")

	if existingToken != "" {
		fmt.Println()
		printMuted(fmt.Sprintf("Current: %s", maskToken(existingToken)))
		fmt.Println()
		printPrompt("New token [Enter to keep]: ")
	} else {
		fmt.Println()
		printPrompt("Token: ")
	}

	token, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	token = strings.TrimSpace(token)

	if token == "" && existingToken != "" {
		return existingToken, nil
	}

	return token, nil
}

// testConnectionWithRetry tests GitLab connection with better formatting
func testConnectionWithRetry(cfg *config.Config, reader *bufio.Reader) error {
	fmt.Println()
	printSection("🔄", "Testing Connection")
	fmt.Println()
	printMuted(fmt.Sprintf("Connecting to %s...", cfg.GitLab.URL))

	client, err := gitlab.New(cfg.GitLab.URL, cfg.GitLab.Token, cfg.GitLab.GetTimeout())
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	for {
		err := client.TestConnection()
		if err == nil {
			fmt.Println()
			printSuccess("Connection successful!")
			return nil
		}

		// Connection failed - show error and options
		fmt.Println()
		printError(fmt.Sprintf("Connection failed: %v", err))
		fmt.Println()
		printMuted("Possible issues:")
		printBullet("Check if GitLab URL is correct")
		printBullet("Verify token has 'read_api' and 'read_repository' scopes")
		printBullet("Ensure GitLab server is accessible")
		printBullet("Check network connection")
		fmt.Println()
		printPrompt("What would you like to do? (R)etry / (E)dit / (C)ancel: ")

		response, err := reader.ReadString('\n')
		if err != nil {
			return err
		}

		response = strings.ToLower(strings.TrimSpace(response))
		switch response {
		case "r", "retry":
			fmt.Println()
			printMuted("Retrying...")
			continue
		case "e", "edit":
			return fmt.Errorf("connection test failed, please run 'glf --init' again to edit settings")
		case "c", "cancel":
			return fmt.Errorf("configuration cancelled")
		default:
			fmt.Println()
			printWarning("Invalid option, please choose R, E, or C")
			continue
		}
	}
}

func init() {
	// Set version info
	rootCmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, buildTime)

	// Disable auto-generated commands
	rootCmd.CompletionOptions.DisableDefaultCmd = true   // Disable 'completion' command
	rootCmd.SetHelpCommand(&cobra.Command{Hidden: true}) // Disable 'help' command (help flag still works)

	// Add flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")
	rootCmd.PersistentFlags().BoolVar(&showScores, "scores", false, "show score breakdown (search + history)")
	rootCmd.PersistentFlags().BoolVar(&autoGo, "go", false, "auto-select first result and open in browser")
	rootCmd.PersistentFlags().BoolVarP(&autoGo, "open", "g", false, "alias for --go (for compatibility)")
	rootCmd.PersistentFlags().BoolVarP(&doSync, "sync", "s", false, "synchronize projects cache")
	rootCmd.PersistentFlags().BoolVar(&forceFull, "full", false, "force full sync (use with --sync)")
	rootCmd.PersistentFlags().BoolVar(&doInit, "init", false, "run interactive configuration wizard")
	rootCmd.PersistentFlags().BoolVar(&resetFlag, "reset", false, "reset configuration and start from scratch (use with --init)")
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "output results in JSON format (for integrations)")
	rootCmd.PersistentFlags().IntVar(&limitResults, "limit", 20, "limit number of results (for JSON mode)")
	rootCmd.PersistentFlags().BoolVar(&showHistory, "history", false, "show search history with scores")
	rootCmd.PersistentFlags().BoolVar(&clearHistory, "clear-history", false, "clear search history")
	rootCmd.PersistentFlags().BoolVar(&showHidden, "show-hidden", false, "show hidden projects (excluded, archived, non-member) - toggle with Ctrl+H in TUI")
	rootCmd.PersistentFlags().StringVar(&jsonRecord, "json-record", "", "record project selection in history (project path, for JSON integrations)")
	rootCmd.PersistentFlags().StringVar(&queryContext, "query", "", "query context for recording selection (optional, used with --json-record)")

	// Set up verbose mode before command execution
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		logger.SetVerbose(verbose)
		logger.Debug("Verbose mode enabled")
	}
}

func main() {
	// Enable interspersed flags (flags can appear anywhere in the command line)
	rootCmd.Flags().SetInterspersed(true)

	if err := rootCmd.Execute(); err != nil {
		logger.Error("%v", err)
		os.Exit(1)
	}
}
