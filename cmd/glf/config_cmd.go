package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/igusev/glf/internal/config"
	"github.com/igusev/glf/internal/gitlab"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure GitLab connection settings",
	Long: `Interactive configuration wizard to set up GitLab URL and access token.
Creates or updates the configuration file at ~/.config/glf/config.yaml`,
	RunE: runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("GLF Configuration Wizard")
	fmt.Println("========================")

	// Load existing config if available
	existingCfg, _ := config.Load()

	// Get GitLab URL
	fmt.Printf("GitLab URL")
	if existingCfg.GitLab.URL != "" {
		fmt.Printf(" [%s]", existingCfg.GitLab.URL)
	}
	fmt.Print(": ")

	url, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	url = strings.TrimSpace(url)

	// Use existing URL if user pressed Enter without input
	if url == "" && existingCfg.GitLab.URL != "" {
		url = existingCfg.GitLab.URL
	}

	if url == "" {
		return fmt.Errorf("GitLab URL is required")
	}

	// Get GitLab token
	fmt.Printf("GitLab Personal Access Token")
	if existingCfg.GitLab.Token != "" {
		fmt.Printf(" [%s]", maskToken(existingCfg.GitLab.Token))
	}
	fmt.Print(": ")

	token, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	token = strings.TrimSpace(token)

	// Use existing token if user pressed Enter without input
	if token == "" && existingCfg.GitLab.Token != "" {
		token = existingCfg.GitLab.Token
	}

	if token == "" {
		return fmt.Errorf("GitLab token is required")
	}

	// Get timeout (optional)
	fmt.Printf("API timeout in seconds")
	if existingCfg.GitLab.Timeout > 0 {
		fmt.Printf(" [%d]", existingCfg.GitLab.Timeout)
	} else {
		fmt.Printf(" [30]")
	}
	fmt.Print(": ")

	timeoutStr, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}
	timeoutStr = strings.TrimSpace(timeoutStr)

	timeout := 30
	if timeoutStr != "" {
		if _, err := fmt.Sscanf(timeoutStr, "%d", &timeout); err != nil {
			fmt.Printf("Warning: invalid timeout '%s', using default %d seconds\n", timeoutStr, 30)
			timeout = 30
		}
	} else if existingCfg.GitLab.Timeout > 0 {
		timeout = existingCfg.GitLab.Timeout
	}

	// Test connection
	fmt.Printf("\nTesting connection to %s...\n", url)

	cfg := &config.Config{
		GitLab: config.GitLabConfig{
			URL:     url,
			Token:   token,
			Timeout: timeout,
		},
		Cache: existingCfg.Cache,
	}

	client, err := gitlab.New(cfg.GitLab.URL, cfg.GitLab.Token, cfg.GitLab.GetTimeout())
	if err != nil {
		return fmt.Errorf("failed to create GitLab client: %w", err)
	}

	if err := client.TestConnection(); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	fmt.Println("✓ Connection successful!")

	// Save configuration
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "glf")
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("\n✓ Configuration saved to %s\n", configPath)
	fmt.Println("\nYou can now run 'glf sync' to fetch projects from GitLab.")

	return nil
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
