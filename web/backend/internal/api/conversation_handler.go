package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type CreateConversationRequest struct {
	Title string `json:"title"`
}

func (s *Server) createConversationHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := getUserID(r)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req CreateConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.Title = "" // Default to empty, will be set to timestamp
	}

	if req.Title == "" {
		req.Title = "Conversation " + time.Now().Format("2006-01-02 15:04:05")
	}

	conv, err := s.db.CreateConversation(r.Context(), userID, req.Title)
	if err != nil {
		http.Error(w, "Failed to create conversation", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

func (s *Server) listConversationsHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := getUserID(r)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	conversations, err := s.db.ListConversations(r.Context(), userID, limit)
	if err != nil {
		http.Error(w, "Failed to list conversations", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conversations)
}

func (s *Server) getConversationHandler(w http.ResponseWriter, r *http.Request) {
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

	conv, err := s.db.GetConversation(r.Context(), convID, userID)
	if err != nil {
		http.Error(w, "Conversation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

func (s *Server) getMessagesHandler(w http.ResponseWriter, r *http.Request) {
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

	messages, err := s.db.GetMessages(r.Context(), convID, userID)
	if err != nil {
		http.Error(w, "Failed to get messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (s *Server) updateConversationStatusHandler(w http.ResponseWriter, r *http.Request) {
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

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateConversationStatus(r.Context(), convID, userID, req.Status); err != nil {
		http.Error(w, "Failed to update conversation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) deleteConversationHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := s.db.DeleteConversation(r.Context(), convID, userID); err != nil {
		http.Error(w, "Failed to delete conversation", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) getLastActiveConversationHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := getUserID(r)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	conv, err := s.db.GetLastActiveConversation(r.Context(), userID)
	if err != nil {
		// No active conversation found
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conv)
}

