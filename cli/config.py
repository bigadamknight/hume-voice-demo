"""Centralized configuration management for CLI application."""
import os
from pathlib import Path
from typing import Optional
from dotenv import load_dotenv

# Load environment variables from .env file (look in current dir and parent)
env_path = Path(__file__).parent / ".env"
if env_path.exists():
    load_dotenv(env_path)
else:
    # Try parent directory
    load_dotenv(Path(__file__).parent.parent / ".env")


class Config:
    """Application configuration loaded from environment variables."""
    
    def __init__(self):
        """Initialize configuration and validate required settings."""
        self.hume_api_key = self._get_required_env("HUME_API_KEY")
        self.hume_config_id = self._get_required_env("HUME_CONFIG_ID")
        self.allow_interrupt = os.getenv("ALLOW_INTERRUPT", "false").lower() == "true"
        
        # Database path relative to cli directory
        db_path_env = os.getenv("DB_PATH")
        if db_path_env:
            self.db_path = db_path_env
        else:
            # Default to conversations.db in the cli directory
            cli_dir = Path(__file__).parent
            self.db_path = str(cli_dir / "conversations.db")
        
        # SSL certificate configuration for macOS
        self.ssl_cert_file = os.getenv("SSL_CERT_FILE")
    
    def _get_required_env(self, key: str) -> str:
        """Get required environment variable or raise ValueError."""
        value = os.getenv(key)
        if not value or value == f"your_{key.lower()}_here":
            raise ValueError(
                f"{key} not found or not configured. "
                f"Set it in .env file or environment variable.\n"
                f"For HUME_API_KEY: Get from https://platform.hume.ai/ → Settings → API Keys\n"
                f"For HUME_CONFIG_ID: Create EVI config at https://platform.hume.ai/ → EVI → Create Configuration"
            )
        return value
    
    @property
    def interruption_enabled(self) -> bool:
        """Check if user interruption is enabled."""
        return self.allow_interrupt


def get_config() -> Config:
    """Get application configuration instance."""
    return Config()

