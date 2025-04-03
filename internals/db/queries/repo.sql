-- name: GetRepository :one
SELECT * FROM repositories WHERE id = ? LIMIT 1;

-- name: GetRepositoryByFullName :one
SELECT * FROM repositories WHERE full_name = ? LIMIT 1;

-- name: ListRepositories :many
SELECT * FROM repositories ORDER BY last_synced_at DESC;

-- name: CreateRepository :execresult
INSERT INTO repositories (
    name, full_name, description, url, language, 
    forks_count, stargazers_count, open_issues_count, 
    watchers_count, created_at, updated_at, last_synced_at, sync_from_date
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
);

-- name: UpdateRepository :exec
UPDATE repositories SET
    name = ?,
    description = ?,
    url = ?,
    language = ?,
    forks_count = ?,
    stargazers_count = ?,
    open_issues_count = ?,
    watchers_count = ?,
    updated_at = ?
WHERE full_name = ?;

-- name: UpdateRepositorySyncFromDate :exec
UPDATE repositories SET sync_from_date = ? WHERE id = ?;

-- name: UpdateRepositoryLastSyncedAt :exec
UPDATE repositories SET last_synced_at = ? WHERE id = ?;

-- name: DeleteRepository :exec
DELETE FROM repositories WHERE id = ?;

-- name: GetCommit :one
SELECT * FROM commits WHERE sha = ? LIMIT 1;

-- name: ListCommitsByRepoID :many
SELECT * FROM commits 
WHERE repo_id = ? 
ORDER BY commit_date DESC
LIMIT ? OFFSET ?;

-- name: CountCommitsByRepoID :one
SELECT COUNT(*) FROM commits WHERE repo_id = ?;

-- name: CreateCommit :exec
INSERT INTO commits (
    sha, repo_id, author_name, author_email, 
    commit_date, message, html_url
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
);

-- name: GetTopCommitAuthors :many
SELECT 
    author_name, 
    author_email, 
    COUNT(*) as commit_count
FROM commits
WHERE repo_id = ?
GROUP BY author_name, author_email
ORDER BY commit_count DESC
LIMIT ?;
