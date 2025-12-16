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

// PostgresStorage implements Storage interface with PostgreSQL
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgresStorage creates a new PostgreSQL storage
func NewPostgresStorage(connStr string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	storage := &PostgresStorage{db: db}
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %w", err)
	}

	return storage, nil
}

// initSchema creates the database tables
func (s *PostgresStorage) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS peers (
		id VARCHAR(255) PRIMARY KEY,
		ip VARCHAR(45) NOT NULL,
		port INTEGER NOT NULL,
		hostname VARCHAR(255),
		registered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		is_online BOOLEAN DEFAULT TRUE,
		bytes_uploaded BIGINT DEFAULT 0,
		bytes_downloaded BIGINT DEFAULT 0,
		files_shared INTEGER DEFAULT 0,
		reputation DECIMAL(5,2) DEFAULT 50.0
	);

	CREATE TABLE IF NOT EXISTS files (
		id VARCHAR(255) PRIMARY KEY,
		hash VARCHAR(64) UNIQUE NOT NULL,
		name VARCHAR(512) NOT NULL,
		size BIGINT NOT NULL,
		chunk_size BIGINT NOT NULL,
		chunks JSONB,
		category VARCHAR(50),
		tags JSONB,
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		added_by VARCHAR(255)
	);

	CREATE TABLE IF NOT EXISTS file_peers (
		file_hash VARCHAR(64) NOT NULL,
		peer_id VARCHAR(255) NOT NULL,
		chunks_available JSONB,
		is_seeder BOOLEAN DEFAULT FALSE,
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (file_hash, peer_id)
	);

	CREATE INDEX IF NOT EXISTS idx_peers_online ON peers(is_online);
	CREATE INDEX IF NOT EXISTS idx_peers_last_seen ON peers(last_seen);
	CREATE INDEX IF NOT EXISTS idx_files_category ON files(category);
	CREATE INDEX IF NOT EXISTS idx_files_name ON files(name);
	CREATE INDEX IF NOT EXISTS idx_file_peers_file ON file_peers(file_hash);
	CREATE INDEX IF NOT EXISTS idx_file_peers_peer ON file_peers(peer_id);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection
func (s *PostgresStorage) Close() error {
	return s.db.Close()
}

// === Peer Operations ===

func (s *PostgresStorage) RegisterPeer(peer *models.Peer) error {
	query := `
		INSERT INTO peers (id, ip, port, hostname, registered_at, last_seen, is_online, 
			bytes_uploaded, bytes_downloaded, files_shared, reputation)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			ip = EXCLUDED.ip,
			port = EXCLUDED.port,
			hostname = EXCLUDED.hostname,
			last_seen = EXCLUDED.last_seen,
			is_online = TRUE
	`
	now := time.Now()
	_, err := s.db.Exec(query, peer.ID, peer.IP, peer.Port, peer.Hostname,
		now, now, true, peer.BytesUploaded, peer.BytesDownloaded, peer.FilesShared, 50.0)
	return err
}

func (s *PostgresStorage) GetPeer(peerID string) (*models.Peer, bool) {
	query := `SELECT id, ip, port, hostname, registered_at, last_seen, is_online,
		bytes_uploaded, bytes_downloaded, files_shared, reputation FROM peers WHERE id = $1`

	peer := &models.Peer{}
	err := s.db.QueryRow(query, peerID).Scan(
		&peer.ID, &peer.IP, &peer.Port, &peer.Hostname,
		&peer.RegisteredAt, &peer.LastSeen, &peer.IsOnline,
		&peer.BytesUploaded, &peer.BytesDownloaded, &peer.FilesShared, &peer.Reputation,
	)
	if err != nil {
		return nil, false
	}
	return peer, true
}

func (s *PostgresStorage) UpdatePeerHeartbeat(peerID string) error {
	query := `UPDATE peers SET last_seen = $1, is_online = TRUE WHERE id = $2`
	_, err := s.db.Exec(query, time.Now(), peerID)
	return err
}

func (s *PostgresStorage) RemovePeer(peerID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove file-peer associations
	if _, err := tx.Exec(`DELETE FROM file_peers WHERE peer_id = $1`, peerID); err != nil {
		return err
	}
	// Remove peer
	if _, err := tx.Exec(`DELETE FROM peers WHERE id = $1`, peerID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *PostgresStorage) CleanupOfflinePeers(timeout time.Duration) {
	cutoff := time.Now().Add(-timeout)
	s.db.Exec(`UPDATE peers SET is_online = FALSE WHERE last_seen < $1`, cutoff)
}

// === File Operations ===

func (s *PostgresStorage) AddFile(file *models.File) error {
	chunksJSON, _ := json.Marshal(file.Chunks)
	tagsJSON, _ := json.Marshal(file.Tags)

	category := file.Category
	if category == "" {
		category = "other"
	}

	query := `
		INSERT INTO files (hash, name, size, chunk_size, chunks, category, tags, added_at, added_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (hash) DO UPDATE SET
			name = EXCLUDED.name,
			chunks = EXCLUDED.chunks,
			category = EXCLUDED.category,
			tags = EXCLUDED.tags
	`
	_, err := s.db.Exec(query, file.Hash, file.Name, file.Size, file.ChunkSize,
		string(chunksJSON), category, string(tagsJSON), time.Now(), file.AddedBy)
	return err
}

func (s *PostgresStorage) GetFile(hash string) (*models.File, bool) {
	query := `SELECT hash, name, size, chunk_size, chunks, category, tags, added_at, added_by
		FROM files WHERE hash = $1`

	file := &models.File{}
	var chunksJSON, tagsJSON string
	var category, addedBy sql.NullString
	var addedAt sql.NullTime

	err := s.db.QueryRow(query, hash).Scan(
		&file.Hash, &file.Name, &file.Size, &file.ChunkSize,
		&chunksJSON, &category, &tagsJSON, &addedAt, &addedBy,
	)
	if err != nil {
		return nil, false
	}

	file.ID = file.Hash // Use hash as ID
	if category.Valid {
		file.Category = category.String
	}
	if addedBy.Valid {
		file.AddedBy = addedBy.String
	}
	if addedAt.Valid {
		file.AddedAt = addedAt.Time
	}

	json.Unmarshal([]byte(chunksJSON), &file.Chunks)
	json.Unmarshal([]byte(tagsJSON), &file.Tags)
	return file, true
}

func (s *PostgresStorage) ListFiles() []protocol.FileListItem {
	query := `SELECT f.hash, f.name, f.size, f.added_at,
		COALESCE(SUM(CASE WHEN fp.is_seeder AND p.is_online THEN 1 ELSE 0 END), 0) as seeders,
		COALESCE(SUM(CASE WHEN NOT fp.is_seeder AND p.is_online THEN 1 ELSE 0 END), 0) as leechers
		FROM files f
		LEFT JOIN file_peers fp ON f.hash = fp.file_hash
		LEFT JOIN peers p ON fp.peer_id = p.id
		GROUP BY f.hash, f.name, f.size, f.added_at`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []protocol.FileListItem
	for rows.Next() {
		var item protocol.FileListItem
		rows.Scan(&item.Hash, &item.Name, &item.Size, &item.AddedAt, &item.Seeders, &item.Leechers)
		items = append(items, item)
	}
	return items
}

func (s *PostgresStorage) SearchFiles(query string) []protocol.FileListItem {
	sqlQuery := `SELECT f.hash, f.name, f.size, f.added_at,
		COALESCE(SUM(CASE WHEN fp.is_seeder AND p.is_online THEN 1 ELSE 0 END), 0) as seeders,
		COALESCE(SUM(CASE WHEN NOT fp.is_seeder AND p.is_online THEN 1 ELSE 0 END), 0) as leechers
		FROM files f
		LEFT JOIN file_peers fp ON f.hash = fp.file_hash
		LEFT JOIN peers p ON fp.peer_id = p.id
		WHERE LOWER(f.name) LIKE LOWER($1)
		GROUP BY f.hash, f.name, f.size, f.added_at`

	rows, err := s.db.Query(sqlQuery, "%"+query+"%")
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []protocol.FileListItem
	for rows.Next() {
		var item protocol.FileListItem
		rows.Scan(&item.Hash, &item.Name, &item.Size, &item.AddedAt, &item.Seeders, &item.Leechers)
		items = append(items, item)
	}
	return items
}

func (s *PostgresStorage) ListFilesByCategory(category string) []protocol.FileListItem {
	sqlQuery := `SELECT f.hash, f.name, f.size, f.added_at,
		COALESCE(SUM(CASE WHEN fp.is_seeder AND p.is_online THEN 1 ELSE 0 END), 0) as seeders,
		COALESCE(SUM(CASE WHEN NOT fp.is_seeder AND p.is_online THEN 1 ELSE 0 END), 0) as leechers
		FROM files f
		LEFT JOIN file_peers fp ON f.hash = fp.file_hash
		LEFT JOIN peers p ON fp.peer_id = p.id
		WHERE LOWER(f.category) = LOWER($1)
		GROUP BY f.hash, f.name, f.size, f.added_at`

	rows, err := s.db.Query(sqlQuery, category)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var items []protocol.FileListItem
	for rows.Next() {
		var item protocol.FileListItem
		rows.Scan(&item.Hash, &item.Name, &item.Size, &item.AddedAt, &item.Seeders, &item.Leechers)
		items = append(items, item)
	}
	return items
}

func (s *PostgresStorage) ListCategories() []CategoryStats {
	query := `SELECT COALESCE(NULLIF(category, ''), 'other') as cat, COUNT(*) as cnt,
		COALESCE(SUM(size), 0) as total FROM files GROUP BY cat`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var stats []CategoryStats
	for rows.Next() {
		var s CategoryStats
		rows.Scan(&s.Category, &s.FileCount, &s.TotalSize)
		stats = append(stats, s)
	}
	return stats
}

// === File-Peer Operations ===

func (s *PostgresStorage) AddFilePeer(fp *models.FilePeer) error {
	chunksJSON, _ := json.Marshal(fp.ChunksAvailable)
	query := `
		INSERT INTO file_peers (file_hash, peer_id, chunks_available, is_seeder, added_at, last_updated)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (file_hash, peer_id) DO UPDATE SET
			chunks_available = EXCLUDED.chunks_available,
			is_seeder = EXCLUDED.is_seeder,
			last_updated = EXCLUDED.last_updated
	`
	now := time.Now()
	_, err := s.db.Exec(query, fp.FileHash, fp.PeerID, chunksJSON, fp.IsSeeder, now, now)
	return err
}

func (s *PostgresStorage) GetPeersForFile(fileHash string) []protocol.PeerFileInfo {
	query := `SELECT p.id, p.ip, p.port, fp.chunks_available, fp.is_seeder
		FROM file_peers fp
		JOIN peers p ON fp.peer_id = p.id
		WHERE fp.file_hash = $1 AND p.is_online = TRUE`

	rows, err := s.db.Query(query, fileHash)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var result []protocol.PeerFileInfo
	for rows.Next() {
		var info protocol.PeerFileInfo
		var chunksJSON []byte
		rows.Scan(&info.PeerID, &info.IP, &info.Port, &chunksJSON, &info.IsSeeder)
		json.Unmarshal(chunksJSON, &info.ChunksAvailable)
		result = append(result, info)
	}
	return result
}

// === Stats ===

func (s *PostgresStorage) GetStats() (peersOnline, peersTotal, filesCount int) {
	s.db.QueryRow(`SELECT COUNT(*) FROM peers WHERE is_online = TRUE`).Scan(&peersOnline)
	s.db.QueryRow(`SELECT COUNT(*) FROM peers`).Scan(&peersTotal)
	s.db.QueryRow(`SELECT COUNT(*) FROM files`).Scan(&filesCount)
	return
}

// === Admin Operations ===

func (s *PostgresStorage) ListAllPeers() []*models.Peer {
	query := `SELECT id, ip, port, hostname, registered_at, last_seen, is_online,
		bytes_uploaded, bytes_downloaded, files_shared, reputation FROM peers`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var peers []*models.Peer
	for rows.Next() {
		peer := &models.Peer{}
		rows.Scan(&peer.ID, &peer.IP, &peer.Port, &peer.Hostname,
			&peer.RegisteredAt, &peer.LastSeen, &peer.IsOnline,
			&peer.BytesUploaded, &peer.BytesDownloaded, &peer.FilesShared, &peer.Reputation)
		peers = append(peers, peer)
	}
	return peers
}

func (s *PostgresStorage) DeleteFile(hash string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM file_peers WHERE file_hash = $1`, hash); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM files WHERE hash = $1`, hash); err != nil {
		return err
	}
	return tx.Commit()
}

// === Reputation Operations ===

func (s *PostgresStorage) UpdatePeerStats(peerID string, bytesUploaded, bytesDownloaded int64) error {
	query := `UPDATE peers SET
		bytes_uploaded = bytes_uploaded + $1,
		bytes_downloaded = bytes_downloaded + $2,
		reputation = LEAST(100, GREATEST(0, 50 +
			CASE WHEN bytes_downloaded + $2 > 0
			THEN LEAST(30, ((bytes_uploaded + $1)::float / (bytes_downloaded + $2)::float) * 10)
			ELSE 30 END))
		WHERE id = $3`
	_, err := s.db.Exec(query, bytesUploaded, bytesDownloaded, peerID)
	return err
}

func (s *PostgresStorage) GetTopPeers(limit int) []*models.Peer {
	query := `SELECT id, ip, port, hostname, registered_at, last_seen, is_online,
		bytes_uploaded, bytes_downloaded, files_shared, reputation
		FROM peers WHERE is_online = TRUE ORDER BY reputation DESC LIMIT $1`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var peers []*models.Peer
	for rows.Next() {
		peer := &models.Peer{}
		rows.Scan(&peer.ID, &peer.IP, &peer.Port, &peer.Hostname,
			&peer.RegisteredAt, &peer.LastSeen, &peer.IsOnline,
			&peer.BytesUploaded, &peer.BytesDownloaded, &peer.FilesShared, &peer.Reputation)
		peers = append(peers, peer)
	}
	return peers
}

// Ensure PostgresStorage implements Storage interface
var _ Storage = (*PostgresStorage)(nil)
