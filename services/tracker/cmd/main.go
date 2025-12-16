package main

import (
	"flag"
	"log"
	"os"

	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/api"
)

func main() {
	addr := flag.String("addr", ":8080", "Tracker server address")
	dbURL := flag.String("db", "", "PostgreSQL connection string (or use DATABASE_URL env)")
	flag.Parse()

	log.Println("=== P2P File Sharing - Tracker Server ===")

	// Check POSTGRES_URL or DATABASE_URL env if -db not provided
	connStr := *dbURL
	if connStr == "" {
		connStr = os.Getenv("POSTGRES_URL")
		if connStr == "" {
			connStr = os.Getenv("DATABASE_URL")
		}
	}

	var server *api.Server
	var err error

	if connStr != "" {
		log.Println("Using PostgreSQL storage")
		server, err = api.NewServerWithDB(*addr, connStr)
		if err != nil {
			log.Fatalf("Failed to initialize database: %v", err)
		}
	} else {
		log.Println("Using in-memory storage (data will be lost on restart)")
		server = api.NewServer(*addr)
	}

	if err := server.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
