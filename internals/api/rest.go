package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/thrillee/gds/internals/db"
)

// RepositoryResponse represents the API response for repository
type RepositoryResponse struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	FullName        string     `json:"full_name"`
	Description     *string    `json:"description,omitempty"`
	Url             string     `json:"url"`
	Language        *string    `json:"language,omitempty"`
	ForksCount      *int64     `json:"forks_count,omitempty"`
	StargazersCount *int64     `json:"stargazers_count,omitempty"`
	OpenIssuesCount *int64     `json:"open_issues_count,omitempty"`
	WatchersCount   *int64     `json:"watchers_count,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	UpdatedAt       *time.Time `json:"updated_at,omitempty"`
	LastSyncedAt    *time.Time `json:"last_synced_at,omitempty"`
	SyncFromDate    *time.Time `json:"sync_from_date,omitempty"`
}

// CommitResponse represents the API response for commit
type CommitResponse struct {
	Sha         string    `json:"sha"`
	RepoID      int64     `json:"repo_id"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
	CommitDate  time.Time `json:"commit_date"`
	Message     string    `json:"message"`
	HtmlUrl     string    `json:"html_url"`
}

// AuthorResponse represents the API response for top authors
type AuthorResponse struct {
	AuthorName  string `json:"author_name"`
	AuthorEmail string `json:"author_email"`
	CommitCount int64  `json:"commit_count"`
}

// handleListRepositoriesAPI returns a list of all repositories
func (s *Server) handleListRepositoriesAPI(w http.ResponseWriter, r *http.Request) {
	repos, err := s.queries.ListRepositories(r.Context())
	if err != nil {
		http.Error(w, "Failed to fetch repositories", http.StatusInternalServerError)
		return
	}

	response := make([]RepositoryResponse, 0, len(repos))
	for _, repo := range repos {
		response = append(response, convertRepositoryToResponse(repo))
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetRepositoryAPI returns a single repository by ID
func (s *Server) handleGetRepositoryAPI(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid repository ID", http.StatusBadRequest)
		return
	}

	repo, err := s.queries.GetRepository(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Repository not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch repository", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(convertRepositoryToResponse(repo))
}

// handleAddRepositoryAPI adds a new repository
func (s *Server) handleAddRepositoryAPI(w http.ResponseWriter, r *http.Request) {
	var req RepositoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FullName == "" {
		http.Error(w, "Repository full name is required", http.StatusBadRequest)
		return
	}

	// Check if repository already exists
	_, err := s.queries.GetRepositoryByFullName(r.Context(), req.FullName)
	if err == nil {
		http.Error(w, "Repository already exists", http.StatusConflict)
		return
	} else if err != sql.ErrNoRows {
		http.Error(w, "Failed to check repository existence", http.StatusInternalServerError)
		return
	}

	// Parse sync from date
	var syncFromDate sql.NullTime
	if req.SyncFromDate != "" {
		date, err := time.Parse("2006-01-02", req.SyncFromDate)
		if err != nil {
			http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		syncFromDate = sql.NullTime{Time: date, Valid: true}
	}

	// Add repository
	repoID, err := s.service.AddRepository(r.Context(), req.FullName)
	if err != nil {
		http.Error(w, "Failed to add repository", http.StatusInternalServerError)
		return
	}

	// Reset sync if date was provided
	if syncFromDate.Valid {
		s.service.ResetRepositorySync(r.Context(), repoID, syncFromDate.Time)
	}

	// Return the created repository
	repo, err := s.queries.GetRepository(r.Context(), repoID)
	if err != nil {
		http.Error(w, "Failed to fetch created repository", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(convertRepositoryToResponse(repo))
}

// handleListCommitsAPI returns commits for a repository
func (s *Server) handleListCommitsAPI(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid repository ID", http.StatusBadRequest)
		return
	}

	// Get pagination parameters
	page, size := getPaginationParams(r)

	commits, err := s.queries.ListCommitsByRepoID(r.Context(), db.ListCommitsByRepoIDParams{
		RepoID: id,
		Limit:  int64(size),
		Offset: int64((page - 1) * size),
	})
	if err != nil {
		http.Error(w, "Failed to fetch commits", http.StatusInternalServerError)
		return
	}

	response := make([]CommitResponse, 0, len(commits))
	for _, commit := range commits {
		response = append(response, CommitResponse{
			Sha:         commit.Sha,
			RepoID:      commit.RepoID,
			AuthorName:  commit.AuthorName,
			AuthorEmail: commit.AuthorEmail,
			CommitDate:  commit.CommitDate,
			Message:     commit.Message,
			HtmlUrl:     commit.HtmlUrl,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleTopAuthorsAPI returns top authors for a repository
func (s *Server) handleTopAuthorsAPI(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid repository ID", http.StatusBadRequest)
		return
	}

	// Get limit parameter (default to 10)
	limit := int64(10)
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.ParseInt(limitStr, 10, 64); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	authors, err := s.queries.GetTopCommitAuthors(r.Context(), db.GetTopCommitAuthorsParams{
		RepoID: id,
		Limit:  limit,
	})
	if err != nil {
		http.Error(w, "Failed to fetch top authors", http.StatusInternalServerError)
		return
	}

	response := make([]AuthorResponse, 0, len(authors))
	for _, author := range authors {
		response = append(response, AuthorResponse{
			AuthorName:  author.AuthorName,
			AuthorEmail: author.AuthorEmail,
			CommitCount: author.CommitCount,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleResetRepositorySync updates the sync from date for a repository
func (s *Server) handleResetRepositorySyncAPI(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid repository ID", http.StatusBadRequest)
		return
	}

	var req struct {
		SyncFromDate string `json:"sync_from_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SyncFromDate == "" {
		http.Error(w, "Sync from date is required", http.StatusBadRequest)
		return
	}

	// Parse the date
	date, err := time.Parse("2006-01-02", req.SyncFromDate)
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	// Update the repository's sync from date
	err = s.queries.UpdateRepositorySyncFromDate(r.Context(), db.UpdateRepositorySyncFromDateParams{
		ID:           id,
		SyncFromDate: sql.NullTime{Time: date, Valid: true},
	})
	if err != nil {
		http.Error(w, "Failed to update sync date", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// convertRepositoryToResponse converts a db.Repository to a RepositoryResponse
func convertRepositoryToResponse(repo db.Repository) RepositoryResponse {
	resp := RepositoryResponse{
		ID:       repo.ID,
		Name:     repo.Name,
		FullName: repo.FullName,
		Url:      repo.Url,
	}

	if repo.Description.Valid {
		resp.Description = &repo.Description.String
	}
	if repo.Language.Valid {
		resp.Language = &repo.Language.String
	}
	if repo.ForksCount.Valid {
		resp.ForksCount = &repo.ForksCount.Int64
	}
	if repo.StargazersCount.Valid {
		resp.StargazersCount = &repo.StargazersCount.Int64
	}
	if repo.OpenIssuesCount.Valid {
		resp.OpenIssuesCount = &repo.OpenIssuesCount.Int64
	}
	if repo.WatchersCount.Valid {
		resp.WatchersCount = &repo.WatchersCount.Int64
	}
	if repo.CreatedAt.Valid {
		resp.CreatedAt = &repo.CreatedAt.Time
	}
	if repo.UpdatedAt.Valid {
		resp.UpdatedAt = &repo.UpdatedAt.Time
	}
	if repo.LastSyncedAt.Valid {
		resp.LastSyncedAt = &repo.LastSyncedAt.Time
	}
	if repo.SyncFromDate.Valid {
		resp.SyncFromDate = &repo.SyncFromDate.Time
	}

	return resp
}

// getPaginationParams extracts pagination parameters from the request
func getPaginationParams(r *http.Request) (page int, size int) {
	page = 1
	size = 20

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if sizeStr := r.URL.Query().Get("size"); sizeStr != "" {
		if sz, err := strconv.Atoi(sizeStr); err == nil && sz > 0 && sz <= 100 {
			size = sz
		}
	}

	return page, size
}
