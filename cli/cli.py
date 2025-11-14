"""CLI interface for Hume EVI voice conversations."""
import asyncio
import logging
import os
from typing import Optional
import click
from rich.console import Console

# Enable logging for debugging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(levelname)s: %(message)s')

from conversation_manager import ConversationManager
from config import get_config

console = Console()


@click.group()
def cli():
    """Hume EVI CLI - Voice conversations with body doubling support."""
    pass


@cli.command()
def start():
    """Start a new conversation."""
    # Set SSL certificate file for macOS
    import certifi
    config = get_config()
    if not config.ssl_cert_file:
        os.environ['SSL_CERT_FILE'] = certifi.where()
    
    manager = ConversationManager(config=config)
    asyncio.run(manager.start_conversation())


@cli.command()
@click.argument("conversation_id", type=int, required=False)
def resume(conversation_id: Optional[int]):
    """Resume a conversation. If no ID provided, resumes last active conversation."""
    # Set SSL certificate file for macOS
    import certifi
    config = get_config()
    if not config.ssl_cert_file:
        os.environ['SSL_CERT_FILE'] = certifi.where()
    
    manager = ConversationManager(config=config)
    
    if not conversation_id:
        last_conv = manager.db.get_last_active_conversation()
        if not last_conv:
            console.print("[yellow]No active conversation found. Starting new conversation...[/yellow]")
            asyncio.run(manager.start_conversation())
        else:
            conversation_id = last_conv["id"]
            console.print(f"[blue]Resuming conversation {conversation_id}...[/blue]")
            asyncio.run(manager.start_conversation(conversation_id))
    else:
        asyncio.run(manager.start_conversation(conversation_id))


@cli.command()
def list():
    """List all conversations."""
    config = get_config()
    manager = ConversationManager(config=config)
    manager.list_conversations()


@cli.command()
@click.argument("conversation_id", type=int)
def view(conversation_id: int):
    """View transcript of a conversation."""
    config = get_config()
    manager = ConversationManager(config=config)
    manager.view_transcript(conversation_id)


if __name__ == "__main__":
    cli()
