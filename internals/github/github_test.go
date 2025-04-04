package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockClient is a test implementation of GitHubClient interface
type MockClient struct {
	server     *httptest.Server
	token      string
	httpClient *http.Client
}

// NewMockClient creates a new mock GitHub client for testing
func NewMockClient(server *httptest.Server, token string) *MockClient {
	return &MockClient{
		server: server,
		token:  token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetRepository implements the GitHubClient interface for testing
func (m *MockClient) GetRepository(fullName string) (*Repository, error) {
	url := m.server.URL + "/repos/" + fullName
	req, err := m.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	var repo Repository
	if err := m.doRequest(req, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// GetCommits implements the GitHubClient interface for testing
func (m *MockClient) GetCommits(fullName string, page, perPage int, since time.Time) ([]Commit, bool, error) {
	sinceStr := since.Format(time.RFC3339)
	url := fmt.Sprintf("%s/repos/%s/commits?per_page=%d&page=%d&since=%s",
		m.server.URL, fullName, perPage, page, sinceStr)

	req, err := m.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	var commits []Commit
	if err := m.doRequest(req, &commits); err != nil {
		return nil, false, err
	}

	// Check if there are more pages
	hasMore := len(commits) == perPage
	return commits, hasMore, nil
}

// newRequest creates a new HTTP request with the necessary headers
func (m *MockClient) newRequest(method, url string, body *strings.Reader) (*http.Request, error) {
	var req *http.Request
	var err error
	if body != nil {
		req, err = http.NewRequest(method, url, body)
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	// Add auth header if token is available
	if m.token != "" {
		req.Header.Add("Authorization", "token "+m.token)
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	req.Header.Add("User-Agent", "GitHub-Data-Service")
	return req, nil
}

// doRequest performs the HTTP request and unmarshals the response
func (m *MockClient) doRequest(req *http.Request, v interface{}) error {
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}
	return nil
}

func TestGetRepository(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/repos/owner/repo" {
			t.Errorf("Expected path /repos/owner/repo, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "token test-token" {
			t.Errorf("Expected Authorization header 'token test-token', got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
			t.Errorf("Expected Accept header 'application/vnd.github.v3+json', got %s", r.Header.Get("Accept"))
		}
		if r.Header.Get("User-Agent") != "GitHub-Data-Service" {
			t.Errorf("Expected User-Agent header 'GitHub-Data-Service', got %s", r.Header.Get("User-Agent"))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		mockRepo := Repository{
			ID:              123,
			Name:            "repo",
			FullName:        "owner/repo",
			Description:     "Test repository",
			URL:             "https://github.com/owner/repo",
			Language:        "Go",
			ForksCount:      10,
			StargazersCount: 20,
			OpenIssuesCount: 5,
			WatchersCount:   15,
			CreatedAt:       time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:       time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		}
		json.NewEncoder(w).Encode(mockRepo)
	}))
	defer server.Close()

	// Create mock client with test server URL
	client := NewMockClient(server, "test-token")

	// Test GetRepository
	repo, err := client.GetRepository("owner/repo")
	if err != nil {
		t.Fatalf("GetRepository returned error: %v", err)
	}

	// Verify response
	if repo.ID != 123 {
		t.Errorf("Expected ID 123, got %d", repo.ID)
	}
	if repo.Name != "repo" {
		t.Errorf("Expected Name 'repo', got %s", repo.Name)
	}
	if repo.FullName != "owner/repo" {
		t.Errorf("Expected FullName 'owner/repo', got %s", repo.FullName)
	}
	if repo.Description != "Test repository" {
		t.Errorf("Expected Description 'Test repository', got %s", repo.Description)
	}
	if repo.URL != "https://github.com/owner/repo" {
		t.Errorf("Expected URL 'https://github.com/owner/repo', got %s", repo.URL)
	}
	if repo.Language != "Go" {
		t.Errorf("Expected Language 'Go', got %s", repo.Language)
	}
	if repo.ForksCount != 10 {
		t.Errorf("Expected ForksCount 10, got %d", repo.ForksCount)
	}
	if repo.StargazersCount != 20 {
		t.Errorf("Expected StargazersCount 20, got %d", repo.StargazersCount)
	}
	if repo.OpenIssuesCount != 5 {
		t.Errorf("Expected OpenIssuesCount 5, got %d", repo.OpenIssuesCount)
	}
	if repo.WatchersCount != 15 {
		t.Errorf("Expected WatchersCount 15, got %d", repo.WatchersCount)
	}
}

func TestGetCommits(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.URL.Path != "/repos/owner/repo/commits" {
			t.Errorf("Expected path /repos/owner/repo/commits, got %s", r.URL.Path)
		}

		// Verify query parameters
		query := r.URL.Query()
		if query.Get("per_page") != "2" {
			t.Errorf("Expected per_page=2, got %s", query.Get("per_page"))
		}
		if query.Get("page") != "1" {
			t.Errorf("Expected page=1, got %s", query.Get("page"))
		}

		// Verify headers
		if r.Header.Get("Authorization") != "token test-token" {
			t.Errorf("Expected Authorization header 'token test-token', got %s", r.Header.Get("Authorization"))
		}

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		mockCommits := []Commit{
			{
				SHA: "abc123",
				Commit: struct {
					Author struct {
						Name  string    `json:"name"`
						Email string    `json:"email"`
						Date  time.Time `json:"date"`
					} `json:"author"`
					Message string `json:"message"`
				}{
					Author: struct {
						Name  string    `json:"name"`
						Email string    `json:"email"`
						Date  time.Time `json:"date"`
					}{
						Name:  "Test Author",
						Email: "author@example.com",
						Date:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					Message: "Test commit message",
				},
				HTMLURL: "https://github.com/owner/repo/commit/abc123",
			},
			{
				SHA: "def456",
				Commit: struct {
					Author struct {
						Name  string    `json:"name"`
						Email string    `json:"email"`
						Date  time.Time `json:"date"`
					} `json:"author"`
					Message string `json:"message"`
				}{
					Author: struct {
						Name  string    `json:"name"`
						Email string    `json:"email"`
						Date  time.Time `json:"date"`
					}{
						Name:  "Another Author",
						Email: "another@example.com",
						Date:  time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC),
					},
					Message: "Another test commit",
				},
				HTMLURL: "https://github.com/owner/repo/commit/def456",
			},
		}
		json.NewEncoder(w).Encode(mockCommits)
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewMockClient(server, "test-token")

	// Test GetCommits
	since := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	commits, hasMore, err := client.GetCommits("owner/repo", 1, 2, since)
	if err != nil {
		t.Fatalf("GetCommits returned error: %v", err)
	}

	// Verify response
	if len(commits) != 2 {
		t.Errorf("Expected 2 commits, got %d", len(commits))
	}
	if !hasMore {
		t.Errorf("Expected hasMore to be true, got false")
	}

	// Verify first commit
	if commits[0].SHA != "abc123" {
		t.Errorf("Expected SHA 'abc123', got %s", commits[0].SHA)
	}
	if commits[0].Commit.Author.Name != "Test Author" {
		t.Errorf("Expected Author.Name 'Test Author', got %s", commits[0].Commit.Author.Name)
	}
	if commits[0].Commit.Author.Email != "author@example.com" {
		t.Errorf("Expected Author.Email 'author@example.com', got %s", commits[0].Commit.Author.Email)
	}
	if !commits[0].Commit.Author.Date.Equal(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("Expected Author.Date '2021-01-01', got %s", commits[0].Commit.Author.Date)
	}
	if commits[0].Commit.Message != "Test commit message" {
		t.Errorf("Expected Message 'Test commit message', got %s", commits[0].Commit.Message)
	}
	if commits[0].HTMLURL != "https://github.com/owner/repo/commit/abc123" {
		t.Errorf("Expected HTMLURL 'https://github.com/owner/repo/commit/abc123', got %s", commits[0].HTMLURL)
	}
}

func TestGetCommits_NoMorePages(t *testing.T) {
	// Setup test server that returns only one commit when per_page is 2
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return mock response with only one commit
		w.Header().Set("Content-Type", "application/json")
		mockCommits := []Commit{
			{
				SHA: "abc123",
				Commit: struct {
					Author struct {
						Name  string    `json:"name"`
						Email string    `json:"email"`
						Date  time.Time `json:"date"`
					} `json:"author"`
					Message string `json:"message"`
				}{
					Author: struct {
						Name  string    `json:"name"`
						Email string    `json:"email"`
						Date  time.Time `json:"date"`
					}{
						Name:  "Test Author",
						Email: "author@example.com",
						Date:  time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
					},
					Message: "Test commit message",
				},
				HTMLURL: "https://github.com/owner/repo/commit/abc123",
			},
		}
		json.NewEncoder(w).Encode(mockCommits)
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewMockClient(server, "test-token")

	// Test GetCommits
	since := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	commits, hasMore, err := client.GetCommits("owner/repo", 1, 2, since)
	if err != nil {
		t.Fatalf("GetCommits returned error: %v", err)
	}

	// Verify response
	if len(commits) != 1 {
		t.Errorf("Expected 1 commit, got %d", len(commits))
	}
	if hasMore {
		t.Errorf("Expected hasMore to be false, got true")
	}
}

func TestGetRepository_APIError(t *testing.T) {
	// Setup test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message": "Not Found"}`))
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewMockClient(server, "test-token")

	// Test GetRepository
	_, err := client.GetRepository("owner/nonexistent")
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestGetCommits_APIError(t *testing.T) {
	// Setup test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message": "Bad credentials"}`))
	}))
	defer server.Close()

	// Create client with test server URL
	client := NewMockClient(server, "invalid-token")

	// Test GetCommits
	since := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := client.GetCommits("owner/repo", 1, 10, since)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

// TestNewClient verifies that NewClient creates a client with the right configuration
func TestNewClient(t *testing.T) {
	client := NewClient("test-token")

	if client.token != "test-token" {
		t.Errorf("Expected token 'test-token', got %s", client.token)
	}

	if client.httpClient == nil {
		t.Error("Expected httpClient to be initialized, got nil")
	}

	if client.httpClient.Timeout != 10*time.Second {
		t.Errorf("Expected timeout of 10 seconds, got %v", client.httpClient.Timeout)
	}
}
