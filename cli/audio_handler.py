"""Audio input/output handling for microphone and speaker."""
import logging
from hume import MicrophoneInterface, Stream

logger = logging.getLogger(__name__)


class AudioHandler:
    """Manages audio input from microphone and output to speakers."""
    
    def __init__(self):
        """Initialize audio handler with extended buffer for smooth playback."""
        # Create stream with buffer to handle interruptions smoothly
        self.stream = Stream.new()
    
    async def start_microphone(self, socket, allow_interrupt: bool = False):
        """Start microphone interface with the EVI socket.
        
        Args:
            socket: The AsyncChatSocketClient from the EVI connection
            allow_interrupt: Allow interrupting EVI while speaking
        
        Note:
            MicrophoneInterface.start() handles:
            1. Recording from microphone and sending to socket
            2. Playing back audio from the byte_stream
            
            Setting allow_user_interrupt=False by default prevents feedback.
            The microphone mutes while EVI speaks, preventing it from picking up
            the speaker output and creating a feedback loop.
            
            To enable interruption (requires headphones), set ALLOW_INTERRUPT=true in .env
        """
        try:
            logger.info(f"Starting microphone (interrupt={allow_interrupt})")
            await MicrophoneInterface.start(
                socket=socket,
                byte_stream=self.stream,
                allow_user_interrupt=allow_interrupt
            )
        except Exception as e:
            logger.error(f"Error starting microphone: {e}", exc_info=True)
            raise
    
    def stop_microphone(self):
        """Stop microphone capture."""
        try:
            logger.info("Stopping microphone")
            # MicrophoneInterface handles cleanup internally
            # The stream will be closed when the context manager exits
        except Exception as e:
            logger.error(f"Error stopping microphone: {e}", exc_info=True)
