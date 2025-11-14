# David Bowie - Hume EVI Voice Conversation Platform

A voice conversation platform using Hume's Empathic Voice Interface (EVI), available as both a CLI application and a web application.

## Project Structure

- **`cli/`** - Python CLI application for voice conversations
- **`web/`** - Go backend + React frontend web application
- **`docs/`** - Documentation for integrations and features

## Quick Start

### CLI Application

See [`cli/README.md`](cli/README.md) for CLI setup and usage.

```bash
cd cli
pip install -r requirements.txt
python cli.py start
```

### Web Application

See [`web/README.md`](web/README.md) for web application setup and deployment.

```bash
cd web
docker-compose up -d
```

## Features

- **Real-time voice conversations** with Hume EVI3
- **Conversation persistence** - SQLite (CLI) or PostgreSQL (web)
- **Multi-user support** - Web application with authentication
- **Knowledge graph integration** - Memgraph for pattern recognition
- **Resume conversations** - Pick up where you left off
- **View transcripts** - Read full conversation history

## Architecture

### CLI Application (`cli/`)
- Python 3.9+ application
- SQLite database
- Direct Hume EVI WebSocket connection
- Commands: `start`, `resume`, `list`, `view`

### Web Application (`web/`)
- **Backend**: Go monolith with REST API and WebSocket proxy
- **Frontend**: React + TypeScript + shadcn/ui
- **Database**: PostgreSQL 16
- **Knowledge Graph**: Memgraph (optional)
- **Deployment**: Docker Compose with Nginx reverse proxy

## Documentation

- [`cli/README.md`](cli/README.md) - CLI application documentation
- [`web/README.md`](web/README.md) - Web application documentation
- [`docs/MEMGRAPH_INTEGRATION.md`](docs/MEMGRAPH_INTEGRATION.md) - Memgraph setup and usage
- [`docs/GRAPH_EXTRACTION_EXAMPLE.md`](docs/GRAPH_EXTRACTION_EXAMPLE.md) - Knowledge graph examples

## Requirements

### CLI
- Python 3.9+
- Hume API key and Config ID

### Web Application
- Docker and Docker Compose
- Hume API key and Config ID

## Getting Your Hume Credentials

1. **API Key**: Get from [Hume Platform](https://platform.hume.ai/) → Settings → API Keys
2. **Config ID**: Create an EVI configuration at [Hume Platform](https://platform.hume.ai/) → EVI → Create Configuration

## License

See individual component READMEs for details.

