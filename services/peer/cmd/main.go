package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/p2p-filesharing/distributed-system/pkg/chunker"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/client"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/downloader"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/p2p"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/storage"
)

func main() {
	trackerURL := flag.String("tracker", "https://p2p.idist.dev", "Tracker server URL")
	port := flag.Int("port", 6881, "P2P listen port")
	dataDir := flag.String("data", "./data", "Data directory")
	daemon := flag.Bool("daemon", false, "Run in daemon mode (no CLI)")
	apiKey := flag.String("api-key", "", "API key for tracker authentication")
	flag.Parse()

	// Generate peer ID
	peerID := uuid.New().String()
	log.Printf("=== P2P File Sharing - Peer Node ===")
	log.Printf("Peer ID: %s", peerID)
	log.Printf("Tracker: %s", *trackerURL)

	// Initialize storage
	store, err := storage.NewLocalStorage(*dataDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize tracker client
	var tracker *client.TrackerClient
	if *apiKey != "" {
		log.Printf("Using API key authentication")
		tracker = client.NewTrackerClientWithAPIKey(*trackerURL, peerID, *apiKey)
	} else {
		tracker = client.NewTrackerClient(*trackerURL, peerID)
	}

	// Initialize P2P server
	p2pServer := p2p.NewServer(*port, peerID, store)

	// Initialize P2P client
	p2pClient := p2p.NewClient(peerID)

	// Initialize chunker
	fileChunker := chunker.New(chunker.DefaultChunkSize)

	// Start P2P server
	if err := p2pServer.Start(); err != nil {
		log.Fatalf("Failed to start P2P server: %v", err)
	}

	// Get public IP
	publicIP := getPublicIP()
	log.Printf("Public IP: %s", publicIP)

	// Register with tracker
	resp, err := tracker.Register(publicIP, *port)
	if err != nil {
		log.Fatalf("Failed to register with tracker: %v", err)
	}
	log.Printf("Registered with tracker: %s", resp.Message)

	// Start heartbeat goroutine
	go startHeartbeat(tracker, store)

	// Handle graceful shutdown
	go handleShutdown(tracker, p2pServer)

	// Run in daemon mode or CLI mode
	if *daemon {
		log.Println("Running in daemon mode...")

		// Create shared directory if not exists
		sharedDir := filepath.Join(*dataDir, "shared")
		if err := os.MkdirAll(sharedDir, 0755); err != nil {
			log.Printf("Failed to create shared directory: %v", err)
		} else {
			log.Printf("Shared directory: %s", sharedDir)
			// Initial scan
			scanAndShareFiles(sharedDir, tracker, store, fileChunker)
			// Start periodic scan
			go startFileScan(sharedDir, tracker, store, fileChunker)
		}

		// Block forever, waiting for shutdown signal
		select {}
	} else {
		// Start CLI loop
		runCLI(tracker, store, p2pClient, fileChunker)
	}
}

func startHeartbeat(tracker *client.TrackerClient, store *storage.LocalStorage) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		hashes := store.GetAllSharedHashes()
		if _, err := tracker.Heartbeat(hashes); err != nil {
			log.Printf("Heartbeat failed: %v", err)
		}
	}
}

// scanAndShareFiles scans the shared directory and announces all files to tracker
func scanAndShareFiles(sharedDir string, tracker *client.TrackerClient, store *storage.LocalStorage, c *chunker.Chunker) {
	entries, err := os.ReadDir(sharedDir)
	if err != nil {
		log.Printf("Error reading shared directory: %v", err)
		return
	}

	sharedCount := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories for now
		}

		filePath := filepath.Join(sharedDir, entry.Name())

		// Check if already shared
		if store.IsFileShared(filePath) {
			continue
		}

		// Chunk and share file
		metadata, err := c.ChunkFile(filePath)
		if err != nil {
			log.Printf("Error chunking file %s: %v", entry.Name(), err)
			continue
		}

		store.AddSharedFile(metadata, filePath)

		resp, err := tracker.AnnounceFile(metadata)
		if err != nil {
			log.Printf("Error announcing file %s: %v", entry.Name(), err)
			continue
		}

		log.Printf("Shared: %s (hash: %s, %d chunks)", metadata.Name, resp.FileID, len(metadata.Chunks))
		sharedCount++
	}

	if sharedCount > 0 {
		log.Printf("Shared %d new files from %s", sharedCount, sharedDir)
	}
}

// startFileScan periodically scans for new files in the shared directory
func startFileScan(sharedDir string, tracker *client.TrackerClient, store *storage.LocalStorage, c *chunker.Chunker) {
	ticker := time.NewTicker(60 * time.Second) // Scan every 60 seconds
	defer ticker.Stop()

	for range ticker.C {
		scanAndShareFiles(sharedDir, tracker, store, c)
	}
}

func handleShutdown(tracker *client.TrackerClient, server *p2p.Server) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	tracker.Leave()
	server.Stop()
	os.Exit(0)
}

func runCLI(tracker *client.TrackerClient, store *storage.LocalStorage, p2pClient *p2p.Client, fileChunker *chunker.Chunker) {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("\nCommands:")
	fmt.Println("  share <filepath>  - Share a file")
	fmt.Println("  list              - List available files")
	fmt.Println("  download <hash>   - Download a file")
	fmt.Println("  status            - Show status")
	fmt.Println("  quit              - Exit")
	fmt.Println()

	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		line := strings.TrimSpace(scanner.Text())
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 0 {
			continue
		}

		cmd := parts[0]
		var arg string
		if len(parts) > 1 {
			arg = parts[1]
		}

		switch cmd {
		case "share":
			cmdShare(arg, tracker, store, fileChunker)
		case "list":
			cmdList(tracker)
		case "download":
			cmdDownload(arg, tracker, store, p2pClient)
		case "status":
			cmdStatus(store)
		case "quit", "exit":
			fmt.Println("Goodbye!")
			return
		default:
			fmt.Println("Unknown command:", cmd)
		}
	}
}

func cmdShare(filepath string, tracker *client.TrackerClient, store *storage.LocalStorage, c *chunker.Chunker) {
	if filepath == "" {
		fmt.Println("Usage: share <filepath>")
		return
	}

	metadata, err := c.ChunkFile(filepath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	store.AddSharedFile(metadata, filepath)

	resp, err := tracker.AnnounceFile(metadata)
	if err != nil {
		fmt.Printf("Error announcing: %v\n", err)
		return
	}

	fmt.Printf("Shared: %s\n", metadata.Name)
	fmt.Printf("Hash: %s\n", resp.FileID)
	fmt.Printf("Chunks: %d\n", len(metadata.Chunks))
}

func cmdList(tracker *client.TrackerClient) {
	resp, err := tracker.ListFiles()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	if len(resp.Files) == 0 {
		fmt.Println("No files available")
		return
	}

	fmt.Println("\nAvailable files:")
	for _, f := range resp.Files {
		fmt.Printf("  [%s] %s (%d bytes) - %d seeders\n", f.Hash[:8], f.Name, f.Size, f.Seeders)
	}
}

func cmdDownload(fileHash string, tracker *client.TrackerClient, store *storage.LocalStorage, p2pClient *p2p.Client) {
	if fileHash == "" {
		fmt.Println("Usage: download <hash>")
		return
	}

	// Get file info and peers from tracker
	fileInfo, err := tracker.GetPeers(fileHash)
	if err != nil {
		fmt.Printf("Error getting file info: %v\n", err)
		return
	}

	if len(fileInfo.Peers) == 0 {
		fmt.Println("No peers available for this file")
		return
	}

	fmt.Printf("Downloading: %s (%d bytes)\n", fileInfo.FileName, fileInfo.FileSize)
	fmt.Printf("Chunks: %d, Peers: %d\n", fileInfo.ChunkCount, len(fileInfo.Peers))

	// Start download
	dl := downloader.New(store, p2pClient)
	if err := dl.DownloadFile(fileInfo); err != nil {
		fmt.Printf("Download failed: %v\n", err)
		return
	}

	fmt.Printf("Download complete: %s\n", fileInfo.FileName)
}

func cmdStatus(store *storage.LocalStorage) {
	hashes := store.GetAllSharedHashes()
	fmt.Printf("Sharing %d files\n", len(hashes))
}

// getPublicIP retrieves the public IP address of this peer
func getPublicIP() string {
	// List of services to try
	services := []string{
		"https://api.ipify.org",
		"https://ifconfig.me/ip",
		"https://icanhazip.com",
		"https://ipecho.net/plain",
	}

	client := &http.Client{Timeout: 5 * time.Second}

	for _, svc := range services {
		resp, err := client.Get(svc)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			ip := strings.TrimSpace(string(body))
			if ip != "" {
				return ip
			}
		}
	}

	// Fallback to localhost if all services fail
	log.Println("Warning: Could not detect public IP, using 127.0.0.1")
	return "127.0.0.1"
}
