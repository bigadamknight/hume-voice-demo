package api

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type AddMessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (s *Server) addMessageHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := getUserID(r)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	convID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid conversation ID", http.StatusBadRequest)
		return
	}

	// Verify conversation belongs to user
	_, err = s.db.GetConversation(r.Context(), convID, userID)
	if err != nil {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	var req AddMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Role == "" || req.Content == "" {
		http.Error(w, "Role and content required", http.StatusBadRequest)
		return
	}

	msg, err := s.db.AddMessage(r.Context(), convID, req.Role, req.Content)
	if err != nil {
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msg)
}

