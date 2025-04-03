# GitHub Data Service (GDS)

This application fetches and stores data about GitHub repositories, providing insights into repository activity and contributors.

## Overview

The GitHub Data Service (GDS) is a Go-based application that performs the following actions:

* **Configuration:** Loads settings from environment variables and potentially a configuration file.
* **Database:** Connects to a SQLite database to store and manage data.
* **GitHub Integration:** Uses the GitHub API to fetch repository information, such as top commit authors.
* **Background Synchronization:** Periodically syncs data from GitHub at a configurable interval.
* **API Server:** Exposes an HTTP API to access the collected data.

## Prerequisites

* **Docker:** Ensure you have Docker installed on your system. You can find installation instructions for your operating system on the [official Docker website](https://docs.docker.com/engine/install/).
* **GitHub Personal Access Token:** You will need a GitHub Personal Access Token with the necessary permissions to access the repositories you want to track. You can generate a token by following the instructions in the [GitHub documentation](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens).

## Getting Started

### Building the Docker Image

1.  Navigate to the root directory of your project (the same directory where your `Dockerfile` is located).
2.  Build the Docker image using the following command:

    ```bash
    docker build -t gds-app .
    ```

    This command will build a Docker image named `gds-app` using the instructions in your `Dockerfile`.

### Running the Docker Container

You can run the GDS application using Docker with the following command:

```bash
docker run -p 8080:8080 -e GITHUB_TOKEN=<YOUR_GITHUB_TOKEN> -e SYNC_INTERVAL_MINUTES=60 --name gds gds-app
