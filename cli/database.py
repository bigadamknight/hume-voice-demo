"""SQLite database management for conversations and messages."""
import sqlite3
import logging
from datetime import datetime
from typing import List, Optional, Tuple, Dict
from contextlib import contextmanager

logger = logging.getLogger(__name__)


class Database:
    """Manages SQLite database operations for conversation storage."""
    
    def __init__(self, db_path: str = "conversations.db"):
        """Initialize database connection and create schema if needed."""
        self.db_path = db_path
        self._init_schema()
    
    def _init_schema(self):
        """Create tables if they don't exist."""
        with self._get_connection() as conn:
            cursor = conn.cursor()
            
            # Conversations table
            cursor.execute("""
                CREATE TABLE IF NOT EXISTS conversations (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    title TEXT,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    status TEXT DEFAULT 'active'
                )
            """)
            
            # Messages table
            cursor.execute("""
                CREATE TABLE IF NOT EXISTS messages (
                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                    conversation_id INTEGER NOT NULL,
                    role TEXT NOT NULL,
                    content TEXT NOT NULL,
                    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                    audio_duration REAL,
                    FOREIGN KEY (conversation_id) REFERENCES conversations (id)
                )
            """)
            
            conn.commit()
    
    @contextmanager
    def _get_connection(self):
        """Get database connection with context management."""
        conn = sqlite3.connect(self.db_path, timeout=10.0)
        conn.row_factory = sqlite3.Row
        # Enable foreign key constraints
        conn.execute("PRAGMA foreign_keys = ON")
        try:
            yield conn
        except sqlite3.Error as e:
            logger.error(f"Database error: {e}", exc_info=True)
            conn.rollback()
            raise
        finally:
            conn.close()
    
    def create_conversation(self, title: Optional[str] = None) -> int:
        """Create a new conversation and return its ID."""
        if not title:
            title = f"Conversation {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}"
        
        with self._get_connection() as conn:
            cursor = conn.cursor()
            try:
                cursor.execute("""
                    INSERT INTO conversations (title, status, created_at, updated_at)
                    VALUES (?, 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
                """, (title,))
                conversation_id = cursor.lastrowid
                conn.commit()
                logger.info(f"Created conversation {conversation_id}")
                return conversation_id
            except sqlite3.Error as e:
                logger.error(f"Error creating conversation: {e}", exc_info=True)
                raise
    
    def get_conversation(self, conversation_id: int) -> Optional[Dict]:
        """Get conversation by ID."""
        with self._get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute("""
                SELECT * FROM conversations WHERE id = ?
            """, (conversation_id,))
            row = cursor.fetchone()
            if row:
                return dict(row)
            return None
    
    def get_last_active_conversation(self) -> Optional[Dict]:
        """Get the last active conversation."""
        with self._get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute("""
                SELECT * FROM conversations 
                WHERE status = 'active' 
                ORDER BY updated_at DESC 
                LIMIT 1
            """)
            row = cursor.fetchone()
            if row:
                return dict(row)
            return None
    
    def list_conversations(self, limit: int = 50) -> List[Dict]:
        """List all conversations, most recent first."""
        with self._get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute("""
                SELECT c.*, 
                       COUNT(m.id) as message_count,
                       MAX(m.timestamp) as last_message_time
                FROM conversations c
                LEFT JOIN messages m ON c.id = m.conversation_id
                GROUP BY c.id
                ORDER BY c.updated_at DESC
                LIMIT ?
            """, (limit,))
            return [dict(row) for row in cursor.fetchall()]
    
    def update_conversation_status(self, conversation_id: int, status: str):
        """Update conversation status (active, paused, completed)."""
        with self._get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute("""
                UPDATE conversations 
                SET status = ?, updated_at = CURRENT_TIMESTAMP
                WHERE id = ?
            """, (status, conversation_id))
            conn.commit()
    
    def add_message(self, conversation_id: int, role: str, content: str, 
                   audio_duration: Optional[float] = None) -> int:
        """Add a message to a conversation and return message ID."""
        with self._get_connection() as conn:
            cursor = conn.cursor()
            try:
                cursor.execute("""
                    INSERT INTO messages (conversation_id, role, content, timestamp, audio_duration)
                    VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?)
                """, (conversation_id, role, content, audio_duration))
                message_id = cursor.lastrowid
                
                # Update conversation's updated_at timestamp
                cursor.execute("""
                    UPDATE conversations 
                    SET updated_at = CURRENT_TIMESTAMP
                    WHERE id = ?
                """, (conversation_id,))
                
                conn.commit()
                logger.debug(f"Added message {message_id} to conversation {conversation_id}")
                return message_id
            except sqlite3.Error as e:
                logger.error(f"Error adding message: {e}", exc_info=True)
                raise
    
    def get_messages(self, conversation_id: int) -> List[Dict]:
        """Get all messages for a conversation, ordered by timestamp."""
        with self._get_connection() as conn:
            cursor = conn.cursor()
            cursor.execute("""
                SELECT * FROM messages 
                WHERE conversation_id = ?
                ORDER BY timestamp ASC
            """, (conversation_id,))
            return [dict(row) for row in cursor.fetchall()]
    
    def get_transcript(self, conversation_id: int) -> str:
        """Get formatted transcript for a conversation."""
        messages = self.get_messages(conversation_id)
        if not messages:
            return "No messages in this conversation."
        
        lines = []
        for msg in messages:
            role = msg['role'].upper()
            content = msg['content']
            timestamp = msg['timestamp']
            lines.append(f"[{timestamp}] {role}: {content}")
        
        return "\n".join(lines)
