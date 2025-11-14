"""Conversation manager coordinating EVI, audio, and database operations."""
import asyncio
import signal
import sys
import logging
from typing import Optional
from rich.console import Console

from database import Database
from evi_client import EVIClient
from audio_handler import AudioHandler
from config import get_config

logger = logging.getLogger(__name__)
console = Console()


class ConversationManager:
    """Manages conversations and coordinates between EVI, audio, and database."""
    
    def __init__(self, config=None):
        """Initialize conversation manager with dependencies."""
        self.config = config or get_config()
        self.db = Database(db_path=self.config.db_path)
        self.evi_client: Optional[EVIClient] = None
        self.audio_handler = AudioHandler()
        self.current_conversation_id: Optional[int] = None
        self.running = False
    
    def _handle_exit(self, signum, frame):
        """Handle graceful shutdown on Ctrl+C."""
        console.print("\n[yellow]Shutting down...[/yellow]")
        self.running = False
        # Raise KeyboardInterrupt to break out of the async loop
        raise KeyboardInterrupt()
    
    def _setup_signal_handlers(self):
        """Setup signal handlers for graceful shutdown."""
        signal.signal(signal.SIGINT, self._handle_exit)
        signal.signal(signal.SIGTERM, self._handle_exit)
    
    def _save_message_callback(self, role: str, content: str):
        """Callback to save messages to database."""
        # Don't save system messages like [Interrupted]
        if role != "system" and self.current_conversation_id:
            try:
                self.db.add_message(self.current_conversation_id, role, content)
            except Exception as e:
                logger.error(f"Failed to save message: {e}", exc_info=True)
        
        # Print messages
        if role == "assistant":
            console.print(f"[cyan]EVI:[/cyan] {content}")
        elif role == "user":
            console.print(f"[green]You:[/green] {content}")
        elif role == "system":
            # Show system messages for debugging
            console.print(f"[dim yellow]{content}[/dim yellow]")
    
    async def start_conversation(self, conversation_id: Optional[int] = None):
        """Start or resume a conversation."""
        try:
            # Initialize EVI client
            console.print("[blue]Connecting to Hume EVI...[/blue]")
            self.evi_client = EVIClient(
                api_key=self.config.hume_api_key,
                config_id=self.config.hume_config_id
            )
            
            # Set up conversation
            if conversation_id:
                conv = self.db.get_conversation(conversation_id)
                if not conv:
                    console.print(f"[red]Conversation {conversation_id} not found.[/red]")
                    return
                self.current_conversation_id = conversation_id
                self.db.update_conversation_status(conversation_id, "active")
                console.print(f"[green]Resumed conversation {conversation_id}[/green]")
            else:
                self.current_conversation_id = self.db.create_conversation()
                console.print(f"[green]Started new conversation {self.current_conversation_id}[/green]")
            
            # Register message callback
            self.evi_client.on_message(self._save_message_callback)
            
            # Set the audio stream for playback
            self.evi_client.set_audio_stream(self.audio_handler.stream)
            
            # Use the connection with callbacks
            async with self.evi_client.get_connection() as chat_socket:
                console.print("[green]Connected![/green]")
                console.print("[yellow]Starting microphone... Say something to begin![/yellow]")
                console.print("[dim]Press Ctrl+C to end conversation[/dim]\n")
                
                self.running = True
                self._setup_signal_handlers()
                
                try:
                    # Check if interruption is enabled
                    allow_interrupt = self.config.interruption_enabled
                    console.print(f"[dim]Interruption setting: {allow_interrupt}[/dim]")
                    
                    if allow_interrupt:
                        console.print("[yellow]⚠️  Interruption ENABLED - Use headphones to avoid feedback![/yellow]")
                    else:
                        console.print("[dim]Microphone will mute while EVI speaks (walkie-talkie mode)[/dim]")
                    
                    # MicrophoneInterface.start() handles everything:
                    # 1. Microphone capture and sending audio to socket
                    # 2. Receiving and playing audio responses from EVI (via the stream)
                    # Message callbacks are handled by the EVIClient
                    await self.audio_handler.start_microphone(chat_socket, allow_interrupt)
                except KeyboardInterrupt:
                    # Allow Ctrl+C to break out
                    console.print("\n[yellow]Stopping...[/yellow]")
                    pass
            
        except KeyboardInterrupt:
            console.print("\n[yellow]Interrupted by user[/yellow]")
        except Exception as e:
            error_msg = str(e)
            # Handle WebSocket connection errors gracefully
            if "ConnectionClosedError" in str(type(e)) or "no close frame" in error_msg:
                console.print(f"\n[yellow]Session ended (EVI has a 30-minute limit)[/yellow]")
                console.print(f"[dim]Your conversation has been saved. Resume with:[/dim]")
                console.print(f"[cyan]python3 cli.py resume[/cyan]")
            else:
                console.print(f"[red]Error: {e}[/red]")
                logger.exception("Error during conversation")
        finally:
            await self._cleanup()
    
    async def _cleanup(self):
        """Clean up resources."""
        try:
            if self.current_conversation_id:
                self.db.update_conversation_status(self.current_conversation_id, "paused")
        except Exception as e:
            logger.error(f"Error updating conversation status: {e}", exc_info=True)
        
        try:
            self.audio_handler.stop_microphone()
        except Exception as e:
            logger.error(f"Error stopping microphone: {e}", exc_info=True)
        
        console.print("[blue]Disconnected[/blue]")
    
    def list_conversations(self):
        """List all conversations."""
        try:
            conversations = self.db.list_conversations()
        except Exception as e:
            console.print(f"[red]Error listing conversations: {e}[/red]")
            logger.exception("Error listing conversations")
            return
        
        if not conversations:
            console.print("[yellow]No conversations yet.[/yellow]")
            return
        
        from rich.table import Table
        table = Table(title="Conversations")
        table.add_column("ID", style="cyan")
        table.add_column("Title", style="green")
        table.add_column("Status", style="yellow")
        table.add_column("Messages", justify="right")
        table.add_column("Created", style="dim")
        table.add_column("Updated", style="dim")
        
        for conv in conversations:
            status = conv.get("status", "unknown")
            message_count = conv.get("message_count", 0)
            created = conv.get("created_at", "")[:19] if conv.get("created_at") else ""
            updated = conv.get("updated_at", "")[:19] if conv.get("updated_at") else ""
            
            table.add_row(
                str(conv["id"]),
                conv.get("title", "Untitled")[:50],
                status,
                str(message_count),
                created,
                updated
            )
        
        console.print(table)
    
    def view_transcript(self, conversation_id: int):
        """View transcript of a conversation."""
        try:
            conv = self.db.get_conversation(conversation_id)
        except Exception as e:
            console.print(f"[red]Error fetching conversation: {e}[/red]")
            logger.exception("Error fetching conversation")
            return
        
        if not conv:
            console.print(f"[red]Conversation {conversation_id} not found.[/red]")
            return
        
        try:
            messages = self.db.get_messages(conversation_id)
        except Exception as e:
            console.print(f"[red]Error fetching messages: {e}[/red]")
            logger.exception("Error fetching messages")
            return
        
        if not messages:
            console.print(f"[yellow]No messages in conversation {conversation_id}.[/yellow]")
            return
        
        # Build transcript
        from rich.panel import Panel
        from rich.markdown import Markdown
        
        transcript_lines = []
        transcript_lines.append(f"# Conversation {conversation_id}")
        transcript_lines.append(f"**Title:** {conv.get('title', 'Untitled')}")
        transcript_lines.append(f"**Created:** {conv.get('created_at', '')}")
        transcript_lines.append(f"**Status:** {conv.get('status', 'unknown')}")
        transcript_lines.append("")
        transcript_lines.append("## Messages")
        transcript_lines.append("")
        
        for msg in messages:
            role = msg["role"].upper()
            content = msg["content"]
            timestamp = msg["timestamp"][:19] if msg.get("timestamp") else ""
            transcript_lines.append(f"**[{timestamp}] {role}:** {content}")
            transcript_lines.append("")
        
        transcript = "\n".join(transcript_lines)
        console.print(Panel(Markdown(transcript), title=f"Transcript - Conversation {conversation_id}", expand=False))

