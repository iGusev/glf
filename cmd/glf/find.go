package main

import (
	"github.com/spf13/cobra"
)

var findCmd = &cobra.Command{
	Use:   "find [query...]",
	Short: "Search for GitLab projects in cache (alias for direct search)",
	Long: `Search for GitLab projects using a simple text filter.
Supports multi-word queries with AND logic.
If no query is provided, all cached projects are listed.

This command is an alias for the direct search: 'glf <query>'
You can use either 'glf find backend' or just 'glf backend'

Examples:
  glf find backend
  glf find api ingress
  glf find payment gateway service`,
	RunE: runSearch, // Use the same function as root command (handles multi-word queries)
}

func init() {
	rootCmd.AddCommand(findCmd)
}
