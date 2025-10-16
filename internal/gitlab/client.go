package gitlab

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/igusev/glf/internal/logger"
	"github.com/igusev/glf/internal/types"
	"github.com/xanzy/go-gitlab"
)

// GitLabClient defines the interface for GitLab API operations
// This interface enables mocking in tests while maintaining production functionality
//
//nolint:revive // GitLabClient is intentional - distinguishes interface from concrete Client struct
type GitLabClient interface {
	FetchAllProjects(since *time.Time) ([]types.Project, error)
	TestConnection() error
	GetCurrentUsername() (string, error)
}

// Client wraps the GitLab API client and implements GitLabClient interface
type Client struct {
	client *gitlab.Client
}

// New creates a new GitLab client with timeout
func New(url, token string, timeout time.Duration) (*Client, error) {
	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: timeout,
	}

	// Create GitLab client with custom HTTP client
	client, err := gitlab.NewClient(
		token,
		gitlab.WithBaseURL(url),
		gitlab.WithHTTPClient(httpClient),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &Client{client: client}, nil
}

// FetchAllProjects fetches all accessible projects from GitLab using parallel pagination
// If since is provided, only fetches projects with last_activity_after >= since (incremental sync)
// Returns a slice of Project structs containing path, name, and starred information
func (c *Client) FetchAllProjects(since *time.Time) ([]types.Project, error) {
	// Step 0: Fetch starred projects
	logger.Debug("Fetching starred projects...")
	starredProjects, err := c.FetchStarredProjects()
	if err != nil {
		logger.Debug("Warning: failed to fetch starred projects: %v", err)
		starredProjects = make(map[string]bool)
	}

	// Step 1: Make initial request to get total pages
	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100, // Maximum allowed per page
			Page:    1,
		},
		Membership: gitlab.Ptr(true), // Only projects the user is a member of
		Simple:     gitlab.Ptr(true), // Return only limited fields for performance
	}

	// Add incremental sync filter if timestamp provided
	if since != nil && !since.IsZero() {
		opt.LastActivityAfter = since
		logger.Debug("Incremental sync: fetching projects changed after %s", since.Format(time.RFC3339))
	} else {
		logger.Debug("Full sync: fetching all projects")
	}

	// First request to get pagination info
	firstPageProjects, resp, err := c.client.Projects.ListProjects(opt)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects (first page): %w", err)
	}

	totalPages := resp.TotalPages
	totalProjects := resp.TotalItems

	logger.Debug("Total pages: %d, Total projects: %d", totalPages, totalProjects)

	if totalPages <= 1 {
		// Only one page, return immediately
		var result []types.Project
		for _, project := range firstPageProjects {
			result = append(result, types.Project{
				Path:        project.PathWithNamespace,
				Name:        project.Name,
				Description: project.Description,
				Starred:     starredProjects[project.PathWithNamespace],
			})
		}
		logger.Debug("Single page, fetched %d projects", len(result))
		return result, nil
	}

	// Step 2: Parallel fetch remaining pages
	const maxConcurrent = 10 // Limit concurrent requests to avoid overwhelming the server

	logger.Debug("Starting parallel fetch: %d pages with max %d concurrent requests", totalPages, maxConcurrent)
	startTime := time.Now()

	type pageResult struct {
		projects []types.Project
		err      error
		page     int
	}

	// Channel to collect results
	results := make(chan pageResult, totalPages)

	// Semaphore to limit concurrent requests
	semaphore := make(chan struct{}, maxConcurrent)

	// WaitGroup to wait for all goroutines
	var wg sync.WaitGroup

	// Counter for completed pages (for progress logging)
	var completedPages int32

	// Add first page to results
	var firstPageProjs []types.Project
	for _, project := range firstPageProjects {
		firstPageProjs = append(firstPageProjs, types.Project{
			Path:        project.PathWithNamespace,
			Name:        project.Name,
			Description: project.Description,
			Starred:     starredProjects[project.PathWithNamespace],
		})
	}
	results <- pageResult{page: 1, projects: firstPageProjs, err: nil}
	atomic.AddInt32(&completedPages, 1)

	// Launch goroutines for pages 2..N
	for page := 2; page <= totalPages; page++ {
		wg.Add(1)
		go func(pageNum int) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Create options for this page (preserve incremental filter)
			pageOpt := &gitlab.ListProjectsOptions{
				ListOptions: gitlab.ListOptions{
					PerPage: 100,
					Page:    pageNum,
				},
				Membership:        gitlab.Ptr(true),
				Simple:            gitlab.Ptr(true),
				LastActivityAfter: opt.LastActivityAfter, // Preserve incremental filter
			}

			// Fetch the page
			projects, _, err := c.client.Projects.ListProjects(pageOpt)
			if err != nil {
				results <- pageResult{page: pageNum, projects: nil, err: err}
				return
			}

			// Extract projects
			var projs []types.Project
			for _, project := range projects {
				projs = append(projs, types.Project{
					Path:        project.PathWithNamespace,
					Name:        project.Name,
					Description: project.Description,
					Starred:     starredProjects[project.PathWithNamespace],
				})
			}

			results <- pageResult{page: pageNum, projects: projs, err: nil}

			// Log progress with integer overflow protection
			completed := atomic.AddInt32(&completedPages, 1)
			logger.Debug("Fetched page %d/%d (%d%%)", completed, totalPages, (int(completed)*100)/totalPages)
		}(page)
	}

	// Close results channel after all goroutines finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Step 3: Collect all results
	pageMap := make(map[int][]types.Project)
	for result := range results {
		if result.err != nil {
			return nil, fmt.Errorf("failed to fetch page %d: %w", result.page, result.err)
		}
		pageMap[result.page] = result.projects
	}

	// Step 4: Combine results in correct order
	var allProjects []types.Project
	for page := 1; page <= totalPages; page++ {
		allProjects = append(allProjects, pageMap[page]...)
	}

	elapsed := time.Since(startTime)
	logger.Debug("Parallel fetch completed in %v: fetched %d projects from %d pages", elapsed, len(allProjects), totalPages)

	return allProjects, nil
}

// TestConnection tests the connection to GitLab by fetching current user
func (c *Client) TestConnection() error {
	_, _, err := c.client.Users.CurrentUser()
	if err != nil {
		return fmt.Errorf("failed to connect to GitLab: %w", err)
	}
	return nil
}

// GetCurrentUsername fetches the username of the authenticated user
func (c *Client) GetCurrentUsername() (string, error) {
	user, _, err := c.client.Users.CurrentUser()
	if err != nil {
		return "", fmt.Errorf("failed to fetch current user: %w", err)
	}
	return user.Username, nil
}

// FetchStarredProjects fetches all projects starred by the current user
// Returns a map of project PathWithNamespace → true for O(1) lookup
func (c *Client) FetchStarredProjects() (map[string]bool, error) {
	result := make(map[string]bool)

	// Step 1: Make initial request to get total pages
	opt := &gitlab.ListProjectsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Starred: gitlab.Ptr(true), // Only starred projects
		Simple:  gitlab.Ptr(true), // Return only limited fields
	}

	// First request to get pagination info
	firstPageProjects, resp, err := c.client.Projects.ListProjects(opt)
	if err != nil {
		// Don't fail completely, just log warning and return empty map
		logger.Debug("Warning: Failed to fetch starred projects: %v", err)
		return result, nil
	}

	totalPages := resp.TotalPages
	logger.Debug("Fetching starred projects: %d pages", totalPages)

	// Add first page results
	for _, project := range firstPageProjects {
		result[project.PathWithNamespace] = true
	}

	if totalPages <= 1 {
		logger.Debug("Fetched %d starred projects", len(result))
		return result, nil
	}

	// Step 2: Parallel fetch remaining pages
	const maxConcurrent = 10

	type pageResult struct {
		paths []string
		page  int
	}

	results := make(chan pageResult, totalPages)
	semaphore := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	// Launch goroutines for pages 2..N
	for page := 2; page <= totalPages; page++ {
		wg.Add(1)
		go func(pageNum int) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			pageOpt := &gitlab.ListProjectsOptions{
				ListOptions: gitlab.ListOptions{
					PerPage: 100,
					Page:    pageNum,
				},
				Starred: gitlab.Ptr(true),
				Simple:  gitlab.Ptr(true),
			}

			projects, _, err := c.client.Projects.ListProjects(pageOpt)
			if err != nil {
				logger.Debug("Warning: Failed to fetch starred projects page %d: %v", pageNum, err)
				return
			}

			var paths []string
			for _, project := range projects {
				paths = append(paths, project.PathWithNamespace)
			}

			results <- pageResult{page: pageNum, paths: paths}
		}(page)
	}

	// Close results channel after all goroutines finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all results
	for pageRes := range results {
		for _, path := range pageRes.paths {
			result[path] = true
		}
	}

	logger.Debug("Fetched %d starred projects total", len(result))
	return result, nil
}
