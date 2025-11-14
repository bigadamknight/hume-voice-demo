package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/hume-evi/web/internal/db"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

type Client struct {
	hub              *Hub
	conn             *websocket.Conn
	send             chan []byte
	userID           string
	conversationID   *uuid.UUID
	humeConn         *websocket.Conn
	humeMutex        sync.Mutex
	db               *db.DB
	humeAPIKey       string
	humeConfigID     string
	ctx              context.Context
	cancel           context.CancelFunc
}

type HumeMessage struct {
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data,omitempty"`
	Message *MessageContent `json:"message,omitempty"`
	Code    string          `json:"code,omitempty"`
	Slug    string          `json:"slug,omitempty"`
}

type MessageContent struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AudioInputMessage struct {
	Type string `json:"type"`
	Data string `json:"data"` // base64 encoded audio
}

type SessionSettings struct {
	Type  string      `json:"type"`
	Audio AudioConfig `json:"audio"`
}

type AudioConfig struct {
	Encoding   string `json:"encoding"`
	Format     string `json:"format"`
	SampleRate int    `json:"sample_rate"`
	Channels   int    `json:"channels"`
}

func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, userID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &Client{
		hub:          hub,
		conn:         conn,
		send:         make(chan []byte, 256),
		userID:       userID,
		db:           hub.db,
		humeAPIKey:   hub.config.HumeAPIKey,
		humeConfigID: hub.config.HumeConfigID,
		ctx:          ctx,
		cancel:       cancel,
	}

	hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
		if c.humeConn != nil {
			c.humeConn.Close()
		}
		c.cancel()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming message from frontend
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse message: %v, raw: %s", err, string(message))
			continue
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			log.Printf("Message missing type field: %v", msg)
			continue
		}

		log.Printf("Received WebSocket message type: %s", msgType)

		switch msgType {
		case "start_conversation":
			c.handleStartConversation(msg)
		case "audio_input":
			c.handleAudioInput(msg)
		case "end_conversation":
			c.handleEndConversation()
		default:
			log.Printf("Unknown message type: %s", msgType)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) handleStartConversation(msg map[string]interface{}) {
	log.Printf("handleStartConversation called")
	var convID *uuid.UUID
	if idStr, ok := msg["conversation_id"].(string); ok && idStr != "" {
		id, err := uuid.Parse(idStr)
		if err == nil {
			convID = &id
			log.Printf("Resuming conversation: %s", idStr)
		}
	}

	// Create new conversation if needed
	if convID == nil {
		userUUID, _ := uuid.Parse(c.userID)
		conv, err := c.db.CreateConversation(c.ctx, userUUID, "")
		if err != nil {
			log.Printf("Failed to create conversation: %v", err)
			c.sendError("Failed to create conversation")
			return
		}
		convID = &conv.ID
		log.Printf("Created new conversation: %s", conv.ID)
	} else {
		// Verify conversation belongs to user
		userUUID, _ := uuid.Parse(c.userID)
		_, err := c.db.GetConversation(c.ctx, *convID, userUUID)
		if err != nil {
			log.Printf("Conversation not found: %v", err)
			c.sendError("Conversation not found")
			return
		}
	}

	c.conversationID = convID

	// Connect to Hume EVI
	log.Printf("Attempting to connect to Hume EVI...")
	if err := c.connectToHume(); err != nil {
		log.Printf("Failed to connect to Hume EVI: %v", err)
		c.sendError("Failed to connect to Hume EVI: " + err.Error())
		return
	}

	log.Printf("Hume EVI connected, starting read loop")
	// Start reading from Hume
	go c.readFromHume()
}

func (c *Client) connectToHume() error {
	// Hume WebSocket URL with config_id as query parameter
	url := fmt.Sprintf("wss://api.hume.ai/v0/evi/chat?config_id=%s", c.humeConfigID)
	
	// Create WebSocket connection with auth header
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
	}
	headers := http.Header{}
	headers.Set("X-Hume-Api-Key", c.humeAPIKey)

	log.Printf("Connecting to Hume EVI at %s", url)
	conn, resp, err := dialer.Dial(url, headers)
	if err != nil {
		if resp != nil {
			log.Printf("Hume connection failed with status: %d, headers: %v", resp.StatusCode, resp.Header)
		}
		return fmt.Errorf("failed to connect to Hume: %w", err)
	}

	log.Printf("Connected to Hume EVI WebSocket")

	c.humeMutex.Lock()
	c.humeConn = conn
	c.humeMutex.Unlock()

	// Send session settings for audio format (with encoding field)
	sessionSettings := SessionSettings{
		Type: "session_settings",
		Audio: AudioConfig{
			Encoding:  "linear16",
			Format:     "linear16",
			SampleRate: 44100,
			Channels:   1,
		},
	}

	settingsJSON, err := json.Marshal(sessionSettings)
	if err != nil {
		return fmt.Errorf("failed to marshal session settings: %w", err)
	}
	
	log.Printf("Sending session settings")
	if err := conn.WriteMessage(websocket.TextMessage, settingsJSON); err != nil {
		return fmt.Errorf("failed to send session settings: %w", err)
	}

	log.Printf("Hume EVI connection established - readFromHume will handle responses")
	return nil
}

func (c *Client) handleAudioInput(msg map[string]interface{}) {
	if c.humeConn == nil {
		return
	}

	data, ok := msg["data"].(string)
	if !ok {
		return
	}

	// Forward audio to Hume
	audioMsg := AudioInputMessage{
		Type: "audio_input",
		Data: data,
	}

	audioJSON, err := json.Marshal(audioMsg)
	if err != nil {
		return
	}

	c.humeMutex.Lock()
	err = c.humeConn.WriteMessage(websocket.TextMessage, audioJSON)
	c.humeMutex.Unlock()

	if err != nil {
		log.Printf("Failed to send audio to Hume: %v", err)
	}
}

func (c *Client) readFromHume() {
	defer func() {
		if c.humeConn != nil {
			c.humeConn.Close()
		}
		log.Printf("Hume read loop ended")
	}()

	for {
		if c.humeConn == nil {
			return
		}

		_, message, err := c.humeConn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Hume connection error: %v", err)
			}
			return
		}

		var humeMsg HumeMessage
		if err := json.Unmarshal(message, &humeMsg); err != nil {
			log.Printf("Failed to parse Hume message: %v, raw: %s", err, string(message))
			continue
		}

		log.Printf("Received Hume message type: %s", humeMsg.Type)

		// Handle different message types
		switch humeMsg.Type {
		case "user_message", "assistant_message":
			c.handleTextMessage(humeMsg)
		case "audio_output":
			c.handleAudioOutput(humeMsg)
		case "user_interruption":
			c.handleInterruption()
		case "error":
			log.Printf("Hume error: code=%s, slug=%s", humeMsg.Code, humeMsg.Slug)
			c.sendError("Hume error: " + humeMsg.Slug)
		default:
			log.Printf("Unhandled Hume message type: %s", humeMsg.Type)
		}
	}
}

func (c *Client) handleTextMessage(msg HumeMessage) {
	if msg.Message == nil || c.conversationID == nil {
		return
	}

	// Save to database
	role := "assistant"
	if msg.Type == "user_message" {
		role = "user"
	}

	_, err := c.db.AddMessage(c.ctx, *c.conversationID, role, msg.Message.Content)
	if err != nil {
		log.Printf("Failed to save message: %v", err)
	}

	// Forward to frontend
	response := map[string]interface{}{
		"type":    msg.Type,
		"role":    msg.Message.Role,
		"content": msg.Message.Content,
	}

	responseJSON, _ := json.Marshal(response)
	select {
	case c.send <- responseJSON:
	default:
	}
}

func (c *Client) handleAudioOutput(msg HumeMessage) {
	// Forward audio to frontend
	response := map[string]interface{}{
		"type": "audio_output",
		"data": string(msg.Data),
	}

	responseJSON, _ := json.Marshal(response)
	select {
	case c.send <- responseJSON:
	default:
	}
}

func (c *Client) handleInterruption() {
	// Notify frontend to clear audio queue
	response := map[string]interface{}{
		"type": "user_interruption",
	}

	responseJSON, _ := json.Marshal(response)
	select {
	case c.send <- responseJSON:
	default:
	}
}

func (c *Client) handleEndConversation() {
	if c.conversationID != nil {
		userUUID, _ := uuid.Parse(c.userID)
		_ = c.db.UpdateConversationStatus(c.ctx, *c.conversationID, userUUID, "paused")
	}

	if c.humeConn != nil {
		c.humeConn.Close()
		c.humeConn = nil
	}
}

func (c *Client) sendError(message string) {
	response := map[string]interface{}{
		"type":    "error",
		"message": message,
	}

	responseJSON, _ := json.Marshal(response)
	select {
	case c.send <- responseJSON:
	default:
	}
}

