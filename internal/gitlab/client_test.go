package gitlab

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		token     string
		timeout   time.Duration
		wantError bool
	}{
		{
			name:      "valid configuration",
			url:       "https://gitlab.com",
			token:     "test-token",
			timeout:   5 * time.Second,
			wantError: false,
		},
		{
			name:      "empty token",
			url:       "https://gitlab.com",
			token:     "",
			timeout:   5 * time.Second,
			wantError: false, // gitlab.NewClient accepts empty token
		},
		{
			name:      "invalid URL",
			url:       "://invalid-url",
			token:     "test-token",
			timeout:   5 * time.Second,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New(tt.url, tt.token, tt.timeout)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if client == nil {
					t.Error("Expected non-nil client")
				}
			}
		})
	}
}

func TestFetchAllProjects_SinglePage(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header
		if r.Header.Get("PRIVATE-TOKEN") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return single page of projects
		w.Header().Set("X-Total-Pages", "1")
		w.Header().Set("X-Total", "2")
		w.Header().Set("Content-Type", "application/json")

		projects := []map[string]interface{}{
			{
				"id":                  1,
				"path_with_namespace": "group/project1",
				"name":                "Project 1",
				"description":         "Description 1",
			},
			{
				"id":                  2,
				"path_with_namespace": "group/project2",
				"name":                "Project 2",
				"description":         "Description 2",
			},
		}

		json.NewEncoder(w).Encode(projects)
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	projects, err := client.FetchAllProjects(nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(projects))
	}

	if projects[0].Path != "group/project1" {
		t.Errorf("Expected path 'group/project1', got '%s'", projects[0].Path)
	}

	if projects[0].Name != "Project 1" {
		t.Errorf("Expected name 'Project 1', got '%s'", projects[0].Name)
	}
}

func TestFetchAllProjects_MultiplePages(t *testing.T) {
	// Create mock server that returns 3 pages
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		pageNum, _ := strconv.Atoi(page)
		if pageNum == 0 {
			pageNum = 1
		}

		w.Header().Set("X-Total-Pages", "3")
		w.Header().Set("X-Total", "5")
		w.Header().Set("Content-Type", "application/json")

		var projects []map[string]interface{}
		switch pageNum {
		case 1:
			projects = []map[string]interface{}{
				{"id": 1, "path_with_namespace": "group/p1", "name": "P1", "description": "D1"},
				{"id": 2, "path_with_namespace": "group/p2", "name": "P2", "description": "D2"},
			}
		case 2:
			projects = []map[string]interface{}{
				{"id": 3, "path_with_namespace": "group/p3", "name": "P3", "description": "D3"},
				{"id": 4, "path_with_namespace": "group/p4", "name": "P4", "description": "D4"},
			}
		case 3:
			projects = []map[string]interface{}{
				{"id": 5, "path_with_namespace": "group/p5", "name": "P5", "description": "D5"},
			}
		}

		json.NewEncoder(w).Encode(projects)
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	projects, err := client.FetchAllProjects(nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(projects) != 5 {
		t.Errorf("Expected 5 projects, got %d", len(projects))
	}

	// Verify order is preserved (page 1, page 2, page 3)
	expectedPaths := []string{"group/p1", "group/p2", "group/p3", "group/p4", "group/p5"}
	for i, expected := range expectedPaths {
		if projects[i].Path != expected {
			t.Errorf("Project %d: expected path '%s', got '%s'", i, expected, projects[i].Path)
		}
	}
}

func TestFetchAllProjects_IncrementalSync(t *testing.T) {
	since := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	var capturedSince string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture the last_activity_after parameter
		capturedSince = r.URL.Query().Get("last_activity_after")

		w.Header().Set("X-Total-Pages", "1")
		w.Header().Set("X-Total", "1")
		w.Header().Set("Content-Type", "application/json")

		projects := []map[string]interface{}{
			{
				"id":                  1,
				"path_with_namespace": "group/project",
				"name":                "Project",
				"description":         "Desc",
			},
		}

		json.NewEncoder(w).Encode(projects)
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.FetchAllProjects(&since)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify the last_activity_after parameter was sent
	if capturedSince == "" {
		t.Fatal("Expected last_activity_after parameter to be set")
	}

	// Parse and verify the timestamp
	parsedSince, err := time.Parse(time.RFC3339, capturedSince)
	if err != nil {
		t.Fatalf("Failed to parse last_activity_after: %v", err)
	}

	if !parsedSince.Equal(since) {
		t.Errorf("Expected last_activity_after to be %v, got %v", since, parsedSince)
	}
}

func TestFetchAllProjects_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.FetchAllProjects(nil)
	if err == nil {
		t.Fatal("Expected error but got none")
	}
}

func TestFetchAllProjects_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Total-Pages", "1")
		w.Header().Set("X-Total", "0")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	projects, err := client.FetchAllProjects(nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(projects) != 0 {
		t.Errorf("Expected 0 projects, got %d", len(projects))
	}
}

func TestTestConnection_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/user" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       1,
				"username": "testuser",
				"name":     "Test User",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.TestConnection()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func TestTestConnection_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "401 Unauthorized",
		})
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	err = client.TestConnection()
	if err == nil {
		t.Fatal("Expected error but got none")
	}
}

func TestFetchAllProjects_Timeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Header().Set("X-Total-Pages", "1")
		w.Header().Set("X-Total", "1")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	// Create client with very short timeout
	client, err := New(server.URL, "test-token", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.FetchAllProjects(nil)
	if err == nil {
		t.Fatal("Expected timeout error but got none")
	}
}

func TestFetchAllProjects_ParallelPagination(t *testing.T) {
	// Test that parallel pagination works correctly with many pages
	const totalPages = 10
	requestOrder := make(chan int, totalPages)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		pageNum, _ := strconv.Atoi(page)
		if pageNum == 0 {
			pageNum = 1
		}

		// Track request order
		requestOrder <- pageNum

		w.Header().Set("X-Total-Pages", fmt.Sprintf("%d", totalPages))
		w.Header().Set("X-Total", fmt.Sprintf("%d", totalPages*2))
		w.Header().Set("Content-Type", "application/json")

		projects := []map[string]interface{}{
			{
				"id":                  pageNum*10 + 1,
				"path_with_namespace": fmt.Sprintf("group/p%d-1", pageNum),
				"name":                fmt.Sprintf("P%d-1", pageNum),
				"description":         fmt.Sprintf("D%d-1", pageNum),
			},
			{
				"id":                  pageNum*10 + 2,
				"path_with_namespace": fmt.Sprintf("group/p%d-2", pageNum),
				"name":                fmt.Sprintf("P%d-2", pageNum),
				"description":         fmt.Sprintf("D%d-2", pageNum),
			},
		}

		json.NewEncoder(w).Encode(projects)
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	projects, err := client.FetchAllProjects(nil)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	close(requestOrder)

	// Verify we got all projects
	if len(projects) != totalPages*2 {
		t.Errorf("Expected %d projects, got %d", totalPages*2, len(projects))
	}

	// Verify results are in correct order (page 1, page 2, ..., page 10)
	for i := 0; i < totalPages; i++ {
		expectedPath1 := fmt.Sprintf("group/p%d-1", i+1)
		expectedPath2 := fmt.Sprintf("group/p%d-2", i+1)

		if projects[i*2].Path != expectedPath1 {
			t.Errorf("Project %d: expected path '%s', got '%s'", i*2, expectedPath1, projects[i*2].Path)
		}
		if projects[i*2+1].Path != expectedPath2 {
			t.Errorf("Project %d: expected path '%s', got '%s'", i*2+1, expectedPath2, projects[i*2+1].Path)
		}
	}

	// Verify that requests were made (some likely in parallel)
	requestCount := 0
	for range requestOrder {
		requestCount++
	}
	if requestCount != totalPages {
		t.Errorf("Expected %d requests, got %d", totalPages, requestCount)
	}
}

func TestGetCurrentUsername_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v4/user" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       42,
				"username": "johndoe",
				"name":     "John Doe",
				"email":    "john@example.com",
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	username, err := client.GetCurrentUsername()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if username != "johndoe" {
		t.Errorf("Expected username 'johndoe', got '%s'", username)
	}
}

func TestGetCurrentUsername_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "401 Unauthorized",
		})
	}))
	defer server.Close()

	client, err := New(server.URL, "test-token", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	_, err = client.GetCurrentUsername()
	if err == nil {
		t.Fatal("Expected error but got none")
	}

	// Verify error message contains expected text
	if !contains(err.Error(), "failed to fetch current user") {
		t.Errorf("Expected 'failed to fetch current user' in error, got: %v", err)
	}
}

// Helper function for substring matching
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
