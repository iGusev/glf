package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

// ErrConfigNotFound is returned when no configuration is found
var ErrConfigNotFound = errors.New("configuration not found")

// Config holds the application configuration
type Config struct {
	GitLab        GitLabConfig `mapstructure:"gitlab"`
	Cache         CacheConfig  `mapstructure:"cache"`
	ExcludedPaths []string     `mapstructure:"excluded_paths"`
}

// GitLabConfig holds GitLab-specific settings
type GitLabConfig struct {
	URL     string `mapstructure:"url"`
	Token   string `mapstructure:"token"`
	Timeout int    `mapstructure:"timeout"` // timeout in seconds
}

// CacheConfig holds cache-specific settings
type CacheConfig struct {
	Dir string `mapstructure:"dir"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Set config file paths
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "glf")
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)
	viper.AddConfigPath(".") // Also check current directory

	// Set environment variable prefix
	viper.SetEnvPrefix("GLF")
	viper.AutomaticEnv()

	// Set defaults
	cacheDir := filepath.Join(os.Getenv("HOME"), ".cache", "glf")
	viper.SetDefault("cache.dir", cacheDir)
	viper.SetDefault("gitlab.timeout", 30) // Default 30 seconds timeout

	// Try to read config file (it's okay if it doesn't exist)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Expand tilde in cache dir path
	if cfg.Cache.Dir != "" {
		cfg.Cache.Dir = expandPath(cfg.Cache.Dir)
	}

	// Validate required fields
	if cfg.GitLab.URL == "" {
		return nil, ErrConfigNotFound
	}
	if cfg.GitLab.Token == "" {
		return nil, ErrConfigNotFound
	}

	// Validate timeout
	if cfg.GitLab.Timeout <= 0 {
		cfg.GitLab.Timeout = 30 // Fallback to default
	}

	return &cfg, nil
}

// GetTimeout returns the GitLab API timeout as time.Duration
func (c *GitLabConfig) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}

// expandPath expands ~ to home directory in paths
func expandPath(path string) string {
	if len(path) > 0 && path[0] == '~' {
		home := os.Getenv("HOME")
		if len(path) == 1 {
			return home
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

// EnsureConfigDir ensures the config directory exists
func EnsureConfigDir() error {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "glf")
	return os.MkdirAll(configDir, 0755)
}

// ExampleConfigPath returns the path where the example config should be created
func ExampleConfigPath() string {
	return filepath.Join(os.Getenv("HOME"), ".config", "glf", "config.yaml.example")
}

// IsExcluded checks if a project path matches any excluded pattern
func (c *Config) IsExcluded(projectPath string) bool {
	for _, pattern := range c.ExcludedPaths {
		// Support prefix matching for patterns ending with /*
		// e.g., "evernum-server/*" matches "evernum-server/api/avatar"
		if len(pattern) > 2 && pattern[len(pattern)-2:] == "/*" {
			prefix := pattern[:len(pattern)-2] + "/"
			if len(projectPath) >= len(prefix) && projectPath[:len(prefix)] == prefix {
				return true
			}
		} else {
			// Use filepath.Match for exact patterns or simple wildcards
			matched, err := filepath.Match(pattern, projectPath)
			if err == nil && matched {
				return true
			}
		}
	}
	return false
}

// AddExclusion adds a new exclusion pattern if it doesn't already exist
func (c *Config) AddExclusion(pattern string) error {
	// Check if pattern already exists
	for _, existing := range c.ExcludedPaths {
		if existing == pattern {
			return nil // Already exists
		}
	}

	c.ExcludedPaths = append(c.ExcludedPaths, pattern)
	return c.Save()
}

// RemoveExclusion removes an exclusion pattern
func (c *Config) RemoveExclusion(pattern string) error {
	newExcluded := make([]string, 0, len(c.ExcludedPaths))
	for _, p := range c.ExcludedPaths {
		if p != pattern {
			newExcluded = append(newExcluded, p)
		}
	}
	c.ExcludedPaths = newExcluded
	return c.Save()
}

// RemoveExclusionForPath removes any exclusion pattern that matches the given path
func (c *Config) RemoveExclusionForPath(projectPath string) error {
	newExcluded := make([]string, 0, len(c.ExcludedPaths))
	changed := false
	for _, pattern := range c.ExcludedPaths {
		matched := false
		// Support prefix matching for patterns ending with /*
		if len(pattern) > 2 && pattern[len(pattern)-2:] == "/*" {
			prefix := pattern[:len(pattern)-2] + "/"
			if len(projectPath) >= len(prefix) && projectPath[:len(prefix)] == prefix {
				matched = true
			}
		} else {
			// Use filepath.Match for exact patterns or simple wildcards
			m, err := filepath.Match(pattern, projectPath)
			if err == nil && m {
				matched = true
			}
		}

		if matched {
			changed = true
			continue // Skip this pattern (remove it)
		}
		newExcluded = append(newExcluded, pattern)
	}

	if changed {
		c.ExcludedPaths = newExcluded
		return c.Save()
	}
	return nil
}

// Save saves the current configuration to file
func (c *Config) Save() error {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "glf")
	configPath := filepath.Join(configDir, "config.yaml")

	// Ensure config dir exists
	if err := EnsureConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Set all config values in viper
	viper.Set("gitlab.url", c.GitLab.URL)
	viper.Set("gitlab.token", c.GitLab.Token)
	viper.Set("gitlab.timeout", c.GitLab.Timeout)
	viper.Set("cache.dir", c.Cache.Dir)
	viper.Set("excluded_paths", c.ExcludedPaths)

	// Write to file
	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CreateExampleConfig creates an example configuration file
func CreateExampleConfig() error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	exampleConfig := `# GLF Configuration File
# Place this file at ~/.config/glf/config.yaml

gitlab:
  # GitLab instance URL (required)
  url: "https://gitlab.example.com"

  # Personal Access Token (required)
  # Create one at: https://gitlab.example.com/-/user_settings/personal_access_tokens
  # Required scopes: read_api, read_repository
  token: "your-gitlab-token-here"

  # API timeout in seconds (optional, defaults to 30)
  timeout: 30

cache:
  # Cache directory (optional, defaults to ~/.cache/glf)
  dir: "~/.cache/glf"

# Excluded project paths (supports wildcards)
# Use Ctrl+X in TUI to add current project
# Use Ctrl+H to toggle showing excluded projects
excluded_paths:
  # - "archived-projects/*"
  # - "legacy/*"
  # - "namespace/specific-project"

# Environment variables can also be used:
# GLF_GITLAB_URL=https://gitlab.example.com
# GLF_GITLAB_TOKEN=your-token-here
`

	examplePath := ExampleConfigPath()
	return os.WriteFile(examplePath, []byte(exampleConfig), 0644)
}
