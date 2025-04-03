package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server struct {
		Port string
	}
	Database struct {
		Driver         string
		DSN            string
		MigrationsPath string
	}
	GitHub struct {
		Token string
	}
	SyncInterval time.Duration
}

// Load configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Server config
	cfg.Server.Port = getEnv("PORT", "8080")

	// Database config
	cfg.Database.Driver = getEnv("DB_DRIVER", "sqlite3")
	cfg.Database.DSN = getEnv("DB_DSN", "./github_data.db")
	cfg.Database.MigrationsPath = getEnv("DB_MIGRATIONS_PATH", "./migrations")

	// GitHub config
	cfg.GitHub.Token = getEnv("GITHUB_TOKEN", "")

	// Sync interval
	syncIntervalMinutes, _ := strconv.Atoi(getEnv("SYNC_INTERVAL_MINUTES", "60"))
	cfg.SyncInterval = time.Duration(syncIntervalMinutes) * time.Minute

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
