package store

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type chunkFS interface {
	Create(name string) (*os.File, error)
	Open(name string) (*os.File, error)
	Remove(name string) error
	Stat(name string) (os.FileInfo, error)
}

type ChunkStore struct {
	db *DB
	fs chunkFS
}

func NewChunkStore(db *DB, fs chunkFS) *ChunkStore {
	return &ChunkStore{db: db, fs: fs}
}

func (s *ChunkStore) CreateChunkRecord(uploadID string, index int, size, offset int64, sha256hex, tempPath string) error {
	var sha256hexPtr *string
	if sha256hex != "" {
		sha256hexPtr = &sha256hex
	}

	_, err := s.db.Exec(
		`INSERT INTO upload_chunks (upload_id, chunk_index, chunk_size, offset, status, chunk_sha256, temp_path) VALUES (?, ?, ?, ?, 'stored', ?, ?)`,
		uploadID, index, size, offset, sha256hexPtr, tempPath,
	)
	if err != nil {
		return fmt.Errorf("create chunk record: %w", err)
	}
	return nil
}

func (s *ChunkStore) GetStoredChunks(uploadID string) ([]int, error) {
	rows, err := s.db.Query(
		`SELECT chunk_index FROM upload_chunks WHERE upload_id = ? AND status = 'stored' ORDER BY chunk_index ASC`,
		uploadID,
	)
	if err != nil {
		return nil, fmt.Errorf("get stored chunks: %w", err)
	}
	defer rows.Close()

	var indices []int
	for rows.Next() {
		var i int
		if err := rows.Scan(&i); err != nil {
			continue
		}
		indices = append(indices, i)
	}
	return indices, rows.Err()
}

func (s *ChunkStore) GetStoredChunkCount(uploadID string) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM upload_chunks WHERE upload_id = ? AND status = 'stored'`,
		uploadID,
	).Scan(&count)
	return count, err
}

func (s *ChunkStore) GetChunkPath(uploadID string, index int) (string, error) {
	var path string
	err := s.db.QueryRow(
		`SELECT temp_path FROM upload_chunks WHERE upload_id = ? AND chunk_index = ?`,
		uploadID, index,
	).Scan(&path)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get chunk path: %w", err)
	}
	return path, nil
}

func (s *ChunkStore) FindMissingChunks(uploadID string, totalChunks int) ([]int, error) {
	rows, err := s.db.Query(
		`SELECT chunk_index FROM upload_chunks WHERE upload_id = ? AND status = 'stored' ORDER BY chunk_index ASC`,
		uploadID,
	)
	if err != nil {
		return nil, fmt.Errorf("find missing chunks: %w", err)
	}
	defer rows.Close()

	stored := make(map[int]bool)
	for rows.Next() {
		var i int
		if err := rows.Scan(&i); err != nil {
			continue
		}
		stored[i] = true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var missing []int
	for i := 0; i < totalChunks; i++ {
		if !stored[i] {
			missing = append(missing, i)
		}
	}
	return missing, nil
}

func (s *ChunkStore) AssembleFile(uploadID string, totalChunks int, destPath string) (string, error) {
	destFile, err := s.fs.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create assembled file: %w", err)
	}
	defer destFile.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(destFile, hasher)

	for i := 0; i < totalChunks; i++ {
		chunkPath, err := s.GetChunkPath(uploadID, i)
		if err != nil {
			return "", fmt.Errorf("get chunk path %d: %w", i, err)
		}
		if chunkPath == "" {
			return "", fmt.Errorf("chunk %d missing for upload %s", i, uploadID)
		}

		chunkFile, err := s.fs.Open(chunkPath)
		if err != nil {
			return "", fmt.Errorf("open chunk %d: %w", i, err)
		}

		if _, err := io.Copy(writer, chunkFile); err != nil {
			chunkFile.Close()
			return "", fmt.Errorf("copy chunk %d: %w", i, err)
		}
		chunkFile.Close()
	}

	if err := destFile.Sync(); err != nil {
		return "", fmt.Errorf("sync assembled file: %w", err)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func (s *ChunkStore) DeleteChunks(uploadID string) error {
	rows, err := s.db.Query(
		`SELECT temp_path FROM upload_chunks WHERE upload_id = ?`,
		uploadID,
	)
	if err != nil {
		return fmt.Errorf("query chunks for deletion: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			continue
		}
		paths = append(paths, p)
	}

	for _, p := range paths {
		if p != "" {
			s.fs.Remove(p)
		}
	}

	_, err = s.db.Exec(`DELETE FROM upload_chunks WHERE upload_id = ?`, uploadID)
	if err != nil {
		return fmt.Errorf("delete chunk records: %w", err)
	}
	return nil
}

func (s *ChunkStore) DeleteAbandonedChunks(maxAgeHours int) (int64, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(maxAgeHours) * time.Hour).Format(time.RFC3339)

	rows, err := s.db.Query(
		`SELECT uc.temp_path FROM upload_chunks uc
		 INNER JOIN upload_jobs uj ON uc.upload_id = uj.id
		 WHERE uj.status IN ('failed','completed','skipped')
		   AND uj.updated_at < ?`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("query abandoned chunks: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			continue
		}
		if p != "" {
			paths = append(paths, p)
		}
	}

	for _, p := range paths {
		s.fs.Remove(p)
	}

	result, err := s.db.Exec(
		`DELETE FROM upload_chunks WHERE upload_id IN
		 (SELECT id FROM upload_jobs WHERE status IN ('failed','completed','skipped') AND updated_at < ?)`,
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("delete abandoned chunks: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected > 0 {
		slog.Info("cleaned up abandoned upload chunks", "deleted", affected)
	}
	return affected, nil
}

func (s *ChunkStore) CleanupOrphanedTempFiles(uploadID string) {
	rows, err := s.db.Query(
		`SELECT temp_path FROM upload_chunks WHERE upload_id = ?`,
		uploadID,
	)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			continue
		}
		if p != "" {
			if _, err := s.fs.Stat(p); err == nil {
				s.fs.Remove(p)
			}
		}
	}
}

func (s *ChunkStore) CleanupOldUploads(maxAgeHours int) ([]string, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(maxAgeHours) * time.Hour).Format(time.RFC3339)

	rows, err := s.db.Query(
		`SELECT id, temp_path FROM upload_jobs
		 WHERE upload_mode = 'chunked'
		   AND status IN ('queued','ready','processing')
		   AND updated_at < ?`,
		cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("query old chunked uploads: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id, tempPath string
		if err := rows.Scan(&id, &tempPath); err != nil {
			continue
		}
		ids = append(ids, id)
	}

	for _, id := range ids {
		s.CleanupOrphanedTempFiles(id)
		s.db.Exec(`DELETE FROM upload_chunks WHERE upload_id = ?`, id)
		s.db.Exec(`UPDATE upload_jobs SET status = 'failed', error = 'upload_expired', updated_at = ? WHERE id = ?`,
			time.Now().UTC().Format(time.RFC3339), id)
	}

	if len(ids) > 0 {
		slog.Info("cleaned up expired chunked uploads", "count", len(ids))
	}

	return ids, nil
}

func ChunkTempDir(originalsDir string) string {
	return filepath.Join(originalsDir, "..", "tmp", "chunks")
}
