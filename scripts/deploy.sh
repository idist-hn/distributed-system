#!/bin/bash
set -e

echo "========================================"
echo "P2P File Sharing - Deploy to Kubernetes"
echo "========================================"

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo "❌ kubectl not found. Please install kubectl first."
    exit 1
fi

# Create namespace
echo ""
echo "📁 Creating namespace..."
kubectl apply -f k8s/namespace.yaml

# Create registry secret
echo ""
echo "🔐 Creating registry secret..."
kubectl delete secret registry-credentials -n p2p-system --ignore-not-found
kubectl create secret docker-registry registry-credentials \
    --docker-server=registry.idist.dev \
    --docker-username=idist-hn \
    --docker-password="HanhHanh2508@" \
    -n p2p-system

# Deploy tracker
echo ""
echo "🚀 Deploying tracker..."
kubectl apply -f k8s/tracker-deployment.yaml

# Wait for tracker to be ready
echo ""
echo "⏳ Waiting for tracker to be ready..."
kubectl rollout status deployment/tracker -n p2p-system --timeout=120s

# Deploy peers
echo ""
echo "🚀 Deploying peers..."
kubectl apply -f k8s/peer-statefulset.yaml

# Deploy ingress
echo ""
echo "🌐 Deploying ingress..."
kubectl apply -f k8s/ingress.yaml

echo ""
echo "✅ Deployment complete!"
echo ""
echo "📊 Status:"
kubectl get all -n p2p-system

echo ""
echo "🌐 Access the tracker at: https://p2p.idist.dev"
echo ""
echo "📝 To check logs:"
echo "   kubectl logs -f deployment/tracker -n p2p-system"
echo "   kubectl logs -f statefulset/peer -n p2p-system"

