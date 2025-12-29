package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/downloader"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/p2p"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/relay"
	"github.com/p2p-filesharing/distributed-system/services/peer/internal/storage"
)

func main() {
	trackerURL := flag.String("tracker", "https://p2p.idist.dev", "Tracker server URL")
	fileHash := flag.String("hash", "", "File hash to download")
	outputDir := flag.String("output", "./downloads", "Output directory")
	listFiles := flag.Bool("list", false, "List available files")
	flag.Parse()

	// Handle positional argument for hash
	if *fileHash == "" && flag.NArg() > 0 {
		*fileHash = flag.Arg(0)
	}

	// Create simple tracker client (no auth needed for public APIs)
	client := NewSimpleClient(*trackerURL)

	if *listFiles {
		files, err := client.ListFiles()
		if err != nil {
			log.Fatalf("Failed to list files: %v", err)
		}
		if len(files) == 0 {
			fmt.Println("No files available")
			return
		}
		fmt.Println("\nAvailable files:")
		fmt.Println(strings.Repeat("-", 80))
		for _, f := range files {
			fmt.Printf("%-12s  %-40s  %10s  %d seeders\n",
				truncate(f.Hash, 12), truncate(f.Name, 40), formatSize(f.Size), f.Seeders)
		}
		fmt.Println(strings.Repeat("-", 80))
		fmt.Println("\nTo download: p2p-download <hash>")
		return
	}

	if *fileHash == "" {
		fmt.Println("Usage: p2p-download [options] <hash>")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  p2p-download --list                  # List available files")
		fmt.Println("  p2p-download abc123def456            # Download file by hash")
		fmt.Println("  p2p-download --output /tmp abc123    # Download to specific directory")
		os.Exit(1)
	}

	// Get file info
	fileInfo, err := client.GetPeers(*fileHash)
	if err != nil {
		log.Fatalf("Failed to get file info: %v", err)
	}

	if len(fileInfo.Peers) == 0 {
		log.Fatalf("No peers available for this file")
	}

	fmt.Printf("Downloading: %s (%s)\n", fileInfo.FileName, formatSize(fileInfo.FileSize))
	fmt.Printf("Chunks: %d, Peers: %d\n", fileInfo.ChunkCount, len(fileInfo.Peers))

	// Initialize storage
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	store, err := storage.NewLocalStorage(*outputDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}

	// Initialize P2P client
	peerID := uuid.New().String()
	p2pClient := p2p.NewClient(peerID)

	// Initialize relay client for NAT traversal fallback
	relayClient := relay.NewClient(peerID, *trackerURL)
	if err := relayClient.Connect(); err != nil {
		log.Printf("Warning: Relay connection failed: %v (will use direct TCP only)", err)
	} else {
		log.Printf("Relay connected for NAT traversal fallback")
		defer relayClient.Close()
	}

	// Start download with relay support
	dl := downloader.NewWithRelay(store, p2pClient, relayClient)
	if err := dl.DownloadFile(fileInfo); err != nil {
		log.Fatalf("Download failed: %v", err)
	}

	fmt.Printf("\n✓ Download complete: %s\n", filepath.Join(*outputDir, fileInfo.FileName))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
