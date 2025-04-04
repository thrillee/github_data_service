package db

import (
	"context"
	"database/sql"
)

// RepositoryQuerier defines the interface for repository database operations
type RepositoryQuerier interface {
	GetRepositoryByFullName(ctx context.Context, fullName string) (Repository, error)
	GetRepository(ctx context.Context, id int64) (Repository, error)
	ListRepositories(ctx context.Context) ([]Repository, error)
	CreateRepository(ctx context.Context, arg CreateRepositoryParams) (sql.Result, error)
	UpdateRepository(ctx context.Context, arg UpdateRepositoryParams) error
	UpdateRepositoryLastSyncedAt(ctx context.Context, arg UpdateRepositoryLastSyncedAtParams) error
	UpdateRepositorySyncFromDate(ctx context.Context, arg UpdateRepositorySyncFromDateParams) error
}

// CommitQuerier defines the interface for commit database operations
type CommitQuerier interface {
	GetCommit(ctx context.Context, sha string) (Commit, error)
	CreateCommit(ctx context.Context, arg CreateCommitParams) error
}

// DBQuerier combines all database query interfaces
type DBQuerier interface {
	RepositoryQuerier
	CommitQuerier
}
