package github

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GitHubClient defines the interface for GitHub API operations
type GitHubClient interface {
	GetRepository(fullName string) (*Repository, error)
	GetCommits(fullName string, page, perPage int, since time.Time) ([]Commit, bool, error)
}

// Client is a GitHub API client
type Client struct {
	token      string
	httpClient *http.Client
	// Rate limiting configuration
	maxRetries     int
	initialBackoff time.Duration
	maxBackoff     time.Duration
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

// RateLimitError represents a GitHub API rate limit error
type RateLimitError struct {
	Message          string
	ResetTimeSeconds int64
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("%s (rate limit resets at %s)",
		e.Message,
		time.Unix(e.ResetTimeSeconds, 0).Format(time.RFC3339))
}

// ClientOption defines a function type for client configuration
type ClientOption func(*Client)

// WithMaxRetries sets the maximum number of retries for the client
func WithMaxRetries(retries int) ClientOption {
	return func(c *Client) {
		c.maxRetries = retries
	}
}

// WithInitialBackoff sets the initial backoff duration for the client
func WithInitialBackoff(duration time.Duration) ClientOption {
	return func(c *Client) {
		c.initialBackoff = duration
	}
}

// WithMaxBackoff sets the maximum backoff duration for the client
func WithMaxBackoff(duration time.Duration) ClientOption {
	return func(c *Client) {
		c.maxBackoff = duration
	}
}

// NewClient creates a new GitHub API client with the specified options
func NewClient(token string, options ...ClientOption) *Client {
	client := &Client{
		token: token,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		// Default rate limiting configuration
		maxRetries:     3,
		initialBackoff: 1 * time.Second,
		maxBackoff:     60 * time.Second,
	}

	// Apply custom options
	for _, option := range options {
		option(client)
	}

	return client
}

// GetRepository fetches repository information
func (c *Client) GetRepository(fullName string) (*Repository, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s", fullName)
	req, err := c.newRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := c.doRequestWithRetry(req, &repo); err != nil {
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
	if err := c.doRequestWithRetry(req, &commits); err != nil {
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

// doRequestWithRetry performs the HTTP request with exponential backoff retry logic
func (c *Client) doRequestWithRetry(req *http.Request, v interface{}) error {
	var lastErr error
	backoff := c.initialBackoff

	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		// Create a new request for each attempt since the request body might be consumed
		newReq := req.Clone(req.Context())

		// Perform the request
		err := c.doRequest(newReq, v)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if it's a rate limit error
		if rateLimitErr, ok := err.(*RateLimitError); ok {
			// Calculate time until reset
			resetTime := time.Unix(rateLimitErr.ResetTimeSeconds, 0)
			waitTime := time.Until(resetTime)

			// If reset time is soon, wait for it; otherwise use backoff
			if waitTime > 0 && waitTime < c.maxBackoff {
				fmt.Printf("Rate limited. Waiting until reset: %s\n", resetTime.Format(time.RFC3339))
				time.Sleep(waitTime + 100*time.Millisecond) // Add a small buffer
				continue
			}
		}

		// For other errors or if rate limit wait is too long, use exponential backoff
		if attempt < c.maxRetries {
			// Calculate backoff with jitter (±20%)
			jitter := 0.8 + 0.4*rand.Float64()
			waitTime := time.Duration(float64(backoff) * jitter)

			fmt.Printf("Request failed (attempt %d/%d). Retrying in %v: %v\n",
				attempt+1, c.maxRetries, waitTime, err)

			time.Sleep(waitTime)

			// Increase backoff for next attempt
			backoff = time.Duration(float64(backoff) * 2)
			if backoff > c.maxBackoff {
				backoff = c.maxBackoff
			}
		}
	}

	return fmt.Errorf("request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}

// doRequest performs the HTTP request and unmarshals the response
func (c *Client) doRequest(req *http.Request, v interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Handle rate limiting
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		resetHeader := resp.Header.Get("X-RateLimit-Reset")
		remainingHeader := resp.Header.Get("X-RateLimit-Remaining")

		// Check if it's actually a rate limit issue
		remaining, _ := strconv.Atoi(remainingHeader)
		if remaining == 0 && resetHeader != "" {
			resetTime, err := strconv.ParseInt(resetHeader, 10, 64)
			if err == nil {
				return &RateLimitError{
					Message:          "GitHub API rate limit exceeded",
					ResetTimeSeconds: resetTime,
				}
			}
		}
	}

	// Handle other error status codes
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	// Parse response body
	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("error decoding response: %w", err)
	}

	return nil
}
