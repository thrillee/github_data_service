# GitHub Data Service (GDS)

The **GitHub Data Service (GDS)** is a Go-based application that fetches and stores data about GitHub repositories, providing insights into repository activity and contributors.

## Overview

GDS performs the following actions:

- **Configuration:** Loads settings from environment variables and potentially a configuration file.
- **Database:** Connects to a SQLite database to store and manage data.
- **GitHub Integration:** Uses the GitHub API to fetch repository information, such as top commit authors.
- **Background Synchronization:** Periodically syncs data from GitHub at a configurable interval.
- **API Server:** Exposes a REST API to access the collected data.

## Technologies Used

- **Golang**
- **Chi v5** (Router for handling HTTP requests)
- **HTMX** (For enhanced frontend interactivity)
- **Templ** (HTML templating engine)

## Prerequisites

- **Docker:** Ensure you have Docker installed on your system. Installation instructions can be found on the [official Docker website](https://docs.docker.com/engine/install/).
- **GitHub Personal Access Token:** You will need a GitHub Personal Access Token with the necessary permissions to access the repositories you want to track. Instructions to generate a token are available in the [GitHub documentation](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens).

## Getting Started

### Building the Docker Image

Navigate to the root directory of your project (where the `Dockerfile` is located) and build the Docker image:

```bash
docker build -t gds-app .
```

### Running the Docker Container

Run the GDS application using Docker:

```bash
docker run -p 8080:8080 -e GITHUB_TOKEN=<YOUR_GITHUB_TOKEN> -e SYNC_INTERVAL_MINUTES=60 --name gds gds-app
```

## REST API Endpoints

### Create a Repository

```bash
curl --location 'http://localhost:8080/api/repositories' \
--header 'Content-Type: application/json' \
--data '{
  "name": "go",
  "full_name": "google/go-github",
  "description": "The Go programming language",
  "url": "https://github.com/google/go-github",
  "language": "Go",
  "sync_from_date": "2009-11-10"
}'
```

#### Sample Response:

```json
{
  "id":1,
  "name":"go-github",
  "full_name":"google/go-github",
  "description":"Go library for accessing the GitHub v3 API",
  "url":"https://github.com/google/go-github",
  "language":"Go",
  "last_synced_at":"2025-04-03T12:25:47.202824+01:00",
  "sync_from_date":"2009-11-10T00:00:00Z"
}
```

### List All Repositories

```bash
curl --location --request GET 'http://localhost:8080/api/repositories' \
--header 'Content-Type: application/json'
```

#### Sample Response:

```json
[
  {
    "id":1,
    "name":"go-github",
    "full_name":"google/go-github",
    "description":"Go library for accessing the GitHub v3 API",
    "url":"https://github.com/google/go-github",
    "language":"Go",
    "last_synced_at":"2025-04-03T12:25:47.202824+01:00",
    "sync_from_date":"2009-11-10T00:00:00Z"
  }
]
```

### Get a Single Repository

```bash
curl --location --request GET 'http://localhost:8080/api/repositories/1' \
--header 'Content-Type: application/json'
```

#### Sample Response:

```json
{
  "id":1,
  "name":"go-github",
  "full_name":"google/go-github",
  "description":"Go library for accessing the GitHub v3 API",
  "url":"https://github.com/google/go-github",
  "language":"Go",
  "last_synced_at":"2025-04-03T12:25:47.202824+01:00",
  "sync_from_date":"2009-11-10T00:00:00Z"
}
```

### List Commits

```bash
curl --location --request GET 'http://localhost:8080/api/repositories/1/commits' \
--header 'Content-Type: application/json'
```

#### Sample Response:

```json
[
    {
        "sha": "3a3f51bc7c5daa18f7da89d1f806f200fcd4ce96",
        "repo_id": 1,
        "author_name": "Oleksandr Redko",
        "author_email": "oleksandr.red+github@gmail.com",
        "commit_date": "2025-04-02T12:52:10Z",
        "message": "chore: Remove redundant in Go 1.22 loop variables (#3537)",
        "html_url": "https://github.com/google/go-github/commit/3a3f51bc7c5daa18f7da89d1f806f200fcd4ce96"
    }
]
```

### List Authors

```bash
curl --location --request GET 'http://localhost:8080/api/repositories/1/authors' \
--header 'Content-Type: application/json'
```

#### Sample Response:

```json
[
    {
        "author_name": "dependabot[bot]",
        "author_email": "49699333+dependabot[bot]@users.noreply.github.com",
        "commit_count": 19
    }
]
```

### Reset Sync

```bash
curl --location 'http://localhost:8080/api/repositories/1/sync' \
--header 'Content-Type: application/json' \
--data '{
  "sync_from_date": "2025-01-01"
}'
```

You can also reset the sync by visiting:

[http://localhost:8080/repositories](http://localhost:8080/repositories)

## Unit Tests

The application includes unit tests for core functionalities. To run the tests, execute:

```bash
go test ./...
```

## Submission Guidelines

- Ensure the repository includes all necessary code, configuration files, and a complete README.
- Follow idiomatic Go practices, clean modular code, meaningful error handling, and performance considerations.
- Test the solution against the Chromium repository and seed the database with its data.

For further details, refer to the [GitHub REST API Documentation](https://docs.github.com/en/rest).
