package magnet

import (
	"strings"
	"testing"
)

func TestParse_Valid(t *testing.T) {
	uri := "magnet:?xt=urn:sha256:abc123def456&dn=test.txt&xl=1024&tr=https://tracker.example.com"

	m, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if m.InfoHash != "abc123def456" {
		t.Errorf("InfoHash = %s, want abc123def456", m.InfoHash)
	}
	if m.DisplayName != "test.txt" {
		t.Errorf("DisplayName = %s, want test.txt", m.DisplayName)
	}
	if m.Size != 1024 {
		t.Errorf("Size = %d, want 1024", m.Size)
	}
	if len(m.Trackers) != 1 || m.Trackers[0] != "https://tracker.example.com" {
		t.Errorf("Trackers = %v, want [https://tracker.example.com]", m.Trackers)
	}
}

func TestParse_MissingInfoHash(t *testing.T) {
	uri := "magnet:?dn=test.txt"

	_, err := Parse(uri)
	if err != ErrMissingInfoHash {
		t.Errorf("Parse() error = %v, want ErrMissingInfoHash", err)
	}
}

func TestParse_InvalidMagnet(t *testing.T) {
	uris := []string{
		"http://example.com",
		"magnet:",
		"not-a-magnet",
	}

	for _, uri := range uris {
		_, err := Parse(uri)
		if err == nil {
			t.Errorf("Parse(%s) should return error", uri)
		}
	}
}

func TestParse_CustomExtensions(t *testing.T) {
	uri := "magnet:?xt=urn:sha256:abc123&x.cs=262144&x.tc=100"

	m, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if m.ChunkSize != 262144 {
		t.Errorf("ChunkSize = %d, want 262144", m.ChunkSize)
	}
	if m.TotalChunks != 100 {
		t.Errorf("TotalChunks = %d, want 100", m.TotalChunks)
	}
}

func TestMagnet_String(t *testing.T) {
	m := New("abc123def456", "test.txt", 1024)
	m.AddTracker("https://tracker.example.com")
	m.SetChunkInfo(262144, 4)

	uri := m.String()

	if !strings.HasPrefix(uri, "magnet:?") {
		t.Errorf("String() should start with magnet:?")
	}
	if !strings.Contains(uri, "xt=urn:sha256:abc123def456") {
		t.Error("String() should contain info hash")
	}
	if !strings.Contains(uri, "dn=test.txt") {
		t.Error("String() should contain display name")
	}
	if !strings.Contains(uri, "xl=1024") {
		t.Error("String() should contain size")
	}
	if !strings.Contains(uri, "x.cs=262144") {
		t.Error("String() should contain chunk size")
	}
}

func TestRoundTrip(t *testing.T) {
	original := &Magnet{
		InfoHash:    "abc123def456789012345678901234567890123456789012345678901234",
		DisplayName: "My File.zip",
		Size:        10485760,
		Trackers:    []string{"https://tracker1.com", "https://tracker2.com"},
		ChunkSize:   262144,
		TotalChunks: 40,
	}

	uri := original.String()
	parsed, err := Parse(uri)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if parsed.InfoHash != original.InfoHash {
		t.Errorf("InfoHash mismatch: got %s, want %s", parsed.InfoHash, original.InfoHash)
	}
	if parsed.Size != original.Size {
		t.Errorf("Size mismatch: got %d, want %d", parsed.Size, original.Size)
	}
	if parsed.ChunkSize != original.ChunkSize {
		t.Errorf("ChunkSize mismatch: got %d, want %d", parsed.ChunkSize, original.ChunkSize)
	}
	if len(parsed.Trackers) != len(original.Trackers) {
		t.Errorf("Trackers count mismatch: got %d, want %d", len(parsed.Trackers), len(original.Trackers))
	}
}

func TestNew(t *testing.T) {
	m := New("hash123", "file.txt", 2048)

	if m.InfoHash != "hash123" {
		t.Errorf("InfoHash = %s, want hash123", m.InfoHash)
	}
	if m.DisplayName != "file.txt" {
		t.Errorf("DisplayName = %s, want file.txt", m.DisplayName)
	}
	if m.Size != 2048 {
		t.Errorf("Size = %d, want 2048", m.Size)
	}
}

func TestAddTracker(t *testing.T) {
	m := New("hash", "file", 100)
	m.AddTracker("tr1").AddTracker("tr2")

	if len(m.Trackers) != 2 {
		t.Errorf("Trackers count = %d, want 2", len(m.Trackers))
	}
}

