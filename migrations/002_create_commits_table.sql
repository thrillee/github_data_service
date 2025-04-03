-- +goose Up
CREATE TABLE commits (
    sha TEXT PRIMARY KEY,
    repo_id INTEGER NOT NULL,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    commit_date DATETIME NOT NULL,
    message TEXT NOT NULL,
    html_url TEXT NOT NULL,
    FOREIGN KEY (repo_id) REFERENCES repositories(id)
);

CREATE INDEX idx_commits_repo_id ON commits(repo_id);
CREATE INDEX idx_commits_author_name ON commits(author_name);
CREATE INDEX idx_commits_commit_date ON commits(commit_date);

-- +goose Down
DROP TABLE commits;
