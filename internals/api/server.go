package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/thrillee/gds/internals/db"
	"github.com/thrillee/gds/internals/service"
)

// Server is an HTTP server
type Server struct {
	queries *db.Queries
	service *service.RepositoryService
	// tmpl    *template.Template
}

// NewServer creates a new HTTP server
func NewServer(queries *db.Queries, service *service.RepositoryService) *Server {
	// Load templates
	// tmpl, err := ParseTemplates("templates")
	// if err != nil {
	// 	panic(err)
	// }

	return &Server{
		queries: queries,
		service: service,
		// tmpl:    tmpl,
	}
}

// Router returns a configured HTTP router
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	// Static files
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Routes
	r.Get("/", s.handleHome)
	r.Get("/repositories", s.handleListRepositories)
	r.Post("/repositories", s.handleAddRepository)
	r.Get("/repositories/{id}", s.handleGetRepository)
	r.Post("/repositories/{id}/reset", s.handleResetRepositorySync)
	r.Post("/repositories/{id}/sync", s.handleRepositorySync)
	r.Get("/repositories/{id}/commits", s.handleListCommits)
	r.Get("/repositories/{id}/authors", s.handleTopAuthors)
	r.Get("/repositories/{id}/reset-sync-modal", s.handleResetSyncModal)

	// REST API Routes
	r.Get("/api/repositories", s.handleListRepositoriesAPI)
	r.Post("/api/repositories", s.handleAddRepositoryAPI)
	r.Get("/api/repositories/{id}", s.handleGetRepositoryAPI)
	r.Get("/api/repositories/{id}/commits", s.handleListCommitsAPI)
	r.Get("/api/repositories/{id}/authors", s.handleTopAuthorsAPI)
	r.Post("/api/repositories/{id}/sync", s.handleResetRepositorySyncAPI)

	return r
}
