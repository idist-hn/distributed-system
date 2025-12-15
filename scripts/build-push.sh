#!/bin/bash
set -e

# Configuration
REGISTRY="registry.idist.dev"
REGISTRY_USER="idist-hn"
REGISTRY_PASS="HanhHanh2508@"
NAMESPACE="p2p-system"
IMAGE_TAG="${1:-latest}"

echo "========================================"
echo "P2P File Sharing - Build & Push Images"
echo "========================================"

# Login to registry
echo ""
echo "📦 Logging in to registry..."
echo "$REGISTRY_PASS" | docker login "$REGISTRY" -u "$REGISTRY_USER" --password-stdin

# Build tracker
echo ""
echo "🔨 Building tracker image..."
docker build --platform linux/amd64 -t "$REGISTRY/$NAMESPACE/p2p-tracker:$IMAGE_TAG" -f docker/tracker.Dockerfile .

# Build peer
echo ""
echo "🔨 Building peer image..."
docker build --platform linux/amd64 -t "$REGISTRY/$NAMESPACE/p2p-peer:$IMAGE_TAG" -f docker/peer.Dockerfile .

# Push images
echo ""
echo "📤 Pushing tracker image..."
docker push "$REGISTRY/$NAMESPACE/p2p-tracker:$IMAGE_TAG"

echo ""
echo "📤 Pushing peer image..."
docker push "$REGISTRY/$NAMESPACE/p2p-peer:$IMAGE_TAG"

echo ""
echo "✅ Done! Images pushed to $REGISTRY"
echo "   - $REGISTRY/$NAMESPACE/p2p-tracker:$IMAGE_TAG"
echo "   - $REGISTRY/$NAMESPACE/p2p-peer:$IMAGE_TAG"

