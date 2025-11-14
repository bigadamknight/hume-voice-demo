# AI-Powered Knowledge Graph Extraction

## How It Works

### Step 1: Conversation Happens

User and EVI have a conversation:
```
User: "I'm stressed about moving house. My daughter lives an hour away."
Assistant: "That sounds challenging. How far away is she?"
User: "About an hour. She has mental health support there."
```

Saved to PostgreSQL as raw transcript.

### Step 2: LLM Extracts Graph Structure

Call OpenAI/Claude with extraction prompt:

```typescript
const prompt = `
You are a knowledge graph extractor. Analyze this conversation and extract:

1. **Entities**: People, topics, emotions, locations, events
   - Each entity should have: type, name, properties (key-value pairs)

2. **Relationships**: How entities connect
   - Each relationship: from, to, type, properties

Format as JSON. Be specific and capture nuanced relationships.

Conversation:
${messages.map(m => `${m.role}: ${m.content}`).join('\n')}

Return ONLY valid JSON:
{
  "entities": [...],
  "relationships": [...]
}
`

const response = await openai.chat.completions.create({
  model: "gpt-4",
  messages: [{ role: "user", content: prompt }],
  response_format: { type: "json_object" }
})

const graphData = JSON.parse(response.choices[0].message.content)
```

### Step 3: LLM Returns Structured Data

```json
{
  "entities": [
    {
      "type": "Topic",
      "name": "moving_house",
      "properties": {
        "stress_level": "high",
        "status": "in_progress",
        "context": "downsizing"
      }
    },
    {
      "type": "Person",
      "name": "daughter",
      "properties": {
        "relation": "child",
        "status": "independent"
      }
    },
    {
      "type": "Emotion",
      "name": "stress",
      "properties": {
        "intensity": "medium",
        "source": "moving_house"
      }
    },
    {
      "type": "Topic",
      "name": "mental_health",
      "properties": {
        "concerns": "daughter_wellbeing",
        "support_type": "professional_care"
      }
    },
    {
      "type": "Location",
      "name": "daughters_residence",
      "properties": {
        "distance_from_user": "1_hour",
        "has_support": "true"
      }
    }
  ],
  "relationships": [
    {
      "from": "user",
      "to": "moving_house",
      "type": "CURRENTLY_DOING",
      "properties": {"urgency": "high"}
    },
    {
      "from": "user",
      "to": "stress",
      "type": "FEELS",
      "properties": {"trigger": "moving_house"}
    },
    {
      "from": "user",
      "to": "daughter",
      "type": "HAS_CHILD",
      "properties": {"emotional_connection": "close"}
    },
    {
      "from": "daughter",
      "to": "daughters_residence",
      "type": "LIVES_AT",
      "properties": {"distance": "1_hour"}
    },
    {
      "from": "daughter",
      "to": "mental_health",
      "type": "REQUIRES_SUPPORT",
      "properties": {"type": "professional"}
    },
    {
      "from": "moving_house",
      "to": "stress",
      "type": "CAUSES",
      "properties": {"severity": "medium"}
    }
  ]
}
```

### Step 4: Store in Memgraph

Convert to Cypher queries:

```cypher
// Create entities
MERGE (t:Topic {name: 'moving_house'})
SET t.stress_level = 'high',
    t.status = 'in_progress',
    t.context = 'downsizing'

MERGE (p:Person {name: 'daughter'})
SET p.relation = 'child',
    p.status = 'independent'

MERGE (e:Emotion {name: 'stress'})
SET e.intensity = 'medium',
    e.source = 'moving_house'

// Create relationships
MATCH (u:User {id: 'adam'})
MATCH (t:Topic {name: 'moving_house'})
MERGE (u)-[r:CURRENTLY_DOING]->(t)
SET r.urgency = 'high'

MATCH (u:User {id: 'adam'})
MATCH (e:Emotion {name: 'stress'})
MERGE (u)-[r:FEELS]->(e)
SET r.trigger = 'moving_house'

MATCH (u:User {id: 'adam'})
MATCH (p:Person {name: 'daughter'})
MERGE (u)-[r:HAS_CHILD]->(p)
SET r.emotional_connection = 'close'
```

### Step 5: Query for Context Injection

Before EVI responds, query the graph:

```cypher
// Find what's stressing the user
MATCH (u:User {id: 'adam'})-[:FEELS]->(e:Emotion {name: 'stress'})<-[:CAUSES]-(cause)
RETURN cause.name, e.intensity

// Find recurring topics
MATCH (u:User {id: 'adam'})-[:DISCUSSES]->(t:Topic)
WITH t.name as topic, count(*) as mentions
WHERE mentions > 2
RETURN topic, mentions
ORDER BY mentions DESC
```

**Returns:**
- User is stressed about `moving_house` (intensity: medium)
- `moving_house` mentioned 5 times across conversations

### Step 6: Inject Smart Context

Use graph insights to guide EVI:

```typescript
// Graph shows: user stressed + moving_house recurring
const contextText = `
User is in the middle of moving house (downsizing). 
Daughter lives 1 hour away with mental health support.
User has recurring stress about this. 
Be supportive, help break down moving tasks, acknowledge family situation.
`

socket.sendSessionSettings({
  type: 'session_settings',
  context: {
    text: contextText,
    type: 'persistent'  // Active for whole conversation
  }
})
```

## Full Integration Code

### Backend: Extract Graph (Go + OpenAI)

```go
package api

import (
    "context"
    "encoding/json"
    "os"
    
    "github.com/sashabaranov/go-openai"
)

func (s *Server) extractWithLLM(messages []Message) (*GraphData, error) {
    client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
    
    // Build prompt
    prompt := buildExtractionPrompt(messages)
    
    // Call OpenAI
    resp, err := client.CreateChatCompletion(
        context.Background(),
        openai.ChatCompletionRequest{
            Model: openai.GPT4,
            Messages: []openai.ChatCompletionMessage{
                {Role: "user", Content: prompt},
            },
            ResponseFormat: &openai.ChatCompletionResponseFormat{
                Type: "json_object",
            },
        },
    )
    
    // Parse response
    var graphData GraphData
    json.Unmarshal([]byte(resp.Choices[0].Message.Content), &graphData)
    
    // Store in Memgraph
    s.graphClient.IngestGraphData(ctx, userID, conversationID, graphData)
    
    return &graphData, nil
}
```

### Frontend: Trigger Extraction

```typescript
// After conversation ends or periodically
async function syncToGraph(conversationId: string) {
  const messages = await fetch(`/api/conversations/${conversationId}/messages`)
  const data = await messages.json()
  
  // Trigger LLM extraction
  await fetch('/api/graph/extract', {
    method: 'POST',
    body: JSON.stringify({
      conversationId,
      messages: data
    })
  })
}
```

## Benefits

1. **Persistent Memory** - Graph remembers user context across sessions
2. **Pattern Recognition** - Find what triggers stress, what helps
3. **Relationship Mapping** - Understand user's life context
4. **Intelligent Intervention** - Context injection based on patterns, not keywords
5. **GraphRAG** - Use graph context to augment LLM responses

## Example Queries

```cypher
// Find all topics that cause stress
MATCH (t:Topic)-[:CAUSES]->(e:Emotion {name: 'stress'})
RETURN t.name, t.stress_level

// Find effective interventions
MATCH (ci:ContextInjection)-[:APPLIED_TO]->(c:Conversation)
MATCH (c)-[:RESULTED_IN]->(outcome:Emotion)
WHERE outcome.intensity < 'medium'
RETURN ci.text, count(*) as effective_count
ORDER BY effective_count DESC

// Find user's support network
MATCH (u:User)-[:HAS_CHILD|WORKS_WITH|LIVES_WITH]->(p:Person)
RETURN p.name, type(r) as relationship
```

## Next Steps

1. **Add OpenAI SDK**: `go get github.com/sashabaranov/go-openai`
2. **Implement LLM extraction** in `graph_handler.go`
3. **Auto-sync** after each conversation
4. **Query graph** before context injection
5. **Visualize** using Memgraph Lab: `http://localhost:7445`


