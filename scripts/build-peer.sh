#!/bin/bash
cd /Volumes/Personal/courses/distributed-system
echo "Building peer for Linux AMD64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bin/peer-linux-amd64 ./services/peer/cmd
echo "Done!"
ls -la bin/peer-linux-amd64

