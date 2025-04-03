// internal/github/client.go
package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client is a GitHub API client
type Client struct {
	token      string
	httpClient *http.Client
}

// Repository represents a GitHub repository
type Repository struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	Description     string    `json:"description"`
	URL             string    `json:"html_url"`
	Language        string    `json:"language"`
	ForksCount      int       `json:"forks_count"`
	StargazersCount int       `json:"stargazers_count"`
	OpenIssuesCount int       `json:"open_issues_count"`
	WatchersCount   int       `json:"watchers_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Commit represents a GitHub commit
type Commit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
	HTMLURL string `json:"html_url"`
}

// NewClient creates a new GitHub API client
func NewClient(token string) *Client {
	return &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetRepository fetches repository information
func (c *Client) GetRepository(fullName string) (*Repository, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", fullName)

	req, err := c.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := c.doRequest(req, &repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

// GetCommits fetches commits for a repository
func (c *Client) GetCommits(fullName string, page, perPage int, since time.Time) ([]Commit, bool, error) {
	sinceStr := since.Format(time.RFC3339)
	url := fmt.Sprintf("https://api.github.com/repos/%s/commits?per_page=%d&page=%d&since=%s",
		fullName, perPage, page, sinceStr)

	req, err := c.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}

	var commits []Commit
	if err := c.doRequest(req, &commits); err != nil {
		return nil, false, err
	}

	// Check if there are more pages
	hasMore := len(commits) == perPage

	return commits, hasMore, nil
}

// newRequest creates a new HTTP request with the necessary headers
func (c *Client) newRequest(method, url string, body *strings.Reader) (*http.Request, error) {
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
	if c.token != "" {
		req.Header.Add("Authorization", "token "+c.token)
	}

	req.Header.Add("Accept", "application/vnd.github.v3+json")
	req.Header.Add("User-Agent", "GitHub-Data-Service")

	return req, nil
}

// doRequest performs the HTTP request and unmarshals the response
func (c *Client) doRequest(req *http.Request, v interface{}) error {
	resp, err := c.httpClient.Do(req)
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
