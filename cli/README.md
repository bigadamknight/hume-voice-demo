# Hume EVI CLI Voice Assistant

A simple CLI application for voice conversations with Hume's Empathic Voice Interface (EVI3), designed for body doubling while working on other tasks. Supports conversation history, resuming previous conversations, and viewing transcripts.

## Features

- **Real-time voice conversations** with Hume EVI3
- **Conversation persistence** - SQLite database stores all conversations
- **Resume conversations** - Pick up where you left off
- **View transcripts** - Read full conversation history
- **Background operation** - Runs while you work on other tasks

## Setup

### Prerequisites

- Python 3.9 or later
- Hume API key and Config ID

### Installation

1. Navigate to the CLI directory:

```bash
cd cli
```

2. Create a virtual environment:

```bash
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
```

3. Install dependencies:

```bash
pip install -r requirements.txt
```

4. Configure environment variables:

Create a `.env` file in the `cli/` directory (or in the project root) with your credentials:

```bash
# Required
HUME_API_KEY=your_api_key_here
HUME_CONFIG_ID=your_config_id_here

# Optional
ALLOW_INTERRUPT=false
DB_PATH=conversations.db
```

**Getting your credentials:**
1. **API Key**: Get your API key from [Hume Platform](https://platform.hume.ai/) → Settings → API Keys
2. **Config ID**: Create an EVI configuration at [Hume Platform](https://platform.hume.ai/) → EVI → Create Configuration. The config ID will be shown after creation.

**Note:** You need to create an EVI configuration to get the config ID. This config defines the voice, personality, and behavior of your assistant.

## Usage

### Start a new conversation

```bash
python cli.py start
```

### Resume last active conversation

```bash
python cli.py resume
```

### Resume specific conversation by ID

```bash
python cli.py resume <conversation_id>
```

### List all conversations

```bash
python cli.py list
```

### View transcript of a conversation

```bash
python cli.py view <conversation_id>
```

## Commands

- `start` - Begin a new voice conversation
- `resume [conversation_id]` - Continue an existing conversation (defaults to last active)
- `list` - Show all past conversations with metadata
- `view <conversation_id>` - Display full transcript of a conversation

## Database

Conversations are stored in `conversations.db` (SQLite) with the following structure:

- **conversations** - Conversation metadata (id, title, created_at, updated_at, status)
- **messages** - Individual messages (id, conversation_id, role, content, timestamp)

## Audio Configuration

### Interruption Support

**Interruption is available, but requires headphones to avoid feedback loops.**

**How it works:**
1. EVI has **server-side interruption detection**
2. When you speak while EVI is talking, it detects the interruption
3. Sends a `user_interruption` message
4. Audio playback stops
5. Your speech is processed

**Configuration:**

By default, interruption is **disabled** (`ALLOW_INTERRUPT=false`) to prevent audio feedback loops.

To enable interruption, add to `.env`:
```
ALLOW_INTERRUPT=true
```

**⚠️ Important:** Only enable interruption if using **headphones or earbuds**!

**Why headphones are needed:**
- Without headphones, the microphone picks up the speaker output
- This creates a feedback loop where EVI's voice is transcribed as your speech
- With headphones, the microphone only hears your voice

**Modes:**
- **Default (`false`):** Walkie-talkie mode - mic mutes while EVI speaks, no feedback
- **Enabled (`true`):** Can interrupt anytime - requires headphones to prevent feedback

## Notes

- Press `Ctrl+C` to end a conversation gracefully
- Conversations are automatically saved as you speak
- The microphone is active during conversations - speak naturally to interact with EVI
- Conversations are marked as "paused" when you exit, allowing you to resume later
- **EVI has a 30-minute session limit** - after 30 minutes the connection will close automatically
  - Your conversation is saved, just run `python3 cli.py resume` to continue

## Database Location

The SQLite database (`conversations.db`) is stored in the `cli/` directory by default. You can override this by setting the `DB_PATH` environment variable.
