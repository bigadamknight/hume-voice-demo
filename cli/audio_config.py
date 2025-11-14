"""Audio configuration utilities for enabling echo cancellation."""
import logging
import sounddevice as sd
import platform

logger = logging.getLogger(__name__)


def configure_echo_cancellation():
    """
    Attempt to enable echo cancellation on macOS.
    
    Note: PortAudio (used by sounddevice) doesn't directly expose macOS's
    Voice Processing I/O unit which provides echo cancellation.
    
    Workarounds:
    1. Use allow_user_interrupt=False (microphone mutes during playback)
    2. Use headphones (physical isolation)
    3. Switch to a different audio backend that supports Voice Processing
    
    Returns:
        dict: Configuration information or None
    """
    if platform.system() == "Darwin":  # macOS
        # Check available devices
        devices = sd.query_devices()
        logger.info("Available audio devices:")
        for i, device in enumerate(devices):
            if device['max_input_channels'] > 0:
                logger.info(f"  Input {i}: {device['name']}")
            if device['max_output_channels'] > 0:
                logger.info(f"  Output {i}: {device['name']}")
        
        # Note: PortAudio on macOS doesn't expose Voice Processing controls
        logger.info("Echo cancellation via Voice Processing I/O is not available through PortAudio")
        logger.info("Recommendations: Use ALLOW_INTERRUPT=false (default) or use headphones")
    
    return None


def list_audio_devices():
    """List all available audio devices.
    
    Returns:
        list: List of audio device dictionaries
    """
    devices = sd.query_devices()
    logger.info("Audio Devices:")
    result = []
    for i, device in enumerate(devices):
        device_info = {"index": i, "device": device}
        if device['max_input_channels'] > 0:
            default = i == sd.default.device[0]
            device_info["type"] = "input"
            device_info["default"] = default
            logger.info(f"[{i}] INPUT:  {device['name']}{' (DEFAULT)' if default else ''}")
        if device['max_output_channels'] > 0:
            default = i == sd.default.device[1]
            device_info["type"] = "output"
            device_info["default"] = default
            logger.info(f"[{i}] OUTPUT: {device['name']}{' (DEFAULT)' if default else ''}")
        result.append(device_info)
    return result


if __name__ == "__main__":
    import sys
    logging.basicConfig(level=logging.INFO)
    list_audio_devices()
    configure_echo_cancellation()

