package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
)

// SimpleClient is a lightweight tracker client for download-only operations
type SimpleClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewSimpleClient creates a new simple client
func NewSimpleClient(trackerURL string) *SimpleClient {
	return &SimpleClient{
		baseURL: trackerURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ListFiles gets all available files from tracker
func (c *SimpleClient) ListFiles() ([]protocol.FileListItem, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/files")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	var listResp protocol.ListFilesResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, err
	}

	return listResp.Files, nil
}

// GetPeers gets peers that have a specific file
func (c *SimpleClient) GetPeers(fileHash string) (*protocol.GetPeersResponse, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/api/files/%s/peers", c.baseURL, fileHash))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status: %d", resp.StatusCode)
	}

	var peersResp protocol.GetPeersResponse
	if err := json.NewDecoder(resp.Body).Decode(&peersResp); err != nil {
		return nil, err
	}

	return &peersResp, nil
}

