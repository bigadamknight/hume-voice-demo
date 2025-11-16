package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CreateUserRequest struct {
	Username string  `json:"username"`
	Password string  `json:"password"`
	Name     *string `json:"name,omitempty"`
	IsAdmin  bool    `json:"is_admin"`
}

type UpdateUserRequest struct {
	Password *string `json:"password,omitempty"`
	Name     *string `json:"name,omitempty"`
	IsAdmin  *bool   `json:"is_admin,omitempty"`
}

func (s *Server) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.ListUsers(r.Context())
	if err != nil {
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}

	// Remove password hashes from response
	for i := range users {
		users[i].PasswordHash = ""
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (s *Server) createUserHandler(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	// Hash password
	passwordHash, err := s.auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Create user
	user, err := s.db.CreateUser(r.Context(), req.Username, passwordHash, req.Name, req.IsAdmin)
	if err != nil {
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Remove password hash from response
	user.PasswordHash = ""

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

func (s *Server) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Verify user exists
	_, err = s.db.GetUserByID(r.Context(), id)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Update password if provided
	var passwordHash *string
	if req.Password != nil {
		hash, err := s.auth.HashPassword(*req.Password)
		if err != nil {
			http.Error(w, "Failed to hash password", http.StatusInternalServerError)
			return
		}
		passwordHash = &hash
	}

	// Update user
	err = s.db.UpdateUser(r.Context(), id, passwordHash, req.Name, req.IsAdmin)
	if err != nil {
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	// Get updated user
	updatedUser, err := s.db.GetUserByID(r.Context(), id)
	if err != nil {
		http.Error(w, "Failed to get updated user", http.StatusInternalServerError)
		return
	}

	// Remove password hash from response
	updatedUser.PasswordHash = ""

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedUser)
}

func (s *Server) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Prevent deleting yourself
	currentUserID := getUserID(r)
	if currentUserID == id.String() {
		http.Error(w, "Cannot delete your own account", http.StatusBadRequest)
		return
	}

	if err := s.db.DeleteUser(r.Context(), id); err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

