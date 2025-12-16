# Production Hardening (Phase 4)

## Tổng Quan

Phase 4 tập trung vào việc chuẩn bị hệ thống cho môi trường production với các tính năng:
- PostgreSQL persistent storage
- API Key Authentication
- JWT Token Authentication
- Rate Limiting
- Prometheus Metrics
- Grafana Dashboard
- Health Check Endpoints

## Live System

| Endpoint | URL |
|----------|-----|
| Dashboard | https://p2p.idist.dev/dashboard |
| Health | https://p2p.idist.dev/health |
| Metrics | https://p2p.idist.dev/metrics |
| API | https://p2p.idist.dev/api/* |

## 1. PostgreSQL Storage

### Mô tả
Thay thế in-memory storage bằng PostgreSQL để dữ liệu không mất khi restart.

### Schema
```sql
-- Peers table
CREATE TABLE IF NOT EXISTS peers (
    id TEXT PRIMARY KEY,
    hostname TEXT NOT NULL,
    ip TEXT NOT NULL,
    port INTEGER NOT NULL,
    public_key TEXT,
    status TEXT DEFAULT 'online',
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Files table
CREATE TABLE IF NOT EXISTS files (
    hash TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    size BIGINT NOT NULL,
    chunk_size INTEGER DEFAULT 262144,
    total_chunks INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- File-Peer relationship
CREATE TABLE IF NOT EXISTS file_peers (
    file_hash TEXT REFERENCES files(hash) ON DELETE CASCADE,
    peer_id TEXT REFERENCES peers(id) ON DELETE CASCADE,
    is_seeder BOOLEAN DEFAULT false,
    PRIMARY KEY (file_hash, peer_id)
);
```

### Cấu hình
```yaml
# Environment variable
POSTGRES_URL: "postgres://user:pass@host:5432/dbname?sslmode=disable"
```

### Package
- `services/tracker/internal/storage/postgres.go`

## 2. JWT Authentication

### Mô tả
Token-based authentication cho peers với role-based access control.

### Endpoints
```
POST /api/auth/login
Request:
{
  "peer_id": "peer-001",
  "hostname": "server01"
}
Headers:
  X-API-Key: your-api-key

Response:
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": "2025-12-16T18:00:00Z"
}
```

### Token Claims
```go
type Claims struct {
    PeerID   string `json:"peer_id"`
    Role     string `json:"role"`
    Hostname string `json:"hostname"`
    jwt.RegisteredClaims
}
```

### Roles
- `admin`: Full access
- `peer`: Standard peer operations
- `guest`: Read-only access

### Package
- `services/tracker/internal/api/jwt.go`

## 3. Rate Limiting

### Mô tả
Token bucket rate limiting để prevent API abuse.

### Cấu hình
```go
type ServerConfig struct {
    RateLimitRPS   float64 // Requests per second (default: 100)
    RateLimitBurst int     // Burst size (default: 200)
}
```

### Per-Endpoint Limits
```go
endpointLimits := map[string]EndpointLimit{
    "/api/peers/register":  {RPS: 10, Burst: 20},
    "/api/files/announce":  {RPS: 50, Burst: 100},
    "/api/peers/heartbeat": {RPS: 100, Burst: 200},
}
```

### Response khi bị rate limit
```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests, please try again later"
}
```
HTTP Status: 429 Too Many Requests

### Package
- `services/tracker/internal/api/ratelimit.go`

## 4. Prometheus Metrics

### Mô tả
Export metrics cho monitoring với Prometheus.

### Endpoint
```
GET /metrics
```

### Metrics Available
| Metric | Type | Description |
|--------|------|-------------|
| `p2p_tracker_http_requests_total` | Counter | Total HTTP requests |
| `p2p_tracker_http_request_duration_seconds` | Histogram | Request latency |
| `p2p_tracker_peers_online` | Gauge | Online peers count |
| `p2p_tracker_peers_total` | Gauge | Total registered peers |
| `p2p_tracker_files_shared` | Gauge | Files being shared |
| `p2p_tracker_peer_registrations_total` | Counter | Peer registrations |
| `p2p_tracker_file_announcements_total` | Counter | File announcements |
| `p2p_tracker_ws_connections_active` | Gauge | Active WebSocket connections |
| `p2p_tracker_relay_connections_active` | Gauge | Active relay connections |
| `p2p_tracker_auth_failures_total` | Counter | Auth failures |
| `p2p_tracker_rate_limit_hits_total` | Counter | Rate limit hits |

### Package
- `services/tracker/internal/api/prometheus.go`

## 5. Grafana Dashboard

### Mô tả
Pre-configured Grafana dashboard cho visualization.

### File
- `k8s/grafana-dashboard.json`

### Panels
1. HTTP Requests Rate
2. Request Latency (p50, p95, p99)
3. Online Peers
4. Files Shared
5. WebSocket Connections
6. Relay Connections
7. Auth Failures
8. Rate Limit Hits

## Deployment

### Kubernetes
```bash
# Apply PostgreSQL StatefulSet
kubectl apply -f k8s/tracker-deployment.yaml

# Verify
kubectl get pods -n p2p-system
kubectl logs -n p2p-system deployment/tracker
```

### Environment Variables
```yaml
env:
  - name: POSTGRES_URL
    valueFrom:
      secretKeyRef:
        name: postgres-secret
        key: postgres-url
  - name: JWT_SECRET
    valueFrom:
      secretKeyRef:
        name: tracker-secret
        key: jwt-secret
  - name: API_KEYS
    valueFrom:
      secretKeyRef:
        name: tracker-secret
        key: api-keys
```

## 6. API Key Authentication

### Mô tả
Simple API key authentication cho tất cả API endpoints.

### Cấu hình
```bash
# Environment variable - comma-separated keys
API_KEYS=peer-key-001,peer-key-002,admin-key-001
```

### Usage
```bash
curl -H "X-API-Key: peer-key-001" https://p2p.idist.dev/api/files
```

### Public Endpoints (không cần auth)
- `/health` - Health check
- `/dashboard` - Web UI
- `/metrics` - Prometheus metrics
- `/ws` - WebSocket (realtime updates)
- `/` - Root redirect

### Package
- `services/tracker/internal/api/middleware.go`

## 7. Health Check

### Endpoints

```bash
# Simple health check
GET /health
Response: OK

# Detailed health check
GET /health/detailed
Response:
{
  "status": "healthy",
  "database": "connected",
  "peers_online": 15,
  "peers_total": 42,
  "files_count": 128,
  "version": "1.3.0"
}
```

### Kubernetes Probes
```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

## Testing

```bash
# Health check
curl -k https://p2p.idist.dev/health

# Detailed health
curl -k https://p2p.idist.dev/health/detailed

# Metrics
curl -k https://p2p.idist.dev/metrics

# API with auth
curl -k -H "X-API-Key: peer-key-001" https://p2p.idist.dev/api/files

# Register peer
curl -k -X POST -H "Content-Type: application/json" \
  -H "X-API-Key: peer-key-001" \
  -d '{"peer_id":"test-peer","ip":"1.2.3.4","port":6881}' \
  https://p2p.idist.dev/api/peers/register

# Get JWT token
curl -k -X POST -H "Content-Type: application/json" \
  -H "X-API-Key: peer-key-001" \
  -d '{"peer_id":"test-peer","hostname":"myhost"}' \
  https://p2p.idist.dev/api/auth/login
```

## Security Checklist

| Item | Status | Notes |
|------|--------|-------|
| API Key Authentication | ✅ | Required for /api/* endpoints |
| Rate Limiting | ✅ | 100 req/min per IP |
| HTTPS Only | ✅ | TLS via Ingress |
| JWT Tokens | ✅ | For advanced auth flows |
| Input Validation | ✅ | JSON schema validation |
| SQL Injection Prevention | ✅ | Parameterized queries |
| CORS Headers | ✅ | Configured in middleware |

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         INGRESS (HTTPS)                         │
│                     p2p.idist.dev:443                           │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      TRACKER SERVICE                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │Prometheus│ │   Rate   │ │   Auth   │ │ Handlers │          │
│  │Middleware│▶│  Limit   │▶│Middleware│▶│          │          │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘          │
│                                               │                 │
│                         ┌─────────────────────┼─────────────┐  │
│                         ▼                     ▼             │  │
│                  ┌──────────────┐     ┌──────────────┐      │  │
│                  │  PostgreSQL  │     │   WSHub      │      │  │
│                  │   Storage    │     │  (Realtime)  │      │  │
│                  └──────────────┘     └──────────────┘      │  │
└─────────────────────────────────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                      POSTGRESQL POD                             │
│                    (StatefulSet)                                │
└─────────────────────────────────────────────────────────────────┘
```

