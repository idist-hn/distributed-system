package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/p2p-filesharing/distributed-system/pkg/protocol"
	"github.com/p2p-filesharing/distributed-system/services/tracker/internal/models"
)

// DatabaseStorage implements persistent storage using PostgreSQL
type DatabaseStorage struct {
	db *sql.DB
}

// NewDatabaseStorage creates a new database storage
// connStr format: "postgres://user:password@host:port/dbname?sslmode=disable"
func NewDatabaseStorage(connStr string) (*DatabaseStorage, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	storage := &DatabaseStorage{db: db}
	if err := storage.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return storage, nil
}

// migrate creates the database schema
func (s *DatabaseStorage) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS peers (
		id TEXT PRIMARY KEY,
		ip TEXT NOT NULL,
		port INTEGER NOT NULL,
		hostname TEXT,
		registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		is_online BOOLEAN DEFAULT TRUE
	);

	CREATE TABLE IF NOT EXISTS files (
		hash TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		size BIGINT NOT NULL,
		chunk_size INTEGER NOT NULL,
		chunks TEXT NOT NULL,
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		added_by TEXT
	);

	CREATE TABLE IF NOT EXISTS file_peers (
		id SERIAL PRIMARY KEY,
		file_hash TEXT NOT NULL,
		peer_id TEXT NOT NULL,
		chunks_available TEXT NOT NULL,
		is_seeder BOOLEAN DEFAULT FALSE,
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(file_hash, peer_id),
		FOREIGN KEY (file_hash) REFERENCES files(hash) ON DELETE CASCADE,
		FOREIGN KEY (peer_id) REFERENCES peers(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_peers_online ON peers(is_online);
	CREATE INDEX IF NOT EXISTS idx_file_peers_file ON file_peers(file_hash);
	CREATE INDEX IF NOT EXISTS idx_file_peers_peer ON file_peers(peer_id);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}

	// Run migrations for new columns (safe to run multiple times)
	migrations := []string{
		// Peer reputation columns
		"ALTER TABLE peers ADD COLUMN IF NOT EXISTS bytes_uploaded BIGINT DEFAULT 0",
		"ALTER TABLE peers ADD COLUMN IF NOT EXISTS bytes_downloaded BIGINT DEFAULT 0",
		"ALTER TABLE peers ADD COLUMN IF NOT EXISTS files_shared INTEGER DEFAULT 0",
		"ALTER TABLE peers ADD COLUMN IF NOT EXISTS reputation REAL DEFAULT 50.0",
		// File category columns
		"ALTER TABLE files ADD COLUMN IF NOT EXISTS category TEXT DEFAULT 'other'",
		"ALTER TABLE files ADD COLUMN IF NOT EXISTS tags TEXT DEFAULT '[]'",
		// Indexes for new columns
		"CREATE INDEX IF NOT EXISTS idx_peers_reputation ON peers(reputation)",
		"CREATE INDEX IF NOT EXISTS idx_files_category ON files(category)",
	}

	for _, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			// Ignore errors for columns that already exist
			continue
		}
	}

	return nil
}

// Close closes the database connection
func (s *DatabaseStorage) Close() error {
	return s.db.Close()
}

// === Peer Operations ===

// RegisterPeer adds or updates a peer
func (s *DatabaseStorage) RegisterPeer(peer *models.Peer) error {
	query := `
		INSERT INTO peers (id, ip, port, hostname, registered_at, last_seen, is_online)
		VALUES ($1, $2, $3, $4, $5, $6, TRUE)
		ON CONFLICT(id) DO UPDATE SET
			ip = EXCLUDED.ip,
			port = EXCLUDED.port,
			hostname = EXCLUDED.hostname,
			last_seen = EXCLUDED.last_seen,
			is_online = TRUE
	`
	now := time.Now()
	_, err := s.db.Exec(query, peer.ID, peer.IP, peer.Port, peer.Hostname, now, now)
	return err
}

// GetPeer retrieves a peer by ID
func (s *DatabaseStorage) GetPeer(peerID string) (*models.Peer, bool) {
	query := `SELECT id, ip, port, hostname, registered_at, last_seen, is_online FROM peers WHERE id = $1`
	peer := &models.Peer{}
	err := s.db.QueryRow(query, peerID).Scan(
		&peer.ID, &peer.IP, &peer.Port, &peer.Hostname,
		&peer.RegisteredAt, &peer.LastSeen, &peer.IsOnline,
	)
	if err != nil {
		return nil, false
	}
	return peer, true
}

// UpdatePeerHeartbeat updates the last seen time
func (s *DatabaseStorage) UpdatePeerHeartbeat(peerID string) error {
	query := `UPDATE peers SET last_seen = $1, is_online = TRUE WHERE id = $2`
	_, err := s.db.Exec(query, time.Now(), peerID)
	return err
}

// RemovePeer removes a peer from the registry
func (s *DatabaseStorage) RemovePeer(peerID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove from file_peers first
	if _, err := tx.Exec(`DELETE FROM file_peers WHERE peer_id = $1`, peerID); err != nil {
		return err
	}
	// Remove peer
	if _, err := tx.Exec(`DELETE FROM peers WHERE id = $1`, peerID); err != nil {
		return err
	}
	return tx.Commit()
}

// CleanupOfflinePeers marks peers as offline if not seen recently
func (s *DatabaseStorage) CleanupOfflinePeers(timeout time.Duration) {
	cutoff := time.Now().Add(-timeout)
	s.db.Exec(`UPDATE peers SET is_online = FALSE WHERE last_seen < $1 AND is_online = TRUE`, cutoff)
}

// === File Operations ===

// AddFile adds a new file to the registry
func (s *DatabaseStorage) AddFile(file *models.File) error {
	chunksJSON, err := json.Marshal(file.Chunks)
	if err != nil {
		return err
	}
	tagsJSON, err := json.Marshal(file.Tags)
	if err != nil {
		return err
	}
	category := file.Category
	if category == "" {
		category = "other"
	}
	query := `
		INSERT INTO files (hash, name, size, chunk_size, chunks, category, tags, added_at, added_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT(hash) DO UPDATE SET
			name = EXCLUDED.name,
			size = EXCLUDED.size,
			chunk_size = EXCLUDED.chunk_size,
			chunks = EXCLUDED.chunks,
			category = EXCLUDED.category,
			tags = EXCLUDED.tags
	`
	_, err = s.db.Exec(query, file.Hash, file.Name, file.Size, file.ChunkSize, string(chunksJSON), category, string(tagsJSON), time.Now(), file.AddedBy)
	return err
}

// GetFile retrieves a file by hash
func (s *DatabaseStorage) GetFile(hash string) (*models.File, bool) {
	query := `SELECT hash, name, size, chunk_size, chunks, COALESCE(category, 'other'), COALESCE(tags, '[]'), added_at, added_by FROM files WHERE hash = $1`
	file := &models.File{}
	var chunksJSON, tagsJSON string
	err := s.db.QueryRow(query, hash).Scan(
		&file.Hash, &file.Name, &file.Size, &file.ChunkSize,
		&chunksJSON, &file.Category, &tagsJSON, &file.AddedAt, &file.AddedBy,
	)
	if err != nil {
		return nil, false
	}
	file.ID = file.Hash
	json.Unmarshal([]byte(chunksJSON), &file.Chunks)
	json.Unmarshal([]byte(tagsJSON), &file.Tags)
	return file, true
}

// ListFiles returns all files
func (s *DatabaseStorage) ListFiles() []protocol.FileListItem {
	query := `SELECT hash, name, size, added_at FROM files`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []protocol.FileListItem
	for rows.Next() {
		var item protocol.FileListItem
		if err := rows.Scan(&item.Hash, &item.Name, &item.Size, &item.AddedAt); err != nil {
			continue
		}
		item.Seeders, item.Leechers = s.countPeers(item.Hash)
		items = append(items, item)
	}
	return items
}

// countPeers counts seeders and leechers for a file
func (s *DatabaseStorage) countPeers(fileHash string) (seeders, leechers int) {
	query := `
		SELECT fp.is_seeder, COUNT(*)
		FROM file_peers fp
		JOIN peers p ON fp.peer_id = p.id
		WHERE fp.file_hash = $1 AND p.is_online = TRUE
		GROUP BY fp.is_seeder
	`
	rows, err := s.db.Query(query, fileHash)
	if err != nil {
		return 0, 0
	}
	defer rows.Close()

	for rows.Next() {
		var isSeeder bool
		var count int
		if err := rows.Scan(&isSeeder, &count); err != nil {
			continue
		}
		if isSeeder {
			seeders = count
		} else {
			leechers = count
		}
	}
	return
}

// === File-Peer Operations ===

// AddFilePeer associates a peer with a file
func (s *DatabaseStorage) AddFilePeer(fp *models.FilePeer) error {
	chunksJSON, err := json.Marshal(fp.ChunksAvailable)
	if err != nil {
		return err
	}
	query := `
		INSERT INTO file_peers (file_hash, peer_id, chunks_available, is_seeder, added_at, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT(file_hash, peer_id) DO UPDATE SET
			chunks_available = EXCLUDED.chunks_available,
			is_seeder = EXCLUDED.is_seeder,
			last_updated = EXCLUDED.last_updated
	`
	now := time.Now()
	_, err = s.db.Exec(query, fp.FileHash, fp.PeerID, string(chunksJSON), fp.IsSeeder, now, now)
	return err
}

// GetPeersForFile returns all peers that have a file
func (s *DatabaseStorage) GetPeersForFile(fileHash string) []protocol.PeerFileInfo {
	query := `
		SELECT p.id, p.ip, p.port, fp.chunks_available, fp.is_seeder
		FROM file_peers fp
		JOIN peers p ON fp.peer_id = p.id
		WHERE fp.file_hash = $1 AND p.is_online = TRUE
	`
	rows, err := s.db.Query(query, fileHash)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []protocol.PeerFileInfo
	for rows.Next() {
		var info protocol.PeerFileInfo
		var chunksJSON string
		if err := rows.Scan(&info.PeerID, &info.IP, &info.Port, &chunksJSON, &info.IsSeeder); err != nil {
			continue
		}
		json.Unmarshal([]byte(chunksJSON), &info.ChunksAvailable)
		result = append(result, info)
	}
	return result
}

// GetStats returns storage statistics
func (s *DatabaseStorage) GetStats() (peersOnline, peersTotal, filesCount int) {
	s.db.QueryRow(`SELECT COUNT(*) FROM peers WHERE is_online = TRUE`).Scan(&peersOnline)
	s.db.QueryRow(`SELECT COUNT(*) FROM peers`).Scan(&peersTotal)
	s.db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&filesCount)
	return
}

// Ping checks database connectivity
func (s *DatabaseStorage) Ping() error {
	return s.db.Ping()
}

// === Admin Operations ===

// ListAllPeers returns all peers
func (s *DatabaseStorage) ListAllPeers() []*models.Peer {
	query := `SELECT id, ip, port, hostname, is_online, registered_at, last_seen FROM peers ORDER BY last_seen DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var peers []*models.Peer
	for rows.Next() {
		p := &models.Peer{}
		if err := rows.Scan(&p.ID, &p.IP, &p.Port, &p.Hostname, &p.IsOnline, &p.RegisteredAt, &p.LastSeen); err != nil {
			continue
		}
		peers = append(peers, p)
	}
	return peers
}

// DeleteFile removes a file and its peer associations
func (s *DatabaseStorage) DeleteFile(hash string) error {
	// Delete file_peers first (foreign key)
	_, err := s.db.Exec(`DELETE FROM file_peers WHERE file_hash = $1`, hash)
	if err != nil {
		return err
	}
	// Delete file
	_, err = s.db.Exec(`DELETE FROM files WHERE hash = $1`, hash)
	return err
}

// SearchFiles searches files by name (case-insensitive)
func (s *DatabaseStorage) SearchFiles(query string) []protocol.FileListItem {
	sqlQuery := `SELECT hash, name, size, added_at FROM files WHERE LOWER(name) LIKE LOWER($1)`
	rows, err := s.db.Query(sqlQuery, "%"+query+"%")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []protocol.FileListItem
	for rows.Next() {
		var item protocol.FileListItem
		if err := rows.Scan(&item.Hash, &item.Name, &item.Size, &item.AddedAt); err != nil {
			continue
		}
		item.Seeders, item.Leechers = s.countPeers(item.Hash)
		items = append(items, item)
	}
	return items
}

// ListFilesByCategory returns files filtered by category
func (s *DatabaseStorage) ListFilesByCategory(category string) []protocol.FileListItem {
	sqlQuery := `SELECT hash, name, size, added_at FROM files WHERE LOWER(COALESCE(category, 'other')) = LOWER($1)`
	rows, err := s.db.Query(sqlQuery, category)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []protocol.FileListItem
	for rows.Next() {
		var item protocol.FileListItem
		if err := rows.Scan(&item.Hash, &item.Name, &item.Size, &item.AddedAt); err != nil {
			continue
		}
		item.Seeders, item.Leechers = s.countPeers(item.Hash)
		items = append(items, item)
	}
	return items
}

// ListCategories returns statistics for all categories
func (s *DatabaseStorage) ListCategories() []CategoryStats {
	sqlQuery := `SELECT COALESCE(category, 'other') as cat, COUNT(*) as cnt, COALESCE(SUM(size), 0) as total_size FROM files GROUP BY cat ORDER BY cnt DESC`
	rows, err := s.db.Query(sqlQuery)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []CategoryStats
	for rows.Next() {
		var cs CategoryStats
		if err := rows.Scan(&cs.Category, &cs.FileCount, &cs.TotalSize); err != nil {
			continue
		}
		result = append(result, cs)
	}
	return result
}

// === Reputation Operations ===

// UpdatePeerStats updates peer upload/download statistics and recalculates reputation
func (s *DatabaseStorage) UpdatePeerStats(peerID string, bytesUploaded, bytesDownloaded int64) error {
	// First update the stats
	query := `
		UPDATE peers SET
			bytes_uploaded = COALESCE(bytes_uploaded, 0) + $1,
			bytes_downloaded = COALESCE(bytes_downloaded, 0) + $2
		WHERE id = $3
	`
	_, err := s.db.Exec(query, bytesUploaded, bytesDownloaded, peerID)
	if err != nil {
		return err
	}

	// Then recalculate reputation
	reputationQuery := `
		UPDATE peers SET reputation =
			LEAST(100, GREATEST(0,
				50 +
				CASE
					WHEN COALESCE(bytes_downloaded, 0) > 0 THEN
						LEAST(30, (COALESCE(bytes_uploaded, 0)::float / bytes_downloaded) * 10)
					WHEN COALESCE(bytes_uploaded, 0) > 0 THEN 30
					ELSE 0
				END +
				LEAST(10, COALESCE(files_shared, 0) * 2)
			))
		WHERE id = $1
	`
	_, err = s.db.Exec(reputationQuery, peerID)
	return err
}

// GetTopPeers returns top peers by reputation
func (s *DatabaseStorage) GetTopPeers(limit int) []*models.Peer {
	query := `
		SELECT id, ip, port, hostname, is_online, registered_at, last_seen,
			   COALESCE(bytes_uploaded, 0), COALESCE(bytes_downloaded, 0),
			   COALESCE(files_shared, 0), COALESCE(reputation, 50)
		FROM peers
		WHERE is_online = TRUE
		ORDER BY reputation DESC
		LIMIT $1
	`
	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var peers []*models.Peer
	for rows.Next() {
		p := &models.Peer{}
		if err := rows.Scan(&p.ID, &p.IP, &p.Port, &p.Hostname, &p.IsOnline,
			&p.RegisteredAt, &p.LastSeen, &p.BytesUploaded, &p.BytesDownloaded,
			&p.FilesShared, &p.Reputation); err != nil {
			continue
		}
		peers = append(peers, p)
	}
	return peers
}
