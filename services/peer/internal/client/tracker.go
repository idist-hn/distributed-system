package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
)

// TrackerClient handles communication with the tracker server
type TrackerClient struct {
	baseURL    string
	httpClient *http.Client
	peerID     string
}

// NewTrackerClient creates a new tracker client
func NewTrackerClient(trackerURL, peerID string) *TrackerClient {
	return &TrackerClient{
		baseURL: trackerURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		peerID: peerID,
	}
}

// Register registers this peer with the tracker
func (c *TrackerClient) Register(ip string, port int) (*protocol.RegisterResponse, error) {
	req := protocol.RegisterRequest{
		PeerID: c.peerID,
		IP:     ip,
		Port:   port,
	}

	var resp protocol.RegisterResponse
	err := c.post("/api/peers/register", req, &resp)
	return &resp, err
}

// Heartbeat sends a heartbeat to the tracker
func (c *TrackerClient) Heartbeat(fileHashes []string) (*protocol.HeartbeatResponse, error) {
	req := protocol.HeartbeatRequest{
		PeerID:      c.peerID,
		FilesHashes: fileHashes,
	}

	var resp protocol.HeartbeatResponse
	err := c.post("/api/peers/heartbeat", req, &resp)
	return &resp, err
}

// Leave notifies the tracker that this peer is leaving
func (c *TrackerClient) Leave() error {
	url := fmt.Sprintf("%s/api/peers/%s", c.baseURL, c.peerID)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("leave failed with status: %d", resp.StatusCode)
	}
	return nil
}

// AnnounceFile announces a file to the tracker
func (c *TrackerClient) AnnounceFile(file *protocol.FileMetadata) (*protocol.AnnounceResponse, error) {
	req := protocol.AnnounceRequest{
		PeerID: c.peerID,
		File:   *file,
	}

	var resp protocol.AnnounceResponse
	err := c.post("/api/files/announce", req, &resp)
	return &resp, err
}

// ListFiles gets all available files from tracker
func (c *TrackerClient) ListFiles() (*protocol.ListFilesResponse, error) {
	var resp protocol.ListFilesResponse
	err := c.get("/api/files", &resp)
	return &resp, err
}

// GetPeers gets peers that have a specific file
func (c *TrackerClient) GetPeers(fileHash string) (*protocol.GetPeersResponse, error) {
	var resp protocol.GetPeersResponse
	err := c.get(fmt.Sprintf("/api/files/%s/peers", fileHash), &resp)
	return &resp, err
}

// Helper methods

func (c *TrackerClient) post(path string, body interface{}, result interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Post(c.baseURL+path, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *TrackerClient) get(path string, result interface{}) error {
	resp, err := c.httpClient.Get(c.baseURL + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(result)
}
