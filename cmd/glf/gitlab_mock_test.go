package main

import (
	"time"

	"github.com/igusev/glf/internal/types"
)

// mockGitLabClient is a mock implementation of gitlab.GitLabClient for testing
type mockGitLabClient struct {
	fetchProjectsFunc  func(*time.Time) ([]types.Project, error)
	testConnectionFunc func() error
	getUsernameFunc    func() (string, error)
}

// FetchAllProjects calls the mock function if set, otherwise returns empty list
func (m *mockGitLabClient) FetchAllProjects(since *time.Time) ([]types.Project, error) {
	if m.fetchProjectsFunc != nil {
		return m.fetchProjectsFunc(since)
	}
	return []types.Project{}, nil
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
