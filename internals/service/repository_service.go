package service

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/thrillee/gds/internals/db"
	"github.com/thrillee/gds/internals/github"
)

// RepositoryService handles business logic for repositories
type RepositoryService struct {
	queries      db.DBQuerier
	githubClient github.GitHubClient
	syncInterval time.Duration
}

// NewRepositoryService creates a new repository service
func NewRepositoryService(queries db.DBQuerier, githubClient github.GitHubClient, syncInterval time.Duration) *RepositoryService {
	return &RepositoryService{
		queries:      queries,
		githubClient: githubClient,
		syncInterval: syncInterval,
	}
}

// AddRepository adds a new repository to the database
func (s *RepositoryService) AddRepository(ctx context.Context, fullName string) (int64, error) {
	// Fetch repository information from GitHub
	repo, err := s.githubClient.GetRepository(fullName)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch repository info: %w", err)
	}

	// Check if repository already exists
	existingRepo, err := s.queries.GetRepositoryByFullName(ctx, fullName)
	if err == nil {
		// Repository exists, update it
		updateParams := db.UpdateRepositoryParams{
			Name:        repo.Name,
			Description: sql.NullString{String: repo.Description, Valid: repo.Description != ""},
			Url:         repo.URL,
			Language:    sql.NullString{String: repo.Language, Valid: repo.Language != ""},
			ForksCount: sql.NullInt64{
				Int64: int64(repo.ForksCount),
			},
			StargazersCount: sql.NullInt64{
				Int64: int64(repo.StargazersCount),
			},
			OpenIssuesCount: sql.NullInt64{
				Int64: int64(repo.OpenIssuesCount),
			},
			WatchersCount: sql.NullInt64{
				Int64: int64(repo.WatchersCount),
			},
			UpdatedAt: sql.NullTime{
				Time: repo.UpdatedAt,
			},
			FullName: fullName,
		}

		err = s.queries.UpdateRepository(ctx, updateParams)
		if err != nil {
			return 0, fmt.Errorf("failed to update repository: %w", err)
		}

		return int64(existingRepo.ID), nil
	} else if err != sql.ErrNoRows {
		return 0, fmt.Errorf("error checking repository existence: %w", err)
	}

	// Repository doesn't exist, create it
	createParams := db.CreateRepositoryParams{
		Name:        repo.Name,
		FullName:    repo.FullName,
		Description: sql.NullString{String: repo.Description, Valid: repo.Description != ""},
		Url:         repo.URL,
		Language:    sql.NullString{String: repo.Language, Valid: repo.Language != ""},
		ForksCount: sql.NullInt64{
			Int64: int64(repo.ForksCount),
		},
		StargazersCount: sql.NullInt64{
			Int64: int64(repo.StargazersCount),
		},
		OpenIssuesCount: sql.NullInt64{
			Int64: int64(repo.OpenIssuesCount),
		},
		WatchersCount: sql.NullInt64{
			Int64: int64(repo.WatchersCount),
		},
		UpdatedAt: sql.NullTime{
			Time: repo.UpdatedAt,
		},
		CreatedAt: sql.NullTime{
			Time: repo.CreatedAt,
		},
		LastSyncedAt: sql.NullTime{Time: time.Now(), Valid: true},
		SyncFromDate: sql.NullTime{Time: time.Now().AddDate(0, -1, 0), Valid: true},
	}

	result, err := s.queries.CreateRepository(ctx, createParams)
	if err != nil {
		return 0, fmt.Errorf("failed to create repository: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}

	err = s.SyncRepository(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("Sync Failed: %w", err)
	}

	return id, nil
}

// SyncRepository synchronizes repository data with GitHub
func (s *RepositoryService) SyncRepository(ctx context.Context, repoID int64) error {
	// Get repository from database
	repo, err := s.queries.GetRepository(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	// Update repository information
	githubRepo, err := s.githubClient.GetRepository(repo.FullName)
	if err != nil {
		return fmt.Errorf("failed to fetch repository info: %w", err)
	}

	// Update repository in database
	updateParams := db.UpdateRepositoryParams{
		Name:        githubRepo.Name,
		Description: sql.NullString{String: githubRepo.Description, Valid: githubRepo.Description != ""},
		Url:         githubRepo.URL,
		Language:    sql.NullString{String: githubRepo.Language, Valid: githubRepo.Language != ""},
		ForksCount: sql.NullInt64{
			Int64: int64(githubRepo.ForksCount),
		},
		StargazersCount: sql.NullInt64{
			Int64: int64(githubRepo.StargazersCount),
		},
		OpenIssuesCount: sql.NullInt64{
			Int64: int64(githubRepo.OpenIssuesCount),
		},
		WatchersCount: sql.NullInt64{
			Int64: int64(githubRepo.WatchersCount),
		},
		UpdatedAt: sql.NullTime{
			Time: githubRepo.UpdatedAt,
		},
		FullName: repo.FullName,
	}

	err = s.queries.UpdateRepository(ctx, updateParams)
	if err != nil {
		return fmt.Errorf("failed to update repository: %w", err)
	}

	// Get sync from date
	syncFromDate := time.Now().AddDate(0, -1, 0) // Default to 1 month ago
	if repo.SyncFromDate.Valid {
		syncFromDate = repo.SyncFromDate.Time
	}

	// Fetch commits since syncFromDate
	err = s.syncCommits(ctx, repoID, repo.FullName, syncFromDate)
	if err != nil {
		return fmt.Errorf("failed to sync commits: %w", err)
	}

	// Update last synced timestamp
	err = s.queries.UpdateRepositoryLastSyncedAt(ctx, db.UpdateRepositoryLastSyncedAtParams{
		LastSyncedAt: sql.NullTime{Time: time.Now(), Valid: true},
		ID:           repoID,
	})
	if err != nil {
		return fmt.Errorf("failed to update last_synced_at: %w", err)
	}

	return nil
}

// syncCommits fetches commits from GitHub and saves them to the database
func (s *RepositoryService) syncCommits(ctx context.Context, repoID int64, fullName string, since time.Time) error {
	page := 1
	perPage := 100

	for {
		commits, hasMore, err := s.githubClient.GetCommits(fullName, page, perPage, since)
		if err != nil {
			return fmt.Errorf("failed to fetch commits: %w", err)
		}

		// Save each commit to the database
		for _, commit := range commits {
			// Check if commit already exists
			_, err := s.queries.GetCommit(ctx, commit.SHA)
			if err == nil {
				// Commit already exists, skip
				continue
			} else if err != sql.ErrNoRows {
				return fmt.Errorf("error checking commit existence: %w", err)
			}

			// Save commit
			createParams := db.CreateCommitParams{
				Sha:         commit.SHA,
				RepoID:      repoID,
				AuthorName:  commit.Commit.Author.Name,
				AuthorEmail: commit.Commit.Author.Email,
				CommitDate:  commit.Commit.Author.Date,
				Message:     commit.Commit.Message,
				HtmlUrl:     commit.HTMLURL,
			}

			err = s.queries.CreateCommit(ctx, createParams)
			if err != nil {
				log.Printf("Error saving commit %s: %v", commit.SHA, err)
				// Continue with other commits
			}
		}

		if !hasMore || len(commits) < perPage {
			break
		}

		page++
		// Add a small delay to avoid rate limiting
		select {
		case <-time.After(500 * time.Millisecond):
			// continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// ResetRepositorySync resets repository sync to start from a specific date
func (s *RepositoryService) ResetRepositorySync(ctx context.Context, repoID int64, startDate time.Time) error {
	return s.queries.UpdateRepositorySyncFromDate(ctx, db.UpdateRepositorySyncFromDateParams{
		SyncFromDate: sql.NullTime{Time: startDate, Valid: true},
		ID:           repoID,
	})
}

// StartSyncJob starts a background job to sync repositories periodically
func (s *RepositoryService) StartSyncJob(ctx context.Context) {
	fmt.Println("Sync Started")
	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	// Do an initial sync
	s.syncAllRepositories(ctx)

	for {
		select {
		case <-ticker.C:
			s.syncAllRepositories(ctx)
		case <-ctx.Done():
			log.Println("Repository sync job stopped")
			return
		}
	}
}

// syncAllRepositories syncs all repositories in the database
func (s *RepositoryService) syncAllRepositories(ctx context.Context) {
	log.Println("Starting sync of all repositories")

	repos, err := s.queries.ListRepositories(ctx)
	if err != nil {
		log.Printf("Error querying repositories: %v", err)
		return
	}

	for _, repo := range repos {
		err := s.SyncRepository(ctx, repo.ID)
		if err != nil {
			log.Printf("Error syncing repository %s: %v", repo.FullName, err)
			continue
		}
	}

	log.Println("Finished sync of all repositories")
}
