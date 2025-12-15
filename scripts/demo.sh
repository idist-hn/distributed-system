#!/bin/bash

# P2P File Sharing Demo Script
# This script demonstrates the P2P file sharing system

set -e

echo "=========================================="
echo "   P2P File Sharing System - Demo"
echo "=========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Create test directories
DEMO_DIR="./demo_data"
PEER1_DIR="$DEMO_DIR/peer1"
PEER2_DIR="$DEMO_DIR/peer2"
SHARED_DIR="$DEMO_DIR/shared"

echo -e "${YELLOW}[1/6] Setting up demo directories...${NC}"
rm -rf $DEMO_DIR
mkdir -p $PEER1_DIR $PEER2_DIR $SHARED_DIR

# Create a test file to share
echo -e "${YELLOW}[2/6] Creating test file...${NC}"
dd if=/dev/urandom of=$SHARED_DIR/testfile.bin bs=1024 count=512 2>/dev/null
echo -e "${GREEN}Created testfile.bin (512 KB)${NC}"

# Build the project
echo -e "${YELLOW}[3/6] Building project...${NC}"
go build -o bin/tracker ./services/tracker/cmd
go build -o bin/peer ./services/peer/cmd
echo -e "${GREEN}Build complete!${NC}"

# Start the tracker
echo -e "${YELLOW}[4/6] Starting Tracker Server...${NC}"
./bin/tracker -addr :8080 &
TRACKER_PID=$!
sleep 1
echo -e "${GREEN}Tracker started (PID: $TRACKER_PID)${NC}"

# Function to cleanup on exit
cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up...${NC}"
    kill $TRACKER_PID 2>/dev/null || true
    kill $PEER1_PID 2>/dev/null || true
    kill $PEER2_PID 2>/dev/null || true
    echo -e "${GREEN}Demo cleanup complete${NC}"
}
trap cleanup EXIT

# Instructions for manual demo
echo ""
echo "=========================================="
echo "   Demo Setup Complete!"
echo "=========================================="
echo ""
echo -e "${GREEN}Tracker is running at http://localhost:8080${NC}"
echo ""
echo "To demo the system, open TWO new terminal windows:"
echo ""
echo -e "${YELLOW}Terminal 1 (Seeder):${NC}"
echo "  cd $(pwd)"
echo "  ./bin/peer -port 6881 -data $PEER1_DIR -tracker http://localhost:8080"
echo "  > share $SHARED_DIR/testfile.bin"
echo ""
echo -e "${YELLOW}Terminal 2 (Leecher):${NC}"
echo "  cd $(pwd)"
echo "  ./bin/peer -port 6882 -data $PEER2_DIR -tracker http://localhost:8080"
echo "  > list"
echo "  > download <file_hash>  # Use hash from list command"
echo ""
echo -e "${YELLOW}API Endpoints:${NC}"
echo "  GET  http://localhost:8080/health           - Health check"
echo "  GET  http://localhost:8080/api/files        - List all files"
echo "  GET  http://localhost:8080/api/files/{hash}/peers - Get peers for file"
echo ""
echo -e "${YELLOW}Press Ctrl+C to stop the demo${NC}"
echo ""

# Keep running until Ctrl+C
wait $TRACKER_PID

