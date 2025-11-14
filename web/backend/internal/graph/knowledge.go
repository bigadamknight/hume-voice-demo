package graph

import (
	"context"
	"fmt"
	"log"
)

// ConversationMessage represents a message in the conversation
type ConversationMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ExtractedEntity represents an entity extracted from conversation
type ExtractedEntity struct {
	Type       string            `json:"type"`       // e.g., "Person", "Topic", "Emotion"
	Name       string            `json:"name"`       // e.g., "stress", "work", "daughter"
	Properties map[string]string `json:"properties"` // Additional metadata
}

// ExtractedRelationship represents a relationship between entities
type ExtractedRelationship struct {
	From       string            `json:"from"`       // Entity name
	To         string            `json:"to"`         // Entity name
	Type       string            `json:"type"`       // e.g., "DISCUSSES", "FEELS", "HAS_CHILD"
	Properties map[string]string `json:"properties"` // Additional metadata
}

// GraphData represents the extracted graph structure
type GraphData struct {
	Entities      []ExtractedEntity      `json:"entities"`
	Relationships []ExtractedRelationship `json:"relationships"`
}

// SyncConversation syncs a conversation to the knowledge graph
func (c *Client) SyncConversation(ctx context.Context, userID, conversationID string, messages []ConversationMessage) error {
	// Create conversation node
	cypher := `
		MERGE (u:User {id: $userId})
		MERGE (c:Conversation {id: $conversationId})
		MERGE (u)-[:HAS_CONVERSATION]->(c)
		SET c.message_count = $messageCount
		SET c.updated_at = datetime()
	`
	params := map[string]interface{}{
		"userId":         userID,
		"conversationId": conversationID,
		"messageCount":   len(messages),
	}

	if err := c.ExecuteWrite(ctx, cypher, params); err != nil {
		return fmt.Errorf("failed to create conversation node: %w", err)
	}

	log.Printf("✅ Synced conversation %s with %d messages to Memgraph", conversationID, len(messages))
	return nil
}

// IngestGraphData ingests extracted entities and relationships into Memgraph
// This is called after an LLM extracts the graph structure from conversation text
func (c *Client) IngestGraphData(ctx context.Context, userID, conversationID string, data GraphData) error {
	// Create entities
	for _, entity := range data.Entities {
		cypher := fmt.Sprintf(`
			MERGE (e:%s {name: $name, conversationId: $conversationId})
			SET e += $properties
			SET e.updated_at = datetime()
		`, entity.Type)

		params := map[string]interface{}{
			"name":           entity.Name,
			"conversationId": conversationID,
			"properties":     entity.Properties,
		}

		if err := c.ExecuteWrite(ctx, cypher, params); err != nil {
			log.Printf("Warning: Failed to create entity %s: %v", entity.Name, err)
		}
	}

	// Create relationships
	for _, rel := range data.Relationships {
		cypher := fmt.Sprintf(`
			MATCH (from {name: $from, conversationId: $conversationId})
			MATCH (to {name: $to, conversationId: $conversationId})
			MERGE (from)-[r:%s]->(to)
			SET r += $properties
			SET r.timestamp = datetime()
		`, rel.Type)

		params := map[string]interface{}{
			"from":           rel.From,
			"to":             rel.To,
			"conversationId": conversationID,
			"properties":     rel.Properties,
		}

		if err := c.ExecuteWrite(ctx, cypher, params); err != nil {
			log.Printf("Warning: Failed to create relationship %s->%s: %v", rel.From, rel.To, err)
		}
	}

	log.Printf("✅ Ingested %d entities and %d relationships for conversation %s",
		len(data.Entities), len(data.Relationships), conversationID)
	return nil
}

// GetUserContext retrieves relevant context from the knowledge graph for a user
// This is used to guide context injection
func (c *Client) GetUserContext(ctx context.Context, userID string) (map[string]interface{}, error) {
	// Find recurring topics and patterns
	cypher := `
		MATCH (u:User {id: $userId})-[:HAS_CONVERSATION]->(c:Conversation)
		MATCH (c)-[:MENTIONS]->(topic:Topic)
		WITH topic, count(*) as mentions
		WHERE mentions > 1
		RETURN topic.name as topic, mentions
		ORDER BY mentions DESC
		LIMIT 5
	`

	params := map[string]interface{}{
		"userId": userID,
	}

	results, err := c.ExecuteRead(ctx, cypher, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get user context: %w", err)
	}

	context := map[string]interface{}{
		"recurring_topics": results,
	}

	// Find emotional patterns
	emotionCypher := `
		MATCH (u:User {id: $userId})-[:HAS_CONVERSATION]->()-[:EXPRESSES]->(e:Emotion)
		WITH e.name as emotion, count(*) as occurrences
		RETURN emotion, occurrences
		ORDER BY occurrences DESC
		LIMIT 3
	`

	emotions, err := c.ExecuteRead(ctx, emotionCypher, params)
	if err == nil {
		context["emotional_patterns"] = emotions
	}

	return context, nil
}


