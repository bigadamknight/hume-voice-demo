# Hume EVI CLI - Usage Guide

## Quick Start

### Start a new conversation
```bash
python3 cli.py start
```

### Resume last conversation
```bash
python3 cli.py resume
```

### List all conversations
```bash
python3 cli.py list
```

### View transcript
```bash
python3 cli.py view <conversation_id>
```

## Common Issues & Solutions

### WebSocket Connection Drops

**Error:** `ConnectionClosedError: no close frame received or sent`

**Cause:** EVI has a **30-minute session limit**

**What happens:**
- After 30 minutes, EVI automatically closes the connection
- Your conversation is automatically saved to the database
- You can resume with `python3 cli.py resume`

**For longer sessions:**
- End your conversation before the 30-minute mark
- Resume with `python3 cli.py resume` to continue in a new session
- All your conversation history is preserved

### Audio Feedback / Self-Interruption

**Symptoms:** 
- EVI's voice is transcribed as "You"
- Conversation loops with EVI responding to itself

**Solution:**
Make sure `ALLOW_INTERRUPT=false` in your `.env` file:
```bash
grep ALLOW_INTERRUPT .env
# Should show: ALLOW_INTERRUPT=false
```

### Interruption Support

**Default behavior:** Walkie-talkie mode
- Microphone mutes while EVI speaks
- No feedback loops
- Wait for EVI to finish before speaking

**To enable interruption:**
1. Use headphones or earbuds
2. Set in `.env`: `ALLOW_INTERRUPT=true`
3. Restart the app

### Voice Changes During Conversation

EVI may change voices mid-conversation due to:
- EVI configuration settings
- Emotional tone adaptation
- Multiple voices in your EVI config

**Fix:** Review your EVI configuration at https://platform.hume.ai/

### Exiting Conversations

**Ctrl+C** - End conversation gracefully
- Conversation marked as "paused"
- Can resume later with `python3 cli.py resume`

## Best Practices

1. **For body doubling:** Keep conversations focused on one topic at a time
2. **For long sessions:** End and resume every 30-45 minutes
3. **Audio quality:** Use headphones for best experience
4. **Network:** Stable internet connection recommended

## Troubleshooting

### Check your setup
```bash
python3 verify_setup.py
```

### View conversation history
```bash
python3 cli.py list
```

### Check last conversation messages
```bash
sqlite3 conversations.db "SELECT role, substr(content, 1, 50), timestamp FROM messages WHERE conversation_id = (SELECT MAX(id) FROM conversations) ORDER BY timestamp DESC LIMIT 10;"
```

### Reset if needed
Delete the database to start fresh (backup first!):
```bash
cp conversations.db conversations.db.backup
rm conversations.db
# Next run will create a fresh database
```

