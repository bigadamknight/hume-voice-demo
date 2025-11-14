# Hume EVI Web Application

Web version of the Hume EVI voice conversation application with team access, authentication, and conversation management.

## Architecture

- **Backend**: Go monolith with REST API and WebSocket proxy to Hume EVI
- **Frontend**: React + TypeScript + shadcn/ui
- **Database**: PostgreSQL 16
- **Knowledge Graph**: Memgraph (optional)
- **Deployment**: Docker Compose with Nginx reverse proxy

## Setup

### Prerequisites

- Docker and Docker Compose
- Hume API key and Config ID

### Configuration

1. Create a `.env` file in the `web/` directory with your credentials:

```bash
# Required
HUME_API_KEY=your_api_key_here
HUME_CONFIG_ID=your_config_id_here
JWT_SECRET=your_random_secret_here
ADMIN_USERNAME=admin
ADMIN_PASSWORD=your_secure_password_here

# Optional (defaults work with docker-compose)
DATABASE_URL=postgresql://hume:hume@db:5432/hume_evi?sslmode=disable
MEMGRAPH_URI=bolt://memgraph:7687
CORS_ORIGIN=*
PORT=8080
```

**Note**: The admin user is automatically created on first startup. You can create additional users through the admin UI after logging in.

### Running with Docker

```bash
cd web
docker-compose up -d
```

The application will be available at:
- Frontend: http://localhost
- Backend API: http://localhost:8081
- Memgraph: bolt://localhost:7688

View logs:
```bash
docker-compose logs -f
```

Stop services:
```bash
docker-compose down
```

### Development

#### Backend

```bash
cd web/backend
go mod download
go run cmd/server/main.go
```

The backend will run on port 8080.

#### Frontend

```bash
cd web/frontend
yarn install
yarn dev
```

The frontend will run on port 5173 (Vite default).

## Features

- Simple username/password authentication
- Admin-only user management (create, edit, delete users)
- Conversation management (create, list, delete)
- Real-time voice chat with Hume EVI
- Echo cancellation via browser Web Audio API
- Conversation transcripts
- User-scoped data isolation
- Admin-only voice configuration management

## Deployment

### Production Deployment

1. **Prepare environment**:
   ```bash
   cd web
   # Create .env file with production values
   # Ensure JWT_SECRET is a strong random string
   # Set secure ADMIN_PASSWORD
   ```

2. **Build and start services**:
   ```bash
   docker-compose up -d --build
   ```

3. **Configure reverse proxy** (for SSL/HTTPS):
   
   The application runs on port 80 (HTTP). For production, configure Nginx or Caddy as a reverse proxy with SSL:
   
   **Nginx example**:
   ```nginx
   server {
       listen 443 ssl http2;
       server_name your-domain.com;
       
       ssl_certificate /path/to/cert.pem;
       ssl_certificate_key /path/to/key.pem;
       
       location / {
           proxy_pass http://localhost:80;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }
   }
   ```
   
   **Caddy example** (automatic SSL):
   ```
   your-domain.com {
       reverse_proxy localhost:80
   }
   ```

4. **Health checks**:
   - Frontend: `curl http://localhost/`
   - Backend: `curl http://localhost:8081/api/auth/me` (requires auth)
   - Database: `docker-compose exec db pg_isready -U hume`
   - Memgraph: `docker-compose exec memgraph mgconsole --execute "RETURN 1"`

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `HUME_API_KEY` | Yes | - | Hume API key |
| `HUME_CONFIG_ID` | Yes | - | Hume EVI configuration ID |
| `JWT_SECRET` | Yes | - | Secret for JWT token signing |
| `ADMIN_USERNAME` | Yes | `admin` | Admin username |
| `ADMIN_PASSWORD` | Yes | - | Admin password |
| `DATABASE_URL` | No | `postgresql://hume:hume@db:5432/hume_evi?sslmode=disable` | PostgreSQL connection string |
| `MEMGRAPH_URI` | No | `bolt://memgraph:7687` | Memgraph connection URI |
| `CORS_ORIGIN` | No | `*` | CORS allowed origin |
| `PORT` | No | `8080` | Backend port |

### Building Images

Build individual images:
```bash
# Backend
docker build -f Dockerfile.backend -t hume-evi-backend .

# Frontend
docker build -f Dockerfile.frontend -t hume-evi-frontend .
```

