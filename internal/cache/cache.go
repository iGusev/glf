package cache

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/igusev/glf/internal/types"
)

const projectsFileName = "projects.txt"

// Cache manages the local project cache
type Cache struct {
	dir string
}

// New creates a new Cache instance
func New(dir string) *Cache {
	return &Cache{dir: dir}
}

// EnsureDir ensures the cache directory exists
func (c *Cache) EnsureDir() error {
	return os.MkdirAll(c.dir, 0755)
}

// ProjectsPath returns the full path to the projects cache file
func (c *Cache) ProjectsPath() string {
	return filepath.Join(c.dir, projectsFileName)
}

// WriteProjects writes a list of projects to the cache
// Format: path|name|description (one per line, description may be empty)
func (c *Cache) WriteProjects(projects []types.Project) error {
	if err := c.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	f, err := os.Create(c.ProjectsPath())
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	for _, project := range projects {
		// Escape newlines and pipe characters in description
		desc := strings.ReplaceAll(project.Description, "\n", " ")
		desc = strings.ReplaceAll(desc, "|", "\\|")
		line := fmt.Sprintf("%s|%s|%s\n", project.Path, project.Name, desc)
		if _, err := writer.WriteString(line); err != nil {
			return fmt.Errorf("failed to write project: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush cache file: %w", err)
	}

	return nil
}

// ReadProjects reads the list of projects from cache
// Format: path|name|description (one per line, description may be empty)
// Also supports old format: path|name (for backward compatibility)
func (c *Cache) ReadProjects() ([]types.Project, error) {
	f, err := os.Open(c.ProjectsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("cache file not found, run 'glf sync' first")
		}
		return nil, fmt.Errorf("failed to open cache file: %w", err)
	}
	defer f.Close()

	var projects []types.Project
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Parse format: path|name|description (or old format: path|name)
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 2 {
			// Skip malformed lines
			continue
		}

		project := types.Project{
			Path: parts[0],
			Name: parts[1],
		}

		// If description field exists, unescape it
		if len(parts) >= 3 {
			desc := parts[2]
			desc = strings.ReplaceAll(desc, "\\|", "|")
			project.Description = desc
		}

		projects = append(projects, project)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	return projects, nil
}

// Stats returns cache statistics
func (c *Cache) Stats() (int, error) {
	projects, err := c.ReadProjects()
	if err != nil {
		return 0, err
	}
	return len(projects), nil
}

// Exists checks if the cache file exists
func (c *Cache) Exists() bool {
	_, err := os.Stat(c.ProjectsPath())
	return err == nil
}

// SaveLastSyncTime saves the last successful sync timestamp
func (c *Cache) SaveLastSyncTime(t time.Time) error {
	if err := c.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	timestampPath := filepath.Join(c.dir, ".last_sync_time")
	data := []byte(t.Format(time.RFC3339))

	if err := os.WriteFile(timestampPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save sync timestamp: %w", err)
	}

	return nil
}

// LoadLastSyncTime loads the last successful sync timestamp
// Returns zero time if file doesn't exist (first sync)
func (c *Cache) LoadLastSyncTime() (time.Time, error) {
	timestampPath := filepath.Join(c.dir, ".last_sync_time")

	data, err := os.ReadFile(timestampPath)
	if err != nil {
		if os.IsNotExist(err) {
			// First sync - return zero time
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to read sync timestamp: %w", err)
	}

	t, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse sync timestamp: %w", err)
	}

	return t, nil
}

// SaveLastFullSyncTime saves the last successful full sync timestamp
func (c *Cache) SaveLastFullSyncTime(t time.Time) error {
	if err := c.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	timestampPath := filepath.Join(c.dir, ".last_full_sync_time")
	data := []byte(t.Format(time.RFC3339))

	if err := os.WriteFile(timestampPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save full sync timestamp: %w", err)
	}

	return nil
}

// LoadLastFullSyncTime loads the last successful full sync timestamp
// Returns zero time if file doesn't exist (never had full sync)
func (c *Cache) LoadLastFullSyncTime() (time.Time, error) {
	timestampPath := filepath.Join(c.dir, ".last_full_sync_time")

	data, err := os.ReadFile(timestampPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Never had full sync - return zero time
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("failed to read full sync timestamp: %w", err)
	}

	t, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse full sync timestamp: %w", err)
	}

	return t, nil
}

// SaveUsername saves the GitLab username to cache
func (c *Cache) SaveUsername(username string) error {
	if err := c.EnsureDir(); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	usernamePath := filepath.Join(c.dir, ".username")
	data := []byte(username)

	if err := os.WriteFile(usernamePath, data, 0644); err != nil {
		return fmt.Errorf("failed to save username: %w", err)
	}

	return nil
}

// LoadUsername loads the GitLab username from cache
// Returns empty string if file doesn't exist
func (c *Cache) LoadUsername() (string, error) {
	usernamePath := filepath.Join(c.dir, ".username")

	data, err := os.ReadFile(usernamePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No cached username
			return "", nil
		}
		return "", fmt.Errorf("failed to read username: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}
