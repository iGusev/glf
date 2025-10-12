package types

import "strings"

// Project represents a GitLab project with its path, name and description
type Project struct {
	Path        string // PathWithNamespace (e.g., "numbuster/api/payment/payselection/callback")
	Name        string // Project name (e.g., "payselection-callback")
	Description string // Project description (may be empty)
}

// SearchableString returns a combined string for fuzzy searching
// Format: "path/name" - this gives priority to project name in search
// Example: "numbuster/api/payment/payselection/payselection-callback"
func (p Project) SearchableString() string {
	return p.Path + "/" + p.Name
}

// DisplayString returns formatted display string in style: [namespace] > project-name
// For path "numbuster/api/payment/payselection/callback" and name "payselection-callback"
// Returns: "[numbuster/api/payment/payselection] > payselection-callback"
func (p Project) DisplayString() string {
	// Remove last segment from path (project slug), keep all groups including root
	// numbuster/api/payment/payselection/callback -> numbuster/api/payment/payselection
	parts := strings.Split(p.Path, "/")
	if len(parts) > 1 {
		// Take all parts except the last one (the namespace)
		namespace := strings.Join(parts[:len(parts)-1], "/")
		return "[" + namespace + "] > " + p.Name
	}
	// Fallback: just return name if single part (no namespace)
	return p.Name
}
