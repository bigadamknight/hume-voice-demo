package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ExtractGraphRequest represents request to extract graph from conversation
type ExtractGraphRequest struct {
	ConversationID string                  `json:"conversationId"`
	Messages       []ConversationMessage   `json:"messages"`
}

// ExtractGraphResponse represents the extracted graph structure
type ExtractGraphResponse struct {
	Entities      []map[string]interface{} `json:"entities"`
	Relationships []map[string]interface{} `json:"relationships"`
}

// extractGraphHandler uses LLM to extract knowledge graph from conversation
func (s *Server) extractGraphHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExtractGraphRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// TODO: Integrate your LLM here to extract entities and relationships
	// Example prompt structure:
	/*
		prompt := `Analyze this conversation and extract:
		1. Entities (people, topics, emotions, locations, etc.)
		2. Relationships between entities
		
		Conversation:
		` + formatMessages(req.Messages) + `
		
		Return JSON:
		{
		  "entities": [
		    {"type": "Person", "name": "daughter", "properties": {"relation": "child"}},
		    {"type": "Topic", "name": "moving_house", "properties": {"stress_level": "high"}},
		    {"type": "Emotion", "name": "stress", "properties": {"intensity": "medium"}}
		  ],
		  "relationships": [
		    {"from": "user", "to": "daughter", "type": "HAS_CHILD"},
		    {"from": "user", "to": "moving_house", "type": "DISCUSSES"},
		    {"from": "user", "to": "stress", "type": "FEELS"}
		  ]
		}`
		
		// Call OpenAI/Claude/etc.
		response := callLLM(prompt)
		graphData := parseGraphData(response)
	*/

	// Default response - minimal extraction
	response := ExtractGraphResponse{
		Entities: []map[string]interface{}{
			{
				"type":       "Conversation",
				"name":       req.ConversationID,
				"properties": map[string]string{"message_count": fmt.Sprintf("%d", len(req.Messages))},
			},
		},
		Relationships: []map[string]interface{}{},
	}

	// Simple keyword-based extraction (replace with LLM)
	topics := extractTopics(req.Messages)
	for _, topic := range topics {
		response.Entities = append(response.Entities, map[string]interface{}{
			"type":       "Topic",
			"name":       topic,
			"properties": map[string]string{},
		})
		response.Relationships = append(response.Relationships, map[string]interface{}{
			"from": "user",
			"to":   topic,
			"type": "DISCUSSES",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Simple topic extraction (replace with LLM-based extraction)
func extractTopics(messages []ConversationMessage) []string {
	keywords := map[string]bool{
		"work":    false,
		"stress":  false,
		"moving":  false,
		"house":   false,
		"family":  false,
		"health":  false,
		"project": false,
	}

	for _, msg := range messages {
		content := msg.Content
		for keyword := range keywords {
			if contains(content, keyword) {
				keywords[keyword] = true
			}
		}
	}

	var topics []string
	for keyword, found := range keywords {
		if found {
			topics = append(topics, keyword)
		}
	}

	return topics
}

// getUserGraphContextHandler retrieves context from the knowledge graph
func (s *Server) getUserGraphContextHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Get user ID from JWT/session
	// userID := "shared-user-id"

	// TODO: Query Memgraph for user context
	/*
		if s.graphClient != nil {
			context, err := s.graphClient.GetUserContext(r.Context(), "shared-user-id")
			if err != nil {
				log.Printf("Failed to get graph context: %v", err)
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(context)
				return
			}
		}
	*/

	// Default response
	response := map[string]interface{}{
		"recurring_topics":    []string{},
		"emotional_patterns":  []string{},
		"relationship_context": []string{},
		"note": "Memgraph integration pending - see graph_handler.go",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}


