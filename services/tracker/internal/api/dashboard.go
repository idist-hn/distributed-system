package api

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

//go:embed templates/*
var templatesFS embed.FS

// DashboardData contains data for the dashboard template
type DashboardData struct {
	Version       string
	PeersOnline   int
	PeersTotal    int
	FilesCount    int
	RelayPeers    int
	Uptime        string
	Peers         []PeerView
	Files         []FileView
	RecentEvents  []EventView
	LastRefreshed string
}

// PeerView represents a peer for display
type PeerView struct {
	ID         string
	IP         string
	Port       int
	Status     string
	LastSeen   string
	FilesCount int
	BytesUp    string
	BytesDown  string
}

// FileView represents a file for display
type FileView struct {
	Hash       string
	FullHash   string
	Name       string
	Size       string
	Category   string
	PeersCount int
	AddedAt    string
}

// EventView represents an event for display
type EventView struct {
	Type      string
	Message   string
	Timestamp string
}

var startTime = time.Now()

// DashboardHandler serves the web dashboard
func (s *Server) DashboardHandler() http.HandlerFunc {
	tmpl := template.Must(template.ParseFS(templatesFS, "templates/dashboard.html"))

	return func(w http.ResponseWriter, r *http.Request) {
		peersOnline, peersTotal, filesCount := s.storage.GetStats()
		relayPeers := len(s.relayHub.GetConnectedPeers())

		// Get peers list
		allPeers := s.storage.ListAllPeers()
		peerViews := make([]PeerView, 0, len(allPeers))
		for _, p := range allPeers {
			status := "offline"
			if time.Since(p.LastSeen) < 90*time.Second {
				status = "online"
			}
			peerID := p.ID
			if len(peerID) > 8 {
				peerID = peerID[:8] + "..."
			}
			peerViews = append(peerViews, PeerView{
				ID:         peerID,
				IP:         p.IP,
				Port:       p.Port,
				Status:     status,
				LastSeen:   formatDuration(time.Since(p.LastSeen)),
				FilesCount: p.FilesShared,
				BytesUp:    formatBytes(p.BytesUploaded),
				BytesDown:  formatBytes(p.BytesDownloaded),
			})
		}

		// Get files list
		allFiles := s.storage.ListFiles()
		fileViews := make([]FileView, 0, len(allFiles))
		for _, f := range allFiles {
			hash := f.Hash
			fullHash := f.Hash
			if len(hash) > 12 {
				hash = hash[:12] + "..."
			}
			fileViews = append(fileViews, FileView{
				Hash:       hash,
				FullHash:   fullHash,
				Name:       truncate(f.Name, 40),
				Size:       formatBytes(f.Size),
				Category:   "",
				PeersCount: f.Seeders + f.Leechers,
				AddedAt:    f.AddedAt.Format("2006-01-02 15:04"),
			})
		}

		data := DashboardData{
			Version:       Version,
			PeersOnline:   peersOnline,
			PeersTotal:    peersTotal,
			FilesCount:    filesCount,
			RelayPeers:    relayPeers,
			Uptime:        formatDuration(time.Since(startTime)),
			Peers:         peerViews,
			Files:         fileViews,
			LastRefreshed: time.Now().Format("15:04:05"),
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		tmpl.Execute(w, data)
	}
}

// formatBytes formats bytes to human readable string
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration to human readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	}
	return fmt.Sprintf("%dd ago", int(d.Hours()/24))
}

// truncate truncates a string to max length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
