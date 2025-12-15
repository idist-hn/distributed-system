package main

import (
	"flag"
	"log"

	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/api"
)

func main() {
	addr := flag.String("addr", ":8080", "Tracker server address")
	flag.Parse()

	log.Println("=== P2P File Sharing - Tracker Server ===")

	server := api.NewServer(*addr)
	if err := server.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
