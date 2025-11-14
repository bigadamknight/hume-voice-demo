package db

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	IsAdmin      bool      `json:"is_admin"`
	CreatedAt    time.Time `json:"created_at"`
}

type Conversation struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	MessageCount int    `json:"message_count,omitempty"`
}

type Message struct {
	ID             uuid.UUID `json:"id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	Timestamp      time.Time `json:"timestamp"`
}

type Voice struct {
	ID                    uuid.UUID `json:"id"`
	Name                  string    `json:"name"`
	Description           string    `json:"description"`
	Prompt                string    `json:"prompt"`
	VoiceDescription      string    `json:"voice_description"`
	HumeVoiceID           string    `json:"hume_voice_id"`
	HumeConfigID          string    `json:"hume_config_id"`
	EVIVersion            string    `json:"evi_version"`
	LanguageModelProvider string    `json:"language_model_provider"`
	LanguageModelResource string    `json:"language_model_resource"`
	Temperature           float64   `json:"temperature"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// User methods
func (db *DB) CreateUser(ctx context.Context, username, passwordHash string, isAdmin bool) (*User, error) {
	var user User
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO users (username, password_hash, is_admin) VALUES ($1, $2, $3) RETURNING id, username, password_hash, is_admin, created_at`,
		username, passwordHash, isAdmin,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt)
	return &user, err
}

func (db *DB) CreateUserWithID(ctx context.Context, id uuid.UUID, username, passwordHash string, isAdmin bool) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO users (id, username, password_hash, is_admin) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO UPDATE SET username = EXCLUDED.username, password_hash = EXCLUDED.password_hash, is_admin = EXCLUDED.is_admin`,
		id, username, passwordHash, isAdmin)
	return err
}

func (db *DB) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	var user User
	err := db.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, is_admin, created_at FROM users WHERE username = $1`,
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt)
	return &user, err
}

func (db *DB) GetUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	err := db.Pool.QueryRow(ctx,
		`SELECT id, username, password_hash, is_admin, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt)
	return &user, err
}

func (db *DB) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, username, password_hash, is_admin, created_at FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (db *DB) UpdateUserPassword(ctx context.Context, id uuid.UUID, passwordHash string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE users SET password_hash = $1 WHERE id = $2`,
		passwordHash, id,
	)
	return err
}

func (db *DB) UpdateUserAdmin(ctx context.Context, id uuid.UUID, isAdmin bool) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE users SET is_admin = $1 WHERE id = $2`,
		isAdmin, id,
	)
	return err
}

func (db *DB) UpdateUser(ctx context.Context, id uuid.UUID, passwordHash *string, isAdmin *bool) error {
	if passwordHash != nil && isAdmin != nil {
		_, err := db.Pool.Exec(ctx,
			`UPDATE users SET password_hash = $1, is_admin = $2 WHERE id = $3`,
			*passwordHash, *isAdmin, id,
		)
		return err
	} else if passwordHash != nil {
		return db.UpdateUserPassword(ctx, id, *passwordHash)
	} else if isAdmin != nil {
		return db.UpdateUserAdmin(ctx, id, *isAdmin)
	}
	return nil
}

func (db *DB) DeleteUser(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		`DELETE FROM users WHERE id = $1`,
		id,
	)
	return err
}

// Conversation methods
func (db *DB) CreateConversation(ctx context.Context, userID uuid.UUID, title string) (*Conversation, error) {
	var conv Conversation
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO conversations (user_id, title) VALUES ($1, $2) RETURNING id, user_id, title, status, created_at, updated_at`,
		userID, title,
	).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.Status, &conv.CreatedAt, &conv.UpdatedAt)
	return &conv, err
}

func (db *DB) GetConversation(ctx context.Context, id, userID uuid.UUID) (*Conversation, error) {
	var conv Conversation
	err := db.Pool.QueryRow(ctx,
		`SELECT id, user_id, title, status, created_at, updated_at FROM conversations WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.Status, &conv.CreatedAt, &conv.UpdatedAt)
	return &conv, err
}

func (db *DB) ListConversations(ctx context.Context, userID uuid.UUID, limit int) ([]Conversation, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT c.id, c.user_id, c.title, c.status, c.created_at, c.updated_at, COUNT(m.id) as message_count
		 FROM conversations c
		 LEFT JOIN messages m ON c.id = m.conversation_id
		 WHERE c.user_id = $1
		 GROUP BY c.id
		 ORDER BY c.updated_at DESC
		 LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var conv Conversation
		err := rows.Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.Status, &conv.CreatedAt, &conv.UpdatedAt, &conv.MessageCount)
		if err != nil {
			return nil, err
		}
		conversations = append(conversations, conv)
	}
	return conversations, rows.Err()
}

func (db *DB) UpdateConversationStatus(ctx context.Context, id, userID uuid.UUID, status string) error {
	_, err := db.Pool.Exec(ctx,
		`UPDATE conversations SET status = $1 WHERE id = $2 AND user_id = $3`,
		status, id, userID,
	)
	return err
}

func (db *DB) GetLastActiveConversation(ctx context.Context, userID uuid.UUID) (*Conversation, error) {
	var conv Conversation
	err := db.Pool.QueryRow(ctx,
		`SELECT id, user_id, title, status, created_at, updated_at FROM conversations 
		 WHERE user_id = $1 AND status = 'active' 
		 ORDER BY updated_at DESC LIMIT 1`,
		userID,
	).Scan(&conv.ID, &conv.UserID, &conv.Title, &conv.Status, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

func (db *DB) DeleteConversation(ctx context.Context, id, userID uuid.UUID) error {
	_, err := db.Pool.Exec(ctx,
		`DELETE FROM conversations WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	return err
}

// Message methods
func (db *DB) AddMessage(ctx context.Context, conversationID uuid.UUID, role, content string) (*Message, error) {
	var msg Message
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO messages (conversation_id, role, content) VALUES ($1, $2, $3) RETURNING id, conversation_id, role, content, timestamp`,
		conversationID, role, content,
	).Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.Timestamp)
	
	// Update conversation updated_at
	_, _ = db.Pool.Exec(ctx,
		`UPDATE conversations SET updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
		conversationID,
	)
	
	return &msg, err
}

func (db *DB) GetMessages(ctx context.Context, conversationID, userID uuid.UUID) ([]Message, error) {
	// Verify conversation belongs to user
	var convID uuid.UUID
	err := db.Pool.QueryRow(ctx,
		`SELECT id FROM conversations WHERE id = $1 AND user_id = $2`,
		conversationID, userID,
	).Scan(&convID)
	if err != nil {
		return nil, err
	}

	rows, err := db.Pool.Query(ctx,
		`SELECT id, conversation_id, role, content, timestamp FROM messages 
		 WHERE conversation_id = $1 ORDER BY timestamp ASC`,
		conversationID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.Timestamp)
		if err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// Voice methods
func (db *DB) CreateVoice(ctx context.Context, voice *Voice) (*Voice, error) {
	var created Voice
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO voices (name, description, prompt, voice_description, hume_voice_id, hume_config_id, evi_version, language_model_provider, language_model_resource, temperature)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 RETURNING id, name, description, prompt, voice_description, hume_voice_id, hume_config_id, evi_version, language_model_provider, language_model_resource, temperature, created_at, updated_at`,
		voice.Name, voice.Description, voice.Prompt, voice.VoiceDescription, voice.HumeVoiceID, voice.HumeConfigID, voice.EVIVersion, voice.LanguageModelProvider, voice.LanguageModelResource, voice.Temperature,
	).Scan(&created.ID, &created.Name, &created.Description, &created.Prompt, &created.VoiceDescription, &created.HumeVoiceID, &created.HumeConfigID, &created.EVIVersion, &created.LanguageModelProvider, &created.LanguageModelResource, &created.Temperature, &created.CreatedAt, &created.UpdatedAt)
	return &created, err
}

func (db *DB) GetVoice(ctx context.Context, id uuid.UUID) (*Voice, error) {
	var voice Voice
	err := db.Pool.QueryRow(ctx,
		`SELECT id, name, description, prompt, voice_description, hume_voice_id, hume_config_id, evi_version, language_model_provider, language_model_resource, temperature, created_at, updated_at
		 FROM voices WHERE id = $1`,
		id,
	).Scan(&voice.ID, &voice.Name, &voice.Description, &voice.Prompt, &voice.VoiceDescription, &voice.HumeVoiceID, &voice.HumeConfigID, &voice.EVIVersion, &voice.LanguageModelProvider, &voice.LanguageModelResource, &voice.Temperature, &voice.CreatedAt, &voice.UpdatedAt)
	return &voice, err
}

func (db *DB) ListVoices(ctx context.Context) ([]Voice, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, name, description, prompt, voice_description, hume_voice_id, hume_config_id, evi_version, language_model_provider, language_model_resource, temperature, created_at, updated_at
		 FROM voices ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var voices []Voice
	for rows.Next() {
		var voice Voice
		err := rows.Scan(&voice.ID, &voice.Name, &voice.Description, &voice.Prompt, &voice.VoiceDescription, &voice.HumeVoiceID, &voice.HumeConfigID, &voice.EVIVersion, &voice.LanguageModelProvider, &voice.LanguageModelResource, &voice.Temperature, &voice.CreatedAt, &voice.UpdatedAt)
		if err != nil {
			return nil, err
		}
		voices = append(voices, voice)
	}
	return voices, rows.Err()
}

func (db *DB) UpdateVoice(ctx context.Context, id uuid.UUID, voice *Voice) (*Voice, error) {
	var updated Voice
	err := db.Pool.QueryRow(ctx,
		`UPDATE voices SET name = $1, description = $2, prompt = $3, voice_description = $4, hume_voice_id = $5, hume_config_id = $6, evi_version = $7, language_model_provider = $8, language_model_resource = $9, temperature = $10
		 WHERE id = $11
		 RETURNING id, name, description, prompt, voice_description, hume_voice_id, hume_config_id, evi_version, language_model_provider, language_model_resource, temperature, created_at, updated_at`,
		voice.Name, voice.Description, voice.Prompt, voice.VoiceDescription, voice.HumeVoiceID, voice.HumeConfigID, voice.EVIVersion, voice.LanguageModelProvider, voice.LanguageModelResource, voice.Temperature, id,
	).Scan(&updated.ID, &updated.Name, &updated.Description, &updated.Prompt, &updated.VoiceDescription, &updated.HumeVoiceID, &updated.HumeConfigID, &updated.EVIVersion, &updated.LanguageModelProvider, &updated.LanguageModelResource, &updated.Temperature, &updated.CreatedAt, &updated.UpdatedAt)
	return &updated, err
}

func (db *DB) DeleteVoice(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM voices WHERE id = $1`, id)
	return err
}

