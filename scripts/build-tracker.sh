#!/bin/bash
cd /Volumes/Personal/courses/distributed-system
echo "Building tracker Docker image..."
docker build --platform linux/amd64 -t registry.idist.dev/p2p-system/p2p-tracker:latest -f docker/tracker.Dockerfile .
echo "Pushing to registry..."
docker push registry.idist.dev/p2p-system/p2p-tracker:latest
echo "Done!"

