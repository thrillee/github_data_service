package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	"github.com/thrillee/gds/internals/api"
	"github.com/thrillee/gds/internals/config"
	"github.com/thrillee/gds/internals/db"
	"github.com/thrillee/gds/internals/github"
	"github.com/thrillee/gds/internals/service"
)

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	dbConn, err := sql.Open(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	// Run database migrations
	if err := db.RunMigrations(cfg.Database.MigrationsPath, cfg.Database.DSN); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create SQLc queries
	queries := db.New(dbConn)

	// Initialize services
	githubClient := github.NewClient(cfg.GitHub.Token)
	repoService := service.NewRepositoryService(queries, githubClient, cfg.SyncInterval)

	// Start background sync job
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go repoService.StartSyncJob(ctx)

	// Initialize HTTP server
	server := api.NewServer(queries, repoService)

	// Start HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: server.Router(),
	}

	// Run server in a goroutine so it doesn't block
	go func() {
		log.Printf("Server starting on port %s", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Create a deadline to wait for
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Doesn't block if no connections, but will wait until
	// the timeout deadline
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
