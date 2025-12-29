# Deployment Guide

## 1. Prerequisites

- Go 1.21+
- Docker & Docker Compose
- kubectl (for Kubernetes)
- PostgreSQL 15+

## 2. Local Development

```bash
# Build all binaries
make build

# Start tracker (with in-memory storage)
./bin/tracker -addr :8080

# Start peer
./bin/peer -port 6881 -data ./data -tracker http://localhost:8080
```

## 3. Docker Deployment

### Build Images

```bash
# Build tracker image
docker build -f docker/tracker.Dockerfile -t p2p-tracker .

# Build peer image
docker build -f docker/peer.Dockerfile -t p2p-peer .
```

### Run with Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: p2p_tracker
      POSTGRES_USER: tracker
      POSTGRES_PASSWORD: secret
    volumes:
      - pgdata:/var/lib/postgresql/data

  tracker:
    image: p2p-tracker
    ports:
      - "8080:8080"
    environment:
      POSTGRES_URL: postgres://tracker:secret@postgres:5432/p2p_tracker?sslmode=disable
      API_KEYS: peer-key-001,peer-key-002
    depends_on:
      - postgres

volumes:
  pgdata:
```

## 4. Kubernetes Deployment

### Apply Manifests

```bash
# Create namespace and apply all resources
kubectl apply -f k8s/

# Check status
kubectl get pods -n p2p-system
kubectl get svc -n p2p-system
```

### Key Resources

| Resource | File | Description |
|----------|------|-------------|
| Namespace | `namespace.yaml` | p2p-system namespace |
| Tracker | `tracker-deployment.yaml` | Tracker deployment + service |
| Ingress | `ingress.yaml` | NGINX ingress with TLS |
| Secrets | `registry-secret.yaml` | Docker registry credentials |

### Environment Variables

```yaml
env:
  - name: POSTGRES_URL
    valueFrom:
      secretKeyRef:
        name: tracker-secrets
        key: postgres-url
  - name: API_KEYS
    value: "peer-key-001,peer-key-002"
  - name: JWT_SECRET
    valueFrom:
      secretKeyRef:
        name: tracker-secrets
        key: jwt-secret
```

## 5. Bare Metal Server Deployment

### Install Peer Binary

```bash
# Copy binary to server
scp bin/peer-linux-amd64 user@server:/opt/p2p/peer

# Create systemd service
sudo cp scripts/p2p-peer.service /etc/systemd/system/

# Start service
sudo systemctl daemon-reload
sudo systemctl enable p2p-peer
sudo systemctl start p2p-peer
```

### Systemd Service File

```ini
# /etc/systemd/system/p2p-peer.service
[Unit]
Description=P2P Peer Node
After=network.target

[Service]
Type=simple
User=p2p
ExecStart=/opt/p2p/peer -tracker https://p2p.idist.dev -api-key peer-key-001 -port 6881 -data /opt/p2p/data -daemon
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

## 6. Production Checklist

- [ ] PostgreSQL with proper backup
- [ ] TLS certificates (Let's Encrypt)
- [ ] API keys configured
- [ ] Rate limiting enabled
- [ ] Monitoring (Prometheus + Grafana)
- [ ] Log aggregation
- [ ] Firewall rules (port 6881 for P2P)

## 7. Monitoring

### Prometheus Metrics

Tracker exposes metrics at `/metrics`:

- `p2p_peers_online` - Current online peers
- `p2p_files_total` - Total files shared
- `p2p_relay_connections` - Active relay connections
- `p2p_requests_total` - Total API requests

### Grafana Dashboard

Import `k8s/grafana-dashboard.json` for pre-built dashboard.

