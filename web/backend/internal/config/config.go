package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	JWTSecret     string
	HumeAPIKey    string
	HumeConfigID  string
	Port          string
	AdminUsername string
	AdminPassword string
	// Optional: Memgraph configuration
	MemgraphURI      string
	MemgraphUsername string
	MemgraphPassword string
	// Optional: CORS origin for frontend
	AppEnv     string
	CORSOrigin string
}

func Load() (*Config, error) {
	// Load .env file if it exists (optional for production)
	_ = godotenv.Load()

	cfg := &Config{
		DatabaseURL:      getEnv("DATABASE_URL", "postgresql://hume:hume@localhost:5432/hume_evi?sslmode=disable"),
		JWTSecret:        getEnv("JWT_SECRET", "change-me-in-production"),
		HumeAPIKey:       getEnv("HUME_API_KEY", ""),
		HumeConfigID:     getEnv("HUME_CONFIG_ID", ""),
		Port:             getEnv("PORT", "8080"),
		AdminUsername:    getEnv("ADMIN_USERNAME", ""),
		AdminPassword:    getEnv("ADMIN_PASSWORD", ""),
		MemgraphURI:      getEnv("MEMGRAPH_URI", "bolt://memgraph:7687"),
		MemgraphUsername: getEnv("MEMGRAPH_USERNAME", ""),
		MemgraphPassword: getEnv("MEMGRAPH_PASSWORD", ""),
		AppEnv:           getEnv("APP_ENV", "development"),
		CORSOrigin:       getEnv("CORS_ORIGIN", "*"),
	}

	// Security validation for production
	if cfg.AppEnv == "production" {
		if cfg.JWTSecret == "change-me-in-production" {
			return nil, fmt.Errorf("JWT_SECRET must be set to a secure value in production")
		}
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
