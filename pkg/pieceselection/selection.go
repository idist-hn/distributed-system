// Package pieceselection implements smart piece selection algorithms for P2P downloads
package pieceselection

import (
	"math/rand"
	"slices"
	"sort"
	"sync"
	"time"
)

// PieceInfo contains information about a piece
type PieceInfo struct {
	Index      int      // Piece index
	Hash       string   // Piece hash
	Size       int64    // Piece size in bytes
	Available  int      // Number of peers that have this piece
	Downloaded bool     // Whether this piece has been downloaded
	Peers      []string // List of peer IDs that have this piece
}

// Selector interface for piece selection strategies
type Selector interface {
	// SelectNext returns the next piece to download and the peer to download from
	SelectNext(pieces []PieceInfo, availablePeers []string) (pieceIndex int, peerID string, ok bool)
	// Name returns the strategy name
	Name() string
}

// RarestFirstSelector implements the rarest-first piece selection strategy
// This prioritizes downloading pieces that are least available in the swarm
type RarestFirstSelector struct {
	mu   sync.Mutex
	rng  *rand.Rand
	name string
}

// NewRarestFirstSelector creates a new rarest-first selector
func NewRarestFirstSelector() *RarestFirstSelector {
	return &RarestFirstSelector{
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
		name: "rarest-first",
	}
}

// Name returns the strategy name
func (s *RarestFirstSelector) Name() string {
	return s.name
}

// SelectNext selects the rarest piece that hasn't been downloaded yet
func (s *RarestFirstSelector) SelectNext(pieces []PieceInfo, availablePeers []string) (int, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Filter pieces that need downloading and have available peers
	var candidates []PieceInfo
	for _, p := range pieces {
		if !p.Downloaded && p.Available > 0 && len(p.Peers) > 0 {
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		return -1, "", false
	}

	// Sort by availability (rarest first)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Available < candidates[j].Available
	})

	// Find all pieces with the same (lowest) availability
	minAvail := candidates[0].Available
	var rarestPieces []PieceInfo
	for _, p := range candidates {
		if p.Available == minAvail {
			rarestPieces = append(rarestPieces, p)
		} else {
			break
		}
	}

	// Randomly select one of the rarest pieces
	selected := rarestPieces[s.rng.Intn(len(rarestPieces))]

	// Select a random peer that has this piece
	peerID := selected.Peers[s.rng.Intn(len(selected.Peers))]

	return selected.Index, peerID, true
}

// RandomFirstSelector implements random piece selection (good for initial bootstrap)
type RandomFirstSelector struct {
	mu   sync.Mutex
	rng  *rand.Rand
	name string
}

// NewRandomFirstSelector creates a new random-first selector
func NewRandomFirstSelector() *RandomFirstSelector {
	return &RandomFirstSelector{
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
		name: "random-first",
	}
}

// Name returns the strategy name
func (s *RandomFirstSelector) Name() string {
	return s.name
}

// SelectNext randomly selects a piece that hasn't been downloaded
func (s *RandomFirstSelector) SelectNext(pieces []PieceInfo, availablePeers []string) (int, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var candidates []PieceInfo
	for _, p := range pieces {
		if !p.Downloaded && len(p.Peers) > 0 {
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		return -1, "", false
	}

	selected := candidates[s.rng.Intn(len(candidates))]
	peerID := selected.Peers[s.rng.Intn(len(selected.Peers))]

	return selected.Index, peerID, true
}

// SequentialSelector downloads pieces in order (useful for streaming)
type SequentialSelector struct {
	name string
}

// NewSequentialSelector creates a new sequential selector
func NewSequentialSelector() *SequentialSelector {
	return &SequentialSelector{name: "sequential"}
}

// Name returns the strategy name
func (s *SequentialSelector) Name() string {
	return s.name
}

// SelectNext selects the first piece that hasn't been downloaded
func (s *SequentialSelector) SelectNext(pieces []PieceInfo, availablePeers []string) (int, string, bool) {
	for _, p := range pieces {
		if !p.Downloaded && len(p.Peers) > 0 {
			return p.Index, p.Peers[0], true
		}
	}
	return -1, "", false
}

// EndgameSelector implements endgame mode - request remaining pieces from all peers
// Used when only a few pieces remain to speed up completion
type EndgameSelector struct {
	mu              sync.Mutex
	rng             *rand.Rand
	name            string
	threshold       int              // Number of pieces remaining to trigger endgame
	requestedPieces map[int]bool     // Track pieces already requested
	piecePeers      map[int][]string // Track which peers were asked for each piece
}

// NewEndgameSelector creates a new endgame selector
func NewEndgameSelector(threshold int) *EndgameSelector {
	if threshold <= 0 {
		threshold = 5 // Default: trigger when 5 or fewer pieces remain
	}
	return &EndgameSelector{
		rng:             rand.New(rand.NewSource(time.Now().UnixNano())),
		name:            "endgame",
		threshold:       threshold,
		requestedPieces: make(map[int]bool),
		piecePeers:      make(map[int][]string),
	}
}

// Name returns the strategy name
func (s *EndgameSelector) Name() string {
	return s.name
}

// ShouldActivate returns true if endgame mode should be activated
func (s *EndgameSelector) ShouldActivate(pieces []PieceInfo) bool {
	remaining := 0
	for _, p := range pieces {
		if !p.Downloaded {
			remaining++
		}
	}
	return remaining > 0 && remaining <= s.threshold
}

// SelectNext in endgame mode requests from multiple peers
func (s *EndgameSelector) SelectNext(pieces []PieceInfo, availablePeers []string) (int, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find pieces not yet downloaded
	var remaining []PieceInfo
	for _, p := range pieces {
		if !p.Downloaded && len(p.Peers) > 0 {
			remaining = append(remaining, p)
		}
	}

	if len(remaining) == 0 {
		return -1, "", false
	}

	// Try to find a piece-peer combination we haven't tried yet
	for _, p := range remaining {
		askedPeers := s.piecePeers[p.Index]
		for _, peer := range p.Peers {
			if !contains(askedPeers, peer) {
				// Found a new peer to ask for this piece
				s.piecePeers[p.Index] = append(s.piecePeers[p.Index], peer)
				s.requestedPieces[p.Index] = true
				return p.Index, peer, true
			}
		}
	}

	// All combinations tried, pick random remaining piece and peer
	piece := remaining[s.rng.Intn(len(remaining))]
	peer := piece.Peers[s.rng.Intn(len(piece.Peers))]
	return piece.Index, peer, true
}

// Reset clears the endgame state (call when download completes or restarts)
func (s *EndgameSelector) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestedPieces = make(map[int]bool)
	s.piecePeers = make(map[int][]string)
}

// CancelPiece cancels requests for a piece (call when piece is received)
func (s *EndgameSelector) CancelPiece(pieceIndex int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.requestedPieces, pieceIndex)
	delete(s.piecePeers, pieceIndex)
}

func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
