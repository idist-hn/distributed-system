package pieceselection

import (
	"testing"
)

func TestRarestFirstSelector_SelectNext(t *testing.T) {
	selector := NewRarestFirstSelector()

	tests := []struct {
		name           string
		pieces         []PieceInfo
		availablePeers []string
		wantOk         bool
		wantRarest     bool // Should select from rarest pieces
	}{
		{
			name:           "empty pieces",
			pieces:         []PieceInfo{},
			availablePeers: []string{"peer1"},
			wantOk:         false,
		},
		{
			name: "all downloaded",
			pieces: []PieceInfo{
				{Index: 0, Downloaded: true, Available: 1, Peers: []string{"peer1"}},
				{Index: 1, Downloaded: true, Available: 2, Peers: []string{"peer1", "peer2"}},
			},
			availablePeers: []string{"peer1"},
			wantOk:         false,
		},
		{
			name: "select rarest",
			pieces: []PieceInfo{
				{Index: 0, Downloaded: false, Available: 5, Peers: []string{"peer1", "peer2", "peer3", "peer4", "peer5"}},
				{Index: 1, Downloaded: false, Available: 1, Peers: []string{"peer1"}},
				{Index: 2, Downloaded: false, Available: 3, Peers: []string{"peer1", "peer2", "peer3"}},
			},
			availablePeers: []string{"peer1"},
			wantOk:         true,
			wantRarest:     true,
		},
		{
			name: "no peers for piece",
			pieces: []PieceInfo{
				{Index: 0, Downloaded: false, Available: 1, Peers: []string{}},
			},
			availablePeers: []string{"peer1"},
			wantOk:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pieceIdx, peerID, ok := selector.SelectNext(tt.pieces, tt.availablePeers)

			if ok != tt.wantOk {
				t.Errorf("SelectNext() ok = %v, want %v", ok, tt.wantOk)
			}

			if tt.wantOk && tt.wantRarest {
				// Should select piece with index 1 (availability = 1)
				if pieceIdx != 1 {
					t.Errorf("SelectNext() pieceIdx = %v, want 1 (rarest)", pieceIdx)
				}
				if peerID != "peer1" {
					t.Errorf("SelectNext() peerID = %v, want peer1", peerID)
				}
			}
		})
	}
}

func TestRandomFirstSelector_SelectNext(t *testing.T) {
	selector := NewRandomFirstSelector()

	pieces := []PieceInfo{
		{Index: 0, Downloaded: false, Available: 2, Peers: []string{"peer1", "peer2"}},
		{Index: 1, Downloaded: true, Available: 1, Peers: []string{"peer1"}},
		{Index: 2, Downloaded: false, Available: 1, Peers: []string{"peer3"}},
	}

	// Run multiple times to verify randomness
	selectedPieces := make(map[int]int)
	for i := 0; i < 100; i++ {
		pieceIdx, _, ok := selector.SelectNext(pieces, []string{"peer1", "peer2", "peer3"})
		if !ok {
			t.Fatal("SelectNext() should return ok=true")
		}
		selectedPieces[pieceIdx]++
	}

	// Should only select pieces 0 and 2 (not 1 which is downloaded)
	if _, ok := selectedPieces[1]; ok {
		t.Error("Should not select downloaded piece")
	}
	if selectedPieces[0] == 0 || selectedPieces[2] == 0 {
		t.Error("Should select both available pieces over 100 iterations")
	}
}

func TestSequentialSelector_SelectNext(t *testing.T) {
	selector := NewSequentialSelector()

	pieces := []PieceInfo{
		{Index: 0, Downloaded: true, Available: 1, Peers: []string{"peer1"}},
		{Index: 1, Downloaded: true, Available: 1, Peers: []string{"peer1"}},
		{Index: 2, Downloaded: false, Available: 1, Peers: []string{"peer2"}},
		{Index: 3, Downloaded: false, Available: 1, Peers: []string{"peer3"}},
	}

	pieceIdx, peerID, ok := selector.SelectNext(pieces, []string{"peer1", "peer2", "peer3"})

	if !ok {
		t.Fatal("SelectNext() should return ok=true")
	}
	if pieceIdx != 2 {
		t.Errorf("SelectNext() pieceIdx = %v, want 2 (first not downloaded)", pieceIdx)
	}
	if peerID != "peer2" {
		t.Errorf("SelectNext() peerID = %v, want peer2", peerID)
	}
}

func TestSelectorNames(t *testing.T) {
	tests := []struct {
		selector Selector
		wantName string
	}{
		{NewRarestFirstSelector(), "rarest-first"},
		{NewRandomFirstSelector(), "random-first"},
		{NewSequentialSelector(), "sequential"},
		{NewEndgameSelector(5), "endgame"},
	}

	for _, tt := range tests {
		if got := tt.selector.Name(); got != tt.wantName {
			t.Errorf("%T.Name() = %v, want %v", tt.selector, got, tt.wantName)
		}
	}
}

func TestEndgameSelector_ShouldActivate(t *testing.T) {
	selector := NewEndgameSelector(3)

	tests := []struct {
		name   string
		pieces []PieceInfo
		want   bool
	}{
		{
			name:   "no remaining",
			pieces: []PieceInfo{{Downloaded: true}, {Downloaded: true}},
			want:   false,
		},
		{
			name:   "below threshold",
			pieces: []PieceInfo{{Downloaded: false}, {Downloaded: false}, {Downloaded: true}},
			want:   true,
		},
		{
			name:   "above threshold",
			pieces: []PieceInfo{{Downloaded: false}, {Downloaded: false}, {Downloaded: false}, {Downloaded: false}},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := selector.ShouldActivate(tt.pieces); got != tt.want {
				t.Errorf("ShouldActivate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEndgameSelector_SelectNext(t *testing.T) {
	selector := NewEndgameSelector(5)

	pieces := []PieceInfo{
		{Index: 0, Downloaded: true, Peers: []string{"peer1"}},
		{Index: 1, Downloaded: false, Peers: []string{"peer1", "peer2"}},
		{Index: 2, Downloaded: false, Peers: []string{"peer2", "peer3"}},
	}

	// First call should return a piece
	pieceIdx, peerID, ok := selector.SelectNext(pieces, []string{"peer1", "peer2", "peer3"})
	if !ok {
		t.Fatal("SelectNext() should return ok=true")
	}
	if pieceIdx != 1 && pieceIdx != 2 {
		t.Errorf("pieceIdx = %d, want 1 or 2", pieceIdx)
	}
	if peerID == "" {
		t.Error("peerID should not be empty")
	}

	// Should track which peers were asked
	// Call multiple times to try different combinations
	for i := 0; i < 10; i++ {
		selector.SelectNext(pieces, []string{"peer1", "peer2", "peer3"})
	}
}

func TestEndgameSelector_CancelPiece(t *testing.T) {
	selector := NewEndgameSelector(5)

	pieces := []PieceInfo{
		{Index: 0, Downloaded: false, Peers: []string{"peer1"}},
	}

	selector.SelectNext(pieces, []string{"peer1"})
	selector.CancelPiece(0)

	// After cancel, state should be cleared for this piece
	if _, ok := selector.requestedPieces[0]; ok {
		t.Error("CancelPiece should clear requestedPieces")
	}
}

func TestEndgameSelector_Reset(t *testing.T) {
	selector := NewEndgameSelector(5)

	pieces := []PieceInfo{
		{Index: 0, Downloaded: false, Peers: []string{"peer1"}},
	}

	selector.SelectNext(pieces, []string{"peer1"})
	selector.Reset()

	if len(selector.requestedPieces) != 0 {
		t.Error("Reset should clear requestedPieces")
	}
	if len(selector.piecePeers) != 0 {
		t.Error("Reset should clear piecePeers")
	}
}
