package api

import (
	"context"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/hume-evi/web/internal/auth"
	"github.com/hume-evi/web/internal/config"
	"github.com/hume-evi/web/internal/db"
)

type Server struct {
	config *config.Config
	db     *db.DB
	auth   *auth.Auth
	router *mux.Router
}

func NewServer(cfg *config.Config, database *db.DB) *Server {
	s := &Server{
		config: cfg,
		db:     database,
		auth:   &auth.Auth{},
		router: mux.NewRouter(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// CORS middleware
	s.router.Use(corsMiddleware)

	// Public routes
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/auth/login", s.loginHandler).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/logout", s.logoutHandler).Methods("POST", "OPTIONS")

	// Protected routes
	protected := api.PathPrefix("").Subrouter()
	protected.Use(s.authMiddleware)
	protected.HandleFunc("/auth/me", s.meHandler).Methods("GET")

	// Conversations
	protected.HandleFunc("/conversations", s.createConversationHandler).Methods("POST")
	protected.HandleFunc("/conversations", s.listConversationsHandler).Methods("GET")
	protected.HandleFunc("/conversations/last-active", s.getLastActiveConversationHandler).Methods("GET")
	protected.HandleFunc("/conversations/{id}", s.getConversationHandler).Methods("GET")
	protected.HandleFunc("/conversations/{id}", s.updateConversationStatusHandler).Methods("PATCH")
	protected.HandleFunc("/conversations/{id}", s.deleteConversationHandler).Methods("DELETE")
	protected.HandleFunc("/conversations/{id}/messages", s.getMessagesHandler).Methods("GET")
	protected.HandleFunc("/conversations/{id}/messages", s.addMessageHandler).Methods("POST")
	
	// AI context analysis
	protected.HandleFunc("/analyze-conversation", s.analyzeConversationHandler).Methods("POST")
	
	// Knowledge graph endpoints
	protected.HandleFunc("/graph/extract", s.extractGraphHandler).Methods("POST")
	protected.HandleFunc("/graph/user-context", s.getUserGraphContextHandler).Methods("GET")
	
	// Voice management endpoints
	// GET is available to all authenticated users
	protected.HandleFunc("/voices", s.listVoicesHandler).Methods("GET")
	protected.HandleFunc("/voices/{id}", s.getVoiceHandler).Methods("GET")
	
	// Admin-only routes
	admin := protected.PathPrefix("/admin").Subrouter()
	admin.Use(s.requireAdminMiddleware)
	
	// User management (admin only)
	admin.HandleFunc("/users", s.listUsersHandler).Methods("GET")
	admin.HandleFunc("/users", s.createUserHandler).Methods("POST")
	admin.HandleFunc("/users/{id}", s.updateUserHandler).Methods("PATCH")
	admin.HandleFunc("/users/{id}", s.deleteUserHandler).Methods("DELETE")
	
	// Voice management (admin only)
	admin.HandleFunc("/voices", s.createVoiceHandler).Methods("POST")
	admin.HandleFunc("/voices/sync", s.syncAllVoicesHandler).Methods("POST")
	admin.HandleFunc("/voices/{id}", s.updateVoiceHandler).Methods("PATCH")
	admin.HandleFunc("/voices/{id}", s.deleteVoiceHandler).Methods("DELETE")
	admin.HandleFunc("/voices/{id}/sync", s.syncVoiceHandler).Methods("POST")
}


func (s *Server) Start(ctx context.Context) error {
	// Run migrations
	if err := s.db.RunMigrations(ctx); err != nil {
		log.Printf("Warning: Migration error: %v", err)
	}

	server := &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: s.router,
	}

	log.Printf("Server starting on port %s", s.config.Port)
	return server.ListenAndServe()
}

