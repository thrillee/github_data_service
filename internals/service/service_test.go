package service_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/thrillee/gds/internals/db"
	"github.com/thrillee/gds/internals/github"
	"github.com/thrillee/gds/internals/service"
)

// MockDBQuerier implements the db.DBQuerier interface for testing
type MockDBQuerier struct {
	mock.Mock
}

func (m *MockDBQuerier) GetRepositoryByFullName(ctx context.Context, fullName string) (db.Repository, error) {
	args := m.Called(ctx, fullName)
	return args.Get(0).(db.Repository), args.Error(1)
}

func (m *MockDBQuerier) GetRepository(ctx context.Context, id int64) (db.Repository, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(db.Repository), args.Error(1)
}

func (m *MockDBQuerier) ListRepositories(ctx context.Context) ([]db.Repository, error) {
	args := m.Called(ctx)
	return args.Get(0).([]db.Repository), args.Error(1)
}

func (m *MockDBQuerier) CreateRepository(ctx context.Context, arg db.CreateRepositoryParams) (sql.Result, error) {
	args := m.Called(ctx, arg)
	return args.Get(0).(sql.Result), args.Error(1)
}

func (m *MockDBQuerier) UpdateRepository(ctx context.Context, arg db.UpdateRepositoryParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDBQuerier) UpdateRepositoryLastSyncedAt(ctx context.Context, arg db.UpdateRepositoryLastSyncedAtParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDBQuerier) UpdateRepositorySyncFromDate(ctx context.Context, arg db.UpdateRepositorySyncFromDateParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

func (m *MockDBQuerier) GetCommit(ctx context.Context, sha string) (db.Commit, error) {
	args := m.Called(ctx, sha)
	return args.Get(0).(db.Commit), args.Error(1)
}

func (m *MockDBQuerier) CreateCommit(ctx context.Context, arg db.CreateCommitParams) error {
	args := m.Called(ctx, arg)
	return args.Error(0)
}

// MockGithubClient implements the github.GitHubClient interface for testing
type MockGithubClient struct {
	mock.Mock
}

func (m *MockGithubClient) GetRepository(fullName string) (*github.Repository, error) {
	args := m.Called(fullName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*github.Repository), args.Error(1)
}

func (m *MockGithubClient) GetCommits(fullName string, page, perPage int, since time.Time) ([]github.Commit, bool, error) {
	args := m.Called(fullName, page, perPage, since)
	return args.Get(0).([]github.Commit), args.Bool(1), args.Error(2)
}

// MockSQLResult implements sql.Result for testing
type MockSQLResult struct {
	lastID int64
}

func (m MockSQLResult) LastInsertId() (int64, error) {
	return m.lastID, nil
}

func (m MockSQLResult) RowsAffected() (int64, error) {
	return 1, nil
}

func TestAddRepository_New(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockQueries := new(MockDBQuerier)
	mockGithubClient := new(MockGithubClient)

	repoService := service.NewRepositoryService(mockQueries, mockGithubClient, time.Hour)

	fullName := "owner/repo"
	now := time.Now()

	// GitHub repository data
	githubRepo := &github.Repository{
		Name:            "repo",
		FullName:        fullName,
		Description:     "Test repo",
		URL:             "https://github.com/owner/repo",
		Language:        "Go",
		ForksCount:      10,
		StargazersCount: 100,
		OpenIssuesCount: 5,
		WatchersCount:   20,
		CreatedAt:       now.Add(-24 * time.Hour),
		UpdatedAt:       now,
	}

	// Mock GetRepositoryByFullName to return Not Found
	mockQueries.On("GetRepositoryByFullName", ctx, fullName).Return(db.Repository{}, sql.ErrNoRows)

	// Mock GetRepository from GitHub
	mockGithubClient.On("GetRepository", fullName).Return(githubRepo, nil)

	// Expected create params
	expectedCreateParams := db.CreateRepositoryParams{
		Name:         "repo",
		FullName:     fullName,
		Description:  sql.NullString{String: "Test repo", Valid: true},
		Url:          "https://github.com/owner/repo",
		Language:     sql.NullString{String: "Go", Valid: true},
		UpdatedAt:    sql.NullTime{Time: now, Valid: true},
		CreatedAt:    sql.NullTime{Time: now.Add(-24 * time.Hour), Valid: true},
		LastSyncedAt: sql.NullTime{Valid: true},
		SyncFromDate: sql.NullTime{Valid: true},
	}

	// Use mock.MatchedBy to match the timestamp fields that we can't predict exactly
	mockQueries.On("CreateRepository", ctx, mock.MatchedBy(func(params db.CreateRepositoryParams) bool {
		// Only check the values we're certain about
		return params.Name == expectedCreateParams.Name &&
			params.FullName == expectedCreateParams.FullName &&
			params.Description.String == expectedCreateParams.Description.String &&
			params.Url == expectedCreateParams.Url &&
			params.Language.String == expectedCreateParams.Language.String &&
			params.LastSyncedAt.Valid &&
			params.SyncFromDate.Valid
	})).Return(MockSQLResult{lastID: 123}, nil)

	// Mock GetRepository for sync
	mockQueries.On("GetRepository", ctx, int64(123)).Return(db.Repository{
		ID:           123,
		FullName:     fullName,
		SyncFromDate: sql.NullTime{Time: now.AddDate(0, -1, 0), Valid: true},
	}, nil)

	// Mock GetRepository from GitHub again for sync
	mockGithubClient.On("GetRepository", fullName).Return(githubRepo, nil)

	// Mock UpdateRepository for sync
	mockQueries.On("UpdateRepository", ctx, mock.AnythingOfType("db.UpdateRepositoryParams")).Return(nil)

	// Mock GetCommits
	mockGithubClient.On("GetCommits", fullName, 1, 100, mock.AnythingOfType("time.Time")).Return([]github.Commit{}, false, nil)

	// Mock UpdateRepositoryLastSyncedAt
	mockQueries.On("UpdateRepositoryLastSyncedAt", ctx, mock.AnythingOfType("db.UpdateRepositoryLastSyncedAtParams")).Return(nil)

	// Execute
	id, err := repoService.AddRepository(ctx, fullName)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, int64(123), id)
	mockQueries.AssertExpectations(t)
	mockGithubClient.AssertExpectations(t)
}

// Add more tests following the same pattern from the previous code...
