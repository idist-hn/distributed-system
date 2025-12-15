#!/bin/bash

# P2P Peer Deployment Script for Real Servers

set -e

TRACKER_URL="https://p2p.idist.dev"
API_KEY="peer-key-001"
BINARY_PATH="bin/peer-linux-amd64"

# Server configurations
SERVER1_IP="171.244.199.139"
SERVER1_PORT="2202"
SERVER1_USER="root"
SERVER1_PASS='Vtdc@Vtdc0'

SERVER2_IP="171.244.195.165"
SERVER2_PORT="22"
SERVER2_USER="root"
SERVER2_PASS='Vtdc@2025'

deploy_to_server() {
    local SERVER_IP=$1
    local SERVER_PORT=$2
    local SERVER_USER=$3
    local SERVER_PASS=$4
    local PEER_NAME=$5
    local P2P_PORT=$6

    echo "=========================================="
    echo "Deploying to $SERVER_IP:$SERVER_PORT"
    echo "=========================================="

    # Upload binary
    echo ">>> Uploading peer binary..."
    sshpass -p "$SERVER_PASS" scp -o StrictHostKeyChecking=no -P $SERVER_PORT \
        $BINARY_PATH $SERVER_USER@$SERVER_IP:/root/peer

    # Create start script and service
    echo ">>> Configuring peer service..."
    sshpass -p "$SERVER_PASS" ssh -o StrictHostKeyChecking=no -p $SERVER_PORT \
        $SERVER_USER@$SERVER_IP "chmod +x /root/peer && mkdir -p /root/p2p-data && pkill -f '/root/peer' || true"

    # Create systemd service file
    sshpass -p "$SERVER_PASS" ssh -o StrictHostKeyChecking=no -p $SERVER_PORT \
        $SERVER_USER@$SERVER_IP "cat > /etc/systemd/system/p2p-peer.service << 'EOF'
[Unit]
Description=P2P Peer Node
After=network.target

[Service]
Type=simple
ExecStart=/root/peer -tracker ${TRACKER_URL} -api-key ${API_KEY} -port ${P2P_PORT} -data /root/p2p-data -daemon
Restart=always
RestartSec=5
Environment=HOME=/root

[Install]
WantedBy=multi-user.target
EOF"

    # Reload and start service
    echo ">>> Starting peer service..."
    sshpass -p "$SERVER_PASS" ssh -o StrictHostKeyChecking=no -p $SERVER_PORT \
        $SERVER_USER@$SERVER_IP "systemctl daemon-reload && systemctl enable p2p-peer && systemctl restart p2p-peer"

    # Check status
    sleep 3
    echo ">>> Checking service status..."
    sshpass -p "$SERVER_PASS" ssh -o StrictHostKeyChecking=no -p $SERVER_PORT \
        $SERVER_USER@$SERVER_IP "systemctl status p2p-peer --no-pager || journalctl -u p2p-peer -n 20 --no-pager"

    echo ">>> Deployment to $SERVER_IP completed!"
    echo ""
}

# Check if sshpass is available
if ! command -v sshpass &> /dev/null; then
    echo "sshpass is required. Install with: brew install hudochenkov/sshpass/sshpass"
    exit 1
fi

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo "Building peer binary for Linux..."
    GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $BINARY_PATH ./services/peer/cmd
fi

echo "Starting deployment to servers..."
echo ""

# Deploy to Server 1
deploy_to_server $SERVER1_IP $SERVER1_PORT $SERVER1_USER "$SERVER1_PASS" "server-01" "6881"

# Deploy to Server 2
deploy_to_server $SERVER2_IP $SERVER2_PORT $SERVER2_USER "$SERVER2_PASS" "server-02" "6881"

echo "=========================================="
echo "All deployments completed!"
echo "=========================================="

