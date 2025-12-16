// Package magnet implements magnet URI parsing and generation for P2P file sharing
package magnet

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Magnet represents a magnet URI for file sharing
type Magnet struct {
	InfoHash    string   // File hash (xt=urn:sha256:...)
	DisplayName string   // File name (dn=...)
	Size        int64    // File size in bytes (xl=...)
	Trackers    []string // Tracker URLs (tr=...)
	WebSeeds    []string // Web seed URLs (ws=...)
	Keywords    []string // Keywords for search (kt=...)
	ChunkSize   int      // Chunk size in bytes (x.cs=...)
	TotalChunks int      // Total chunks (x.tc=...)
}

var (
	ErrInvalidMagnet   = errors.New("invalid magnet URI")
	ErrMissingInfoHash = errors.New("missing info hash")
)

// Parse parses a magnet URI string into a Magnet struct
func Parse(magnetURI string) (*Magnet, error) {
	if !strings.HasPrefix(magnetURI, "magnet:?") {
		return nil, ErrInvalidMagnet
	}

	query := strings.TrimPrefix(magnetURI, "magnet:?")
	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse magnet query: %w", err)
	}

	m := &Magnet{}

	// Parse info hash (xt=urn:sha256:HASH or xt=urn:btih:HASH)
	if xt := values.Get("xt"); xt != "" {
		hash, err := parseInfoHash(xt)
		if err != nil {
			return nil, err
		}
		m.InfoHash = hash
	}

	if m.InfoHash == "" {
		return nil, ErrMissingInfoHash
	}

	// Parse display name
	m.DisplayName = values.Get("dn")

	// Parse size
	if xl := values.Get("xl"); xl != "" {
		size, err := strconv.ParseInt(xl, 10, 64)
		if err == nil {
			m.Size = size
		}
	}

	// Parse trackers
	m.Trackers = values["tr"]

	// Parse web seeds
	m.WebSeeds = values["ws"]

	// Parse keywords
	if kt := values.Get("kt"); kt != "" {
		m.Keywords = strings.Split(kt, "+")
	}

	// Parse custom extensions
	if cs := values.Get("x.cs"); cs != "" {
		if chunkSize, err := strconv.Atoi(cs); err == nil {
			m.ChunkSize = chunkSize
		}
	}
	if tc := values.Get("x.tc"); tc != "" {
		if totalChunks, err := strconv.Atoi(tc); err == nil {
			m.TotalChunks = totalChunks
		}
	}

	return m, nil
}

// String converts the Magnet to a magnet URI string
func (m *Magnet) String() string {
	var parts []string

	// Info hash (required)
	parts = append(parts, fmt.Sprintf("xt=urn:sha256:%s", m.InfoHash))

	// Display name
	if m.DisplayName != "" {
		parts = append(parts, fmt.Sprintf("dn=%s", url.QueryEscape(m.DisplayName)))
	}

	// Size
	if m.Size > 0 {
		parts = append(parts, fmt.Sprintf("xl=%d", m.Size))
	}

	// Trackers
	for _, tr := range m.Trackers {
		parts = append(parts, fmt.Sprintf("tr=%s", url.QueryEscape(tr)))
	}

	// Web seeds
	for _, ws := range m.WebSeeds {
		parts = append(parts, fmt.Sprintf("ws=%s", url.QueryEscape(ws)))
	}

	// Keywords
	if len(m.Keywords) > 0 {
		parts = append(parts, fmt.Sprintf("kt=%s", strings.Join(m.Keywords, "+")))
	}

	// Custom extensions
	if m.ChunkSize > 0 {
		parts = append(parts, fmt.Sprintf("x.cs=%d", m.ChunkSize))
	}
	if m.TotalChunks > 0 {
		parts = append(parts, fmt.Sprintf("x.tc=%d", m.TotalChunks))
	}

	return "magnet:?" + strings.Join(parts, "&")
}

// New creates a new Magnet with the required fields
func New(infoHash, displayName string, size int64) *Magnet {
	return &Magnet{
		InfoHash:    infoHash,
		DisplayName: displayName,
		Size:        size,
	}
}

// AddTracker adds a tracker URL
func (m *Magnet) AddTracker(trackerURL string) *Magnet {
	m.Trackers = append(m.Trackers, trackerURL)
	return m
}

// SetChunkInfo sets chunk information
func (m *Magnet) SetChunkInfo(chunkSize, totalChunks int) *Magnet {
	m.ChunkSize = chunkSize
	m.TotalChunks = totalChunks
	return m
}

func parseInfoHash(xt string) (string, error) {
	// Support multiple URN formats
	prefixes := []string{
		"urn:sha256:",
		"urn:sha-256:",
		"urn:btih:", // BitTorrent compatible
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(xt, prefix) {
			hash := strings.TrimPrefix(xt, prefix)
			// Validate hex string
			if _, err := hex.DecodeString(hash); err != nil {
				return "", fmt.Errorf("invalid hash format: %w", err)
			}
			return hash, nil
		}
	}

	return "", fmt.Errorf("unsupported info hash format: %s", xt)
}

