package api

import (
	"encoding/json"
	"net/http"
)

// ConversationMessage represents a message in the conversation
type ConversationMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AnalyzeRequest represents the request to analyze a conversation
type AnalyzeRequest struct {
	History []ConversationMessage `json:"history"`
}

// AnalyzeResponse represents the AI analysis response
type AnalyzeResponse struct {
	ShouldIntervene bool   `json:"shouldIntervene"`
	ContextText     string `json:"contextText"`
	ContextType     string `json:"contextType"` // "temporary" or "persistent"
	Reasoning       string `json:"reasoning"`
}

// analyzeConversationHandler handles requests to analyze conversations with an external AI
func (s *Server) analyzeConversationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Integrate your external AI model here
	// Examples of what you might do:
	// 1. Call OpenAI/Claude/etc. to analyze sentiment and topics
	// 2. Detect patterns (e.g., user is stuck, needs redirection)
	// 3. Generate context to guide EVI's response
	//
	// Example integration:
	/*
		// Build prompt for your AI
		prompt := buildAnalysisPrompt(req.History)
		
		// Call your AI API (OpenAI, Claude, etc.)
		analysis, err := callExternalAI(prompt)
		if err != nil {
			http.Error(w, "Failed to analyze conversation", http.StatusInternalServerError)
			return
		}
		
		// Parse AI response to determine intervention
		response := AnalyzeResponse{
			ShouldIntervene: analysis.NeedsIntervention,
			ContextText:     analysis.SuggestedContext,
			ContextType:     "temporary", // or "persistent"
			Reasoning:       analysis.Reasoning,
		}
	*/

	// Default response - no intervention
	response := AnalyzeResponse{
		ShouldIntervene: false,
		ContextText:     "",
		ContextType:     "temporary",
		Reasoning:       "Default: No intervention needed",
	}

	// Example: Simple rule-based analysis
	if len(req.History) > 0 {
		lastMessage := req.History[len(req.History)-1]
		
		// Check for stress indicators
		if containsKeywords(lastMessage.Content, []string{"stress", "overwhelm", "anxious", "worried"}) {
			response.ShouldIntervene = true
			response.ContextText = "The user is expressing stress. Be empathetic, validate their feelings, and help them break down the problem into manageable steps."
			response.ContextType = "temporary"
			response.Reasoning = "Detected stress keywords in user message"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Helper function to check for keywords
func containsKeywords(text string, keywords []string) bool {
	lowerText := text
	for _, keyword := range keywords {
		if contains(lowerText, keyword) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	// Simple case-insensitive contains
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			c1 := s[i+j]
			c2 := substr[j]
			if c1 >= 'A' && c1 <= 'Z' {
				c1 += 'a' - 'A'
			}
			if c2 >= 'A' && c2 <= 'Z' {
				c2 += 'a' - 'A'
			}
			if c1 != c2 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

