-- +goose Up
CREATE TABLE repositories (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    full_name TEXT NOT NULL UNIQUE,
    description TEXT,
    url TEXT NOT NULL,
    language TEXT,
    forks_count INTEGER,
    stargazers_count INTEGER,
    open_issues_count INTEGER,
    watchers_count INTEGER,
    created_at DATETIME,
    updated_at DATETIME,
    last_synced_at DATETIME,
    sync_from_date DATETIME
);

-- +goose Down
DROP TABLE repositories;
