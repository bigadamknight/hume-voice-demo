"""Hume EVI client wrapper for voice conversations."""
import base64
import logging
from typing import Optional, Callable
from hume import AsyncHumeClient, Stream
from hume.empathic_voice.chat.socket_client import ChatConnectOptions
from hume.empathic_voice.chat.types import SubscribeEvent

logger = logging.getLogger(__name__)


class EVIClient:
    """Wrapper around Hume's EVI WebSocket client."""
    
    def __init__(self, api_key: str, config_id: str):
        """Initialize EVI client with API key and config ID.
        
        Args:
            api_key: Hume API key (required)
            config_id: Hume EVI configuration ID (required)
        """
        if not api_key:
            raise ValueError("api_key is required")
        if not config_id:
            raise ValueError("config_id is required")
        
        self.api_key = api_key
        self.config_id = config_id
        self.client = AsyncHumeClient(api_key=self.api_key)
        self.message_callbacks = []
        self.audio_stream: Optional[Stream] = None
    
    def on_message(self, callback: Callable[[str, str], None]):
        """Register callback for incoming messages (role, content)."""
        self.message_callbacks.append(callback)
    
    def set_audio_stream(self, stream: Stream):
        """Set the audio stream for playback."""
        self.audio_stream = stream
    
    async def _handle_message(self, message: SubscribeEvent):
        """Handle incoming messages from EVI."""
        logger.debug(f"Received message type: {message.type}")
        
        try:
            # Handle audio output for playback
            if message.type == "audio_output":
                if self.audio_stream:
                    # Decode base64 audio and queue for playback
                    await self.audio_stream.put(
                        base64.b64decode(message.data.encode("utf-8"))
                    )
            # Handle user interruption - drain audio queue
            elif message.type == "user_interruption":
                if self.audio_stream:
                    # Drain the audio queue to stop playback
                    try:
                        while not self.audio_stream.queue.empty():
                            self.audio_stream.queue.get_nowait()
                    except Exception:
                        pass  # Queue already empty
                # Notify via callback
                for callback in self.message_callbacks:
                    callback("system", "[Interrupted]")
            # Handle text messages
            elif message.type == "user_message":
                content = message.message.content if hasattr(message.message, 'content') else str(message)
                logger.info(f"User message: {content[:100]}")
                for callback in self.message_callbacks:
                    callback("user", content)
            elif message.type == "assistant_message":
                content = message.message.content if hasattr(message.message, 'content') else str(message)
                logger.info(f"Assistant message: {content[:100]}")
                for callback in self.message_callbacks:
                    callback("assistant", content)
            # Log other message types for debugging
            else:
                logger.debug(f"Unhandled message type: {message.type}")
                for callback in self.message_callbacks:
                    callback("system", f"[{message.type}]")
        except Exception as e:
            logger.error(f"Error handling message: {e}", exc_info=True)
    
    def get_connection(self):
        """Get the async context manager for EVI connection with callbacks."""
        def on_error(err):
            """Handle WebSocket errors."""
            error_str = str(err)
            if "no close frame" not in error_str:
                logger.warning(f"WebSocket error: {err}")
        
        return self.client.empathic_voice.chat.connect_with_callbacks(
            options=ChatConnectOptions(config_id=self.config_id),
            on_open=lambda: logger.info("WebSocket connection opened"),
            on_message=self._handle_message,
            on_close=lambda: logger.info("WebSocket connection closed"),
            on_error=on_error
        )
