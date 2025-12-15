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

# Build the tracker binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o tracker ./services/tracker/cmd

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/tracker .

# Expose port
EXPOSE 8080

# Run the tracker
CMD ["./tracker"]

