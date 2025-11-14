package api

import (
	"context"
	"net/http"
)

type contextKey string

const userIDKey contextKey = "user_id"
const usernameKey contextKey = "username"
const isAdminKey contextKey = "is_admin"

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from cookie
		cookie, err := r.Cookie("auth_token")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate token
		claims, err := s.auth.ValidateJWT(cookie.Value, s.config.JWTSecret)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Extract user info from claims
		userID, ok := claims["user_id"].(string)
		if !ok {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		username, ok := claims["username"].(string)
		if !ok {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Extract is_admin (default to false if not present for backward compatibility)
		isAdmin := false
		if adminVal, ok := claims["is_admin"].(bool); ok {
			isAdmin = adminVal
		}

		// Add to context
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		ctx = context.WithValue(ctx, usernameKey, username)
		ctx = context.WithValue(ctx, isAdminKey, isAdmin)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func getUserID(r *http.Request) string {
	if userID, ok := r.Context().Value(userIDKey).(string); ok {
		return userID
	}
	return ""
}

func getUsername(r *http.Request) string {
	if username, ok := r.Context().Value(usernameKey).(string); ok {
		return username
	}
	return ""
}

func isAdmin(r *http.Request) bool {
	if admin, ok := r.Context().Value(isAdminKey).(bool); ok {
		return admin
	}
	return false
}

// requireAdminMiddleware ensures the user is an admin
func (s *Server) requireAdminMiddleware(next http.Handler) http.Handler {
	return s.authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAdmin(r) {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

