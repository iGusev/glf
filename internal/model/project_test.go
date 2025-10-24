package model

import "testing"

func TestProject_SearchableString(t *testing.T) {
	tests := []struct {
		name     string
		project  Project
		expected string
	}{
		{
			name: "full project with path and name",
			project: Project{
				Path: "group/subgroup/project",
				Name: "my-project",
			},
			expected: "group/subgroup/project/my-project",
		},
		{
			name: "single level path",
			project: Project{
				Path: "group/project",
				Name: "my-project",
			},
			expected: "group/project/my-project",
		},
		{
			name: "root level project",
			project: Project{
				Path: "project",
				Name: "my-project",
			},
			expected: "project/my-project",
		},
		{
			name: "deep nested path",
			project: Project{
				Path: "org/team/subteam/area/subarea/project",
				Name: "deep-project",
			},
			expected: "org/team/subteam/area/subarea/project/deep-project",
		},
		{
			name: "empty name",
			project: Project{
				Path: "group/project",
				Name: "",
			},
			expected: "group/project/",
		},
		{
			name: "empty path",
			project: Project{
				Path: "",
				Name: "project",
			},
			expected: "/project",
		},
		{
			name: "both empty",
			project: Project{
				Path: "",
				Name: "",
			},
			expected: "/",
		},
		{
			name: "with cyrillic characters",
			project: Project{
				Path: "группа/проект",
				Name: "мой-проект",
			},
			expected: "группа/проект/мой-проект",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.project.SearchableString()
			if result != tt.expected {
				t.Errorf("SearchableString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestProject_DisplayString(t *testing.T) {
	tests := []struct {
		name     string
		project  Project
		expected string
	}{
		{
			name: "multi-level path - normal case",
			project: Project{
				Path: "company/group/subgroup/service/callback",
				Name: "service-callback",
			},
			expected: "[company/group/subgroup/service] > service-callback",
		},
		{
			name: "two-level path",
			project: Project{
				Path: "group/project",
				Name: "my-project",
			},
			expected: "[group] > my-project",
		},
		{
			name: "single-level path (no namespace)",
			project: Project{
				Path: "project",
				Name: "my-project",
			},
			expected: "my-project",
		},
		{
			name: "deep nested path",
			project: Project{
				Path: "org/team/subteam/area/subarea/project",
				Name: "deep-project",
			},
			expected: "[org/team/subteam/area/subarea] > deep-project",
		},
		{
			name: "empty path - fallback to name",
			project: Project{
				Path: "",
				Name: "standalone-project",
			},
			expected: "standalone-project",
		},
		{
			name: "path with slash only",
			project: Project{
				Path: "/",
				Name: "root-project",
			},
			expected: "[] > root-project",
		},
		{
			name: "path ending with slash",
			project: Project{
				Path: "group/project/",
				Name: "trailing-slash",
			},
			expected: "[group/project] > trailing-slash",
		},
		{
			name: "cyrillic namespace and name",
			project: Project{
				Path: "организация/команда/проект",
				Name: "мой-проект",
			},
			expected: "[организация/команда] > мой-проект",
		},
		{
			name: "name with description (description not used)",
			project: Project{
				Path:        "group/subgroup/project",
				Name:        "test-project",
				Description: "This is a description",
			},
			expected: "[group/subgroup] > test-project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.project.DisplayString()
			if result != tt.expected {
				t.Errorf("DisplayString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestProject_SearchableString_Consistency(t *testing.T) {
	// Test that SearchableString is consistent across multiple calls
	project := Project{
		Path: "group/subgroup/project",
		Name: "my-project",
	}

	result1 := project.SearchableString()
	result2 := project.SearchableString()

	if result1 != result2 {
		t.Errorf("SearchableString() not consistent: first=%q, second=%q", result1, result2)
	}
}

func TestProject_DisplayString_Consistency(t *testing.T) {
	// Test that DisplayString is consistent across multiple calls
	project := Project{
		Path: "group/subgroup/project",
		Name: "my-project",
	}

	result1 := project.DisplayString()
	result2 := project.DisplayString()

	if result1 != result2 {
		t.Errorf("DisplayString() not consistent: first=%q, second=%q", result1, result2)
	}
}
