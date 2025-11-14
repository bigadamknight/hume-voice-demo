package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hume-evi/web/internal/api"
	"github.com/hume-evi/web/internal/auth"
	"github.com/hume-evi/web/internal/config"
	"github.com/hume-evi/web/internal/db"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Validate required config
	if cfg.HumeAPIKey == "" {
		log.Fatal("HUME_API_KEY is required")
	}
	if cfg.HumeConfigID == "" {
		log.Fatal("HUME_CONFIG_ID is required")
	}
	if cfg.AdminUsername == "" {
		log.Fatal("ADMIN_USERNAME is required")
	}
	if cfg.AdminPassword == "" {
		log.Fatal("ADMIN_PASSWORD is required")
	}

	// Connect to database
	database, err := db.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Run migrations first
	ctx := context.Background()
	if err := database.RunMigrations(ctx); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Ensure admin user exists in database
	authService := &auth.Auth{}
	adminUser, err := database.GetUserByUsername(context.Background(), cfg.AdminUsername)
	if err != nil {
		// Admin user doesn't exist, create it
		log.Printf("Creating admin user %s...", cfg.AdminUsername)
		passwordHash, err := authService.HashPassword(cfg.AdminPassword)
		if err != nil {
			log.Fatal("Failed to hash admin password:", err)
		}
		_, err = database.CreateUser(context.Background(), cfg.AdminUsername, passwordHash, true)
		if err != nil {
			log.Fatal("Failed to create admin user:", err)
		}
		log.Printf("Admin user created successfully")
	} else {
		// Admin user exists, update password if it changed
		log.Printf("Admin user already exists, ensuring password is up to date")
		passwordHash, err := authService.HashPassword(cfg.AdminPassword)
		if err != nil {
			log.Fatal("Failed to hash admin password:", err)
		}
		// Update password and ensure is_admin is true
		isAdmin := true
		err = database.UpdateUser(context.Background(), adminUser.ID, &passwordHash, &isAdmin)
		if err != nil {
			log.Printf("Warning: Failed to update admin user: %v", err)
		} else {
			log.Printf("Admin user password updated")
		}
	}

	// Create server
	server := api.NewServer(cfg, database)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		cancel()
	}()

	// Start server
	if err := server.Start(ctx); err != nil {
		log.Fatal("Server error:", err)
	}
}

