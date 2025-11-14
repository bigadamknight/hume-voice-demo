package api

import (
	"encoding/json"
	"net/http"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsAdmin  bool   `json:"is_admin"`
	Token    string `json:"token"`
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	// Get user from database
	user, err := s.db.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Check password
	if !s.auth.CheckPassword(req.Password, user.PasswordHash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT
	token, err := s.auth.GenerateJWT(user.ID.String(), user.Username, user.IsAdmin, s.config.JWTSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   60 * 60 * 24 * 7, // 7 days
	})

	json.NewEncoder(w).Encode(AuthResponse{
		UserID:   user.ID.String(),
		Username: user.Username,
		IsAdmin:  user.IsAdmin,
		Token:    token,
	})
}

func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:    -1,
	})
	w.WriteHeader(http.StatusOK)
}

func (s *Server) meHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	username := getUsername(r)
	isAdmin := isAdmin(r)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"user_id":  userID,
		"username": username,
		"is_admin": isAdmin,
	})
}

