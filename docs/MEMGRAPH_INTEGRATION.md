# Memgraph Knowledge Graph Integration

This system uses **Memgraph** to build AI-powered knowledge graphs from conversations, enabling intelligent context injection and pattern recognition.

## Architecture

```
Voice Conversation → PostgreSQL (raw transcript)
                           ↓
                    LLM Entity Extraction
                           ↓
                    Memgraph (knowledge graph)
                           ↓
                    Pattern Analysis
                           ↓
                    Context Injection → EVI
```

## Setup

### 1. Start Memgraph

```bash
cd web
docker-compose up -d memgraph
```

Memgraph runs on:
- **Bolt protocol**: `bolt://localhost:7688` (mapped from container port 7687)
- **Monitoring**: `http://localhost:7445` (mapped from container port 7444)

### 2. Install Dependencies

```bash
cd backend
go get github.com/neo4j/neo4j-go-driver/v5
go mod tidy
```

### 3. Configure

Add to `docker-compose.yml` backend environment:
```env
MEMGRAPH_URI=bolt://memgraph:7687  # Internal Docker network
MEMGRAPH_USER=
MEMGRAPH_PASSWORD=
OPENAI_API_KEY=${OPENAI_API_KEY}  # For LLM-based graph extraction
```

External connection (from host): `bolt://localhost:7688`

## API Endpoints

### Extract Knowledge Graph

**POST** `/api/graph/extract`

Analyzes conversation and extracts entities/relationships using LLM.

```bash
curl -X POST http://localhost/api/graph/extract \
  -H "Content-Type: application/json" \
  -d '{
    "conversationId": "...",
    "messages": [
      {"role": "user", "content": "I'm stressed about moving house"},
      {"role": "assistant", "content": "That sounds challenging..."}
    ]
  }'
```

**Response:**
```json
{
  "entities": [
    {"type": "Topic", "name": "moving_house", "properties": {"stress_level": "high"}},
    {"type": "Emotion", "name": "stress", "properties": {"intensity": "medium"}}
  ],
  "relationships": [
    {"from": "user", "to": "moving_house", "type": "DISCUSSES"},
    {"from": "user", "to": "stress", "type": "FEELS"}
  ]
}
```

### Get User Context

**GET** `/api/graph/user-context`

Retrieves patterns from knowledge graph to guide context injection.

```bash
curl http://localhost/api/graph/user-context
```

**Response:**
```json
{
  "recurring_topics": ["stress", "work", "moving"],
  "emotional_patterns": ["anxious", "overwhelmed"],
  "relationship_context": ["daughter", "family"]
}
```

## LLM Integration

### Step 1: Extract Graph from Conversation

Edit `web/backend/internal/api/graph_handler.go`:

```go
// In extractGraphHandler():

// Build prompt for LLM
prompt := buildGraphExtractionPrompt(req.Messages)

// Call OpenAI/Claude/your model
response, err := callLLM(prompt)

// Parse LLM response to GraphData
graphData := parseGraphResponse(response)

// Ingest into Memgraph
if s.graphClient != nil {
    err = s.graphClient.IngestGraphData(ctx, userID, req.ConversationID, graphData)
}
```

### Step 2: Example LLM Prompt

```
You are a knowledge graph extractor. Analyze this conversation and extract:

1. **Entities**: People, topics, emotions, locations, events
2. **Relationships**: How entities connect

Conversation:
User: "I'm stressed about moving house. My daughter lives far away."
Assistant: "That sounds challenging. How far away is she?"
User: "About an hour. She has mental health support there."

Return JSON:
{
  "entities": [
    {"type": "Topic", "name": "moving_house", "properties": {"stress_level": "high"}},
    {"type": "Person", "name": "daughter", "properties": {"relation": "child"}},
    {"type": "Emotion", "name": "stress", "properties": {"intensity": "medium"}},
    {"type": "Topic", "name": "mental_health", "properties": {"concern": "true"}},
    {"type": "Location", "name": "daughters_place", "properties": {"distance": "1_hour"}}
  ],
  "relationships": [
    {"from": "user", "to": "moving_house", "type": "DISCUSSES"},
    {"from": "user", "to": "stress", "type": "FEELS"},
    {"from": "user", "to": "daughter", "type": "HAS_CHILD"},
    {"from": "daughter", "to": "mental_health", "type": "REQUIRES_SUPPORT"},
    {"from": "daughter", "to": "daughters_place", "type": "LIVES_AT"}
  ]
}
```

### Step 3: Use Graph Context for Injection

Edit `web/backend/internal/api/context_handler.go`:

```go
// In analyzeConversationHandler():

// Get context from knowledge graph
graphContext, err := s.graphClient.GetUserContext(r.Context(), userID)

// Combine with current conversation analysis
if hasRecurringTopic(graphContext, "stress") {
    response.ShouldIntervene = true
    response.ContextText = "User has recurring stress patterns. Focus on coping strategies and breaking down problems."
    response.ContextType = "persistent"
}
```

## Query Examples

### Find Conversation Patterns

```cypher
// Find users who discuss similar topics
MATCH (u1:User)-[:DISCUSSES]->(t:Topic)<-[:DISCUSSES]-(u2:User)
RETURN u1.id, u2.id, t.name

// Find emotional triggers
MATCH (u:User)-[:DISCUSSES]->(topic)-[:TRIGGERS]->(emotion:Emotion)
RETURN topic.name, emotion.name, count(*) as frequency
ORDER BY frequency DESC

// Find effective interventions
MATCH (intervention:ContextInjection)-[:REDUCES]->(emotion:Emotion)
WHERE emotion.name = 'stress'
RETURN intervention.text, count(*) as effectiveness
ORDER BY effectiveness DESC
```

## Benefits

1. **Pattern Recognition**: Find what topics consistently stress users
2. **Personalization**: Remember user context across conversations
3. **Intervention Optimization**: Learn which context injections work
4. **Relationship Mapping**: Track family, work, social connections
5. **Temporal Analysis**: See how topics/emotions evolve over time

## Next Steps

1. Integrate OpenAI/Claude for entity extraction
2. Build automatic sync after each conversation
3. Create dashboard to visualize knowledge graph
4. Train on patterns to improve context injection
5. Use Memgraph's GraphRAG for retrieval-augmented responses

