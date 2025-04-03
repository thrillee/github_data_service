package api

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/thrillee/gds/internals/db"
	"github.com/thrillee/gds/templates"
)

// handleHome redirects to repositories page
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/repositories", http.StatusSeeOther)
}

// handleListRepositories displays all repositories
func (s *Server) handleListRepositories(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	repos, err := s.queries.ListRepositories(ctx)
	if err != nil {
		http.Error(w, "Failed to fetch repositories", http.StatusInternalServerError)
		log.Printf("Error fetching repositories: %v", err)
		return
	}

	templates.RepositoriesPage(repos).Render(ctx, w)
}

// RepositoryRequest represents the request to add a repository
type RepositoryRequest struct {
	FullName     string `json:"full_name"`
	SyncFromDate string `json:"sync_from_date"`
}

// handleAddRepository adds a new repository to track
func (s *Server) handleAddRepository(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	fullName := r.FormValue("full_name")
	syncFromDate := r.FormValue("sync_from_date")

	if fullName == "" {
		http.Error(w, "Repository name is required", http.StatusBadRequest)
		return
	}

	// Check if repository exists
	_, err := s.queries.GetRepositoryByFullName(ctx, fullName)
	if err == nil {
		http.Error(w, "Repository already exists", http.StatusBadRequest)
		return
	} else if err != sql.ErrNoRows {
		http.Error(w, "Failed to check repository existence", http.StatusInternalServerError)
		log.Printf("Error checking repository existence: %v", err)
		return
	}

	// Parse sync date
	var syncFromDatePtr sql.NullTime
	if syncFromDate != "" {
		date, err := time.Parse("2006-01-02", syncFromDate)
		if err != nil {
			http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		syncFromDatePtr = sql.NullTime{Time: date, Valid: true}
	}

	// Create repository
	repoID, err := s.service.AddRepository(ctx, fullName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to add repository: %v", err), http.StatusInternalServerError)
		log.Printf("Error adding repository: %v", err)
		return
	}

	s.service.ResetRepositorySync(ctx, repoID, syncFromDatePtr.Time)

	// HTMX response
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/repositories")
		w.WriteHeader(http.StatusOK)
	} else {
		http.Redirect(w, r, "/repositories", http.StatusSeeOther)
	}
}

// handleGetRepository displays a single repository and its details
func (s *Server) handleGetRepository(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid repository ID", http.StatusBadRequest)
		return
	}

	repo, err := s.queries.GetRepository(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Repository not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch repository", http.StatusInternalServerError)
			log.Printf("Error fetching repository: %v", err)
		}
		return
	}

	commitCount, err := s.queries.CountCommitsByRepoID(ctx, id)
	if err != nil {
		log.Printf("Error counting commits: %v", err)
		commitCount = 0
	}

	templates.RepositoryDetail(repo, commitCount).Render(ctx, w)
}

// handleResetRepositorySync updates the sync from date for a repository
func (s *Server) handleResetRepositorySync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	fmt.Println("IDS: ", idStr)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid repository ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	syncFromDate := r.FormValue("sync_from_date")
	if syncFromDate == "" {
		http.Error(w, "Sync from date is required", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", syncFromDate)
	if err != nil {
		http.Error(w, "Invalid date format. Use YYYY-MM-DD", http.StatusBadRequest)
		return
	}

	err = s.queries.UpdateRepositorySyncFromDate(ctx, db.UpdateRepositorySyncFromDateParams{
		ID:           id,
		SyncFromDate: sql.NullTime{Time: date, Valid: true},
	})
	if err != nil {
		http.Error(w, "Failed to update sync date", http.StatusInternalServerError)
		log.Printf("Error updating sync date: %v", err)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Trigger", "syncDateUpdated")
		w.WriteHeader(http.StatusOK)
		http.Redirect(w, r, fmt.Sprintf("/repositories/%d", id), http.StatusSeeOther)
		fmt.Fprintf(w, "Sync date updated to %s", date.Format("2006-01-02"))
	} else {
		http.Redirect(w, r, fmt.Sprintf("/repositories/%d", id), http.StatusSeeOther)
	}
}

// handleListCommits displays commits for a repository
func (s *Server) handleListCommits(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid repository ID", http.StatusBadRequest)
		return
	}

	// Pagination
	page := 1
	pageSize := 20
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if sizeStr := r.URL.Query().Get("size"); sizeStr != "" {
		if sz, err := strconv.Atoi(sizeStr); err == nil && sz > 0 && sz <= 100 {
			pageSize = sz
		}
	}

	offset := int64((page - 1) * pageSize)

	repo, err := s.queries.GetRepository(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Repository not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch repository", http.StatusInternalServerError)
			log.Printf("Error fetching repository: %v", err)
		}
		return
	}

	commits, err := s.queries.ListCommitsByRepoID(ctx, db.ListCommitsByRepoIDParams{
		RepoID: id,
		Limit:  int64(pageSize),
		Offset: offset,
	})
	if err != nil {
		http.Error(w, "Failed to fetch commits", http.StatusInternalServerError)
		log.Printf("Error fetching commits: %v", err)
		return
	}

	totalCommits, err := s.queries.CountCommitsByRepoID(ctx, id)
	if err != nil {
		log.Printf("Error counting commits: %v", err)
		totalCommits = int64(len(commits))
	}

	totalPages := (totalCommits + int64(pageSize) - 1) / int64(pageSize)

	data := struct {
		Repository   db.Repository
		Commits      []db.Commit
		CurrentPage  int
		TotalPages   int64
		TotalCommits int64
		PageSize     int
	}{
		Repository:   repo,
		Commits:      commits,
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalCommits: totalCommits,
		PageSize:     pageSize,
	}

	if r.Header.Get("HX-Request") == "true" && r.URL.Query().Get("partial") == "commits" {
		templates.CommitsList(data).Render(ctx, w)
	} else {
		templates.CommitsPage(data).Render(ctx, w)
	}
}

// handleTopAuthors displays top commit authors for a repository
func (s *Server) handleTopAuthors(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid repository ID", http.StatusBadRequest)
		return
	}

	limit := int64(10)
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.ParseInt(limitStr, 10, 64); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	repo, err := s.queries.GetRepository(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Repository not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to fetch repository", http.StatusInternalServerError)
			log.Printf("Error fetching repository: %v", err)
		}
		return
	}

	authors, err := s.queries.GetTopCommitAuthors(ctx, db.GetTopCommitAuthorsParams{
		RepoID: id,
		Limit:  limit,
	})
	if err != nil {
		http.Error(w, "Failed to fetch top authors", http.StatusInternalServerError)
		log.Printf("Error fetching top authors: %v", err)
		return
	}

	data := struct {
		Repository db.Repository
		Authors    []db.GetTopCommitAuthorsRow
		Limit      int64
	}{
		Repository: repo,
		Authors:    authors,
		Limit:      limit,
	}

	if r.Header.Get("HX-Request") == "true" && r.URL.Query().Get("partial") == "authors" {
		templates.AuthorsList(data).Render(ctx, w)
	} else {
		templates.AuthorsPage(data).Render(ctx, w)
	}
}

func (s *Server) handleRepositorySync(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid repository ID", http.StatusBadRequest)
		return
	}

	_, err = s.queries.GetRepository(r.Context(), id)
	if err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	ctx := r.Context()
	err = s.service.SyncRepository(ctx, id)
	if err != nil {
		http.Error(w, "Sycning Failed", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/repositories/%d", id), http.StatusSeeOther)
}

func (s *Server) handleResetSyncModal(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid repository ID", http.StatusBadRequest)
		return
	}

	repo, err := s.queries.GetRepository(r.Context(), id)
	if err != nil {
		http.Error(w, "Repository not found", http.StatusNotFound)
		return
	}

	component := templates.ResetSyncModal(repo.ID)
	component.Render(r.Context(), w)
}
