# AwoChat

**Deterministic Real-Time Messaging** – A minimal, single-node real-time group chat system engineered for constrained infrastructure (1 CPU, 1GB RAM VPS).

## Quick Start (Windows)

```bash
# 1. Run the full setup script
d:\Projects\AwoChat\scripts\setup-all.bat

# 2. Start development servers
d:\Projects\AwoChat\scripts\start-dev.bat

# 3. Open browser
http://localhost:3000
```

For detailed setup instructions, see [SETUP.md](./SETUP.md).

## Production Deployment

**Domain:** https://awochat.indraprhmbd.my.id  
**Repository:** https://github.com/indraprhmbd/AwoChat  
**Author:** [@indraprhmbd](https://github.com/indraprhmbd)

See [DEPLOYMENT.md](./DEPLOYMENT.md) for VPS deployment guide.

## Features

- ✅ Email + password authentication (session-based, no JWT)
- ✅ Private rooms via invite links
- ✅ Real-time messaging via WebSocket
- ✅ Typing indicators (ephemeral, TTL-based)
- ✅ Message persistence with pagination
- ✅ Role-based access (admin/member)
- ✅ Rate limiting (signup, login, messages)
- ✅ Background session cleanup
- ✅ Graceful shutdown

## Non-Goals (Explicitly Excluded)

- Microservices, Redis, message brokers
- File uploads, message edit/delete
- Reactions, read receipts, push notifications
- Horizontal scaling, distributed state

## Tech Stack

| Layer | Technology |
|-------|------------|
| Backend | Go (latest), `net/http`, `gorilla/websocket`, `pgx` |
| Database | PostgreSQL (local) |
| Frontend | React 18 + Vite |
| Deployment | Single binary + systemd (no Docker) |

## Project Structure

```
awochat/
├── backend/
│   ├── cmd/
│   │   ├── main.go              # Application entry point
│   │   └── migrate/
│   │       └── main.go          # Migration CLI tool
│   ├── internal/
│   │   ├── config/              # Configuration management
│   │   ├── database/            # Database layer (pgx)
│   │   ├── handlers/            # HTTP/WebSocket handlers
│   │   ├── models/              # Data models
│   │   ├── middleware/          # HTTP middleware
│   │   ├── ratelimiter/         # In-memory rate limiting
│   │   └── websocket/           # WebSocket manager
│   └── migrations/              # SQL migration files
├── frontend/
│   ├── src/
│   │   ├── api.js               # API client
│   │   ├── App.jsx              # Main app component
│   │   ├── contexts/            # React contexts (Auth)
│   │   ├── hooks/               # Custom hooks (useWebSocket)
│   │   ├── pages/               # Page components
│   │   └── main.jsx             # React entry point
│   ├── index.html
│   ├── package.json
│   └── vite.config.js
├── prompts/                     # Project specification documents
└── README.md
```

## Prerequisites

- Go 1.21+
- Node.js 18+
- PostgreSQL 14+

## Quick Start

### 1. Database Setup

```bash
# Create PostgreSQL database and user
psql -U postgres
CREATE DATABASE awochat;
CREATE USER awochat WITH PASSWORD 'awochat';
GRANT ALL PRIVILEGES ON DATABASE awochat TO awochat;
\q
```

### 2. Backend Setup

```bash
cd backend

# Run migrations
go run cmd/migrate/main.go -command up

# Start server
go run cmd/main.go
```

Server runs on `http://localhost:8080`

### 3. Frontend Setup

```bash
cd frontend

# Install dependencies
npm install

# Start dev server (with proxy to backend)
npm run dev
```

Frontend runs on `http://localhost:3000`

## Configuration

Environment variables (all optional, defaults shown):

```bash
# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=awochat
DB_PASSWORD=awochat
DB_NAME=awochat
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=5

# Session
SESSION_COOKIE_NAME=awochat_session
SESSION_COOKIE_SECURE=false
SESSION_COOKIE_HTTP_ONLY=true
SESSION_COOKIE_SAME_SITE=Strict

# Limits
MAX_ROOM_MEMBERS=100
MAX_MESSAGE_SIZE=2048
MAX_SEND_BUFFER_SIZE=32
SIGNUP_RATE_LIMIT=5
LOGIN_RATE_LIMIT=10
MESSAGE_RATE_LIMIT=10
TYPING_THROTTLE_SEC=1
MAX_ROOMS_PER_USER=50
MAX_SESSIONS_PER_USER=10
```

## API Endpoints

### Authentication

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/auth/signup` | Register new user |
| POST | `/api/auth/login` | Login |
| POST/GET | `/api/auth/logout` | Logout |
| GET | `/api/auth/me` | Get current user |

### Rooms

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/rooms` | Create room |
| GET | `/api/rooms` | List user's rooms |
| GET | `/api/rooms/:id` | Get room details |
| POST | `/api/rooms/join` | Join room via token |
| POST | `/api/rooms/leave` | Leave room |
| GET | `/api/rooms/members?room_id=` | Get room members |

### Messages

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/api/messages?room_id=&limit=&offset=` | Get messages (paginated) |

### WebSocket

| Endpoint | Description |
|----------|-------------|
| `GET /ws?room_id=` | WebSocket upgrade for real-time messaging |

**Message Types:**

```json
// Client → Server: Send message
{"type": "message", "content": "Hello!"}

// Client → Server: Typing indicator
{"type": "typing"}

// Server → Client: New message
{"type": "message", "id": 1, "user_id": "...", "user_email": "user@example.com", "content": "Hello!", "created_at": "..."}

// Server → Client: Typing update
{"type": "typing", "user_id": "..."}

// Server → Client: Error
{"type": "error", "error": "error message"}
```

### Health & Metrics

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/metrics` | Basic metrics (connections, rooms, goroutines) |

## Migration Tool

```bash
# Apply all pending migrations
go run cmd/migrate/main.go -command up

# Apply up to specific version
go run cmd/migrate/main.go -command up -version 3

# Revert last migration
go run cmd/migrate/main.go -command down

# Revert to specific version
go run cmd/migrate/main.go -command down -version 2

# Create new migration
go run cmd/migrate/main.go -command create -name add_indexes
```

## Deployment (VPS)

### 1. Build

```bash
# Backend
cd backend
go build -o awochat ./cmd/main.go

# Frontend
cd frontend
npm run build
```

### 2. Systemd Service

Create `/etc/systemd/system/awochat.service`:

```ini
[Unit]
Description=AwoChat Server
After=network.target postgresql.service

[Service]
Type=simple
User=awochat
WorkingDirectory=/opt/awochat
ExecStart=/opt/awochat/awochat
Restart=on-failure
Environment="DB_HOST=localhost"
Environment="DB_PASSWORD=your_secure_password"

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable awochat
sudo systemctl start awochat
```

### 3. Nginx Reverse Proxy

```nginx
server {
    listen 80;
    server_name awochat.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
    }
}
```

## Load Assumptions

| Metric | Target |
|--------|--------|
| MAU | < 100 |
| Concurrent WebSocket | < 300 |
| Messages/day | ~10,000 |
| Max room size | 100 members |
| Memory budget | < 600MB |

## Security Considerations

- bcrypt password hashing
- Session-based auth (HttpOnly, Secure, SameSite=Strict cookies)
- Rate limiting on auth endpoints and messages
- Parameterized queries (SQL injection prevention)
- Room membership validation on every WebSocket connection
- Max 100 members per room (enforced via DB transaction)

**Explicit Limitations:**

- Single-node architecture (no horizontal scaling)
- No DDoS protection
- No distributed session store
- Not designed for large-scale public exposure

## License

MIT
