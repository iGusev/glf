// Package model defines core data structures for GitLab projects
package model

import "strings"

// Project represents a GitLab project with its path, name and description
type Project struct {
	Path        string // PathWithNamespace (e.g., "company/group/subgroup/project-name")
	Name        string // Project name (e.g., "project-name")
	Description string // Project description (may be empty)
	Starred     bool   // Whether the project is starred by the user
	Archived    bool   // Whether the project is archived
	Member      bool   // Whether the user is a member of this project
}

// SearchableString returns a combined string for fuzzy searching
// Format: "path/name" - this gives priority to project name in search
// Example: "company/group/subgroup/project-name"
func (p Project) SearchableString() string {
	return p.Path + "/" + p.Name
}

// DisplayString returns formatted display string in style: [namespace] > project-name
// For path "company/group/subgroup/myproject" and name "myproject"
// Returns: "[company/group/subgroup] > myproject"
func (p Project) DisplayString() string {
	// Remove last segment from path (project slug), keep all groups including root
	// company/group/subgroup/myproject -> company/group/subgroup
	parts := strings.Split(p.Path, "/")
	if len(parts) > 1 {
		// Take all parts except the last one (the namespace)
		namespace := strings.Join(parts[:len(parts)-1], "/")
		return "[" + namespace + "] > " + p.Name
	}
	// Fallback: just return name if single part (no namespace)
	return p.Name
}
