package main

import (
	"time"

	"github.com/igusev/glf/internal/model"
)

// mockGitLabClient is a mock implementation of gitlab.GitLabClient for testing
type mockGitLabClient struct {
	fetchProjectsFunc  func(*time.Time, bool) ([]model.Project, error)
	testConnectionFunc func() error
	getUsernameFunc    func() (string, error)
}

// FetchAllProjects calls the mock function if set, otherwise returns empty list with Member=true
func (m *mockGitLabClient) FetchAllProjects(since *time.Time, membership bool) ([]model.Project, error) {
	if m.fetchProjectsFunc != nil {
		return m.fetchProjectsFunc(since, membership)
	}
	// Return empty list with Member=true as default for tests
	return []model.Project{}, nil
}

// TestConnection calls the mock function if set, otherwise returns nil
func (m *mockGitLabClient) TestConnection() error {
	if m.testConnectionFunc != nil {
		return m.testConnectionFunc()
	}
	return nil
}

// GetCurrentUsername calls the mock function if set, otherwise returns empty string
func (m *mockGitLabClient) GetCurrentUsername() (string, error) {
	if m.getUsernameFunc != nil {
		return m.getUsernameFunc()
	}
	return "", nil
}
