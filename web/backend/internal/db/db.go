package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func New(databaseURL string) (*DB, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}

func (db *DB) RunMigrations(ctx context.Context) error {
	migrationSQL := `
	-- Create users table
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		username VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		is_admin BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	-- Add is_admin column if it doesn't exist (for existing databases)
	ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT FALSE;
	
	-- Create index for admin lookups
	CREATE INDEX IF NOT EXISTS idx_users_is_admin ON users(is_admin);

	-- Create conversations table
	CREATE TABLE IF NOT EXISTS conversations (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		title VARCHAR(255),
		status VARCHAR(50) DEFAULT 'active',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create messages table
	CREATE TABLE IF NOT EXISTS messages (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
		role VARCHAR(50) NOT NULL,
		content TEXT NOT NULL,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create indexes
	CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
	CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);

	-- Create function to update updated_at timestamp
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = CURRENT_TIMESTAMP;
		RETURN NEW;
	END;
	$$ language 'plpgsql';

	-- Create trigger for conversations
	DROP TRIGGER IF EXISTS update_conversations_updated_at ON conversations;
	CREATE TRIGGER update_conversations_updated_at
		BEFORE UPDATE ON conversations
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();

	-- Create voices table
	CREATE TABLE IF NOT EXISTS voices (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(255) NOT NULL,
		description TEXT,
		prompt TEXT NOT NULL,
		voice_description TEXT,
		hume_voice_id VARCHAR(255),
		hume_config_id VARCHAR(255),
		evi_version VARCHAR(10) DEFAULT '3',
		language_model_provider VARCHAR(50) DEFAULT 'ANTHROPIC',
		language_model_resource VARCHAR(100) DEFAULT 'claude-3-7-sonnet-latest',
		temperature DECIMAL(3,2) DEFAULT 1.0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Create index for voices
	CREATE INDEX IF NOT EXISTS idx_voices_name ON voices(name);
	CREATE INDEX IF NOT EXISTS idx_voices_hume_config_id ON voices(hume_config_id);

	-- Create trigger for voices updated_at
	DROP TRIGGER IF EXISTS update_voices_updated_at ON voices;
	CREATE TRIGGER update_voices_updated_at
		BEFORE UPDATE ON voices
		FOR EACH ROW
		EXECUTE FUNCTION update_updated_at_column();
	`

	_, err := db.Pool.Exec(ctx, migrationSQL)
	return err
}
