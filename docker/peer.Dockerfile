# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the peer binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o peer ./services/peer/cmd

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Create data directory
RUN mkdir -p /data/shared /data/downloads /data/temp

# Copy binary from builder
COPY --from=builder /app/peer .

# Expose ports
EXPOSE 6881

# Data volume
VOLUME ["/data"]

# Run the peer
CMD ["./peer", "-data", "/data"]

