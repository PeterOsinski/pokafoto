package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type UploadJobStore struct {
	db *DB
}

func NewUploadJobStore(db *DB) *UploadJobStore {
	return &UploadJobStore{db: db}
}

func (s *UploadJobStore) Create(job *model.UploadJob) error {
	job.ID = uuid.New().String()
	job.CreatedAt = time.Now().UTC()
	job.UpdatedAt = time.Now().UTC()

	_, err := s.db.Exec(
		`INSERT INTO upload_jobs (id, batch_id, user_id, filename, size_bytes, temp_path, folder_id, skip_name_size_dedup, status, stage, progress, error, reason, file_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.BatchID, job.UserID, job.Filename, job.SizeBytes, job.TempPath, job.FolderID, boolToInt(job.SkipNameSizeDedup), string(job.Status), nil, job.Progress, nil, nil, nil, job.CreatedAt.Format(time.RFC3339), job.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert upload job: %w", err)
	}

	return nil
}

func (s *UploadJobStore) Claim() (*model.UploadJob, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}

		job, err := s.claimOnce()
		if err == nil {
			return job, nil
		}
		if !isSQLiteBusy(err) {
			return nil, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("claim failed after %d retries: %w", maxRetries, lastErr)
}

func (s *UploadJobStore) claimOnce() (*model.UploadJob, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin claim tx: %w", err)
	}
	defer tx.Rollback()

	var id string
	err = tx.QueryRow(
		`SELECT id FROM upload_jobs WHERE status = 'queued' ORDER BY created_at ASC LIMIT 1`,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select queued job: %w", err)
	}

	result, err := tx.Exec(
		`UPDATE upload_jobs SET status = 'processing', updated_at = ? WHERE id = ? AND status = 'queued'`,
		time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return nil, fmt.Errorf("claim job: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return nil, nil
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claim: %w", err)
	}

	return s.FindByID(id)
}

func isSQLiteBusy(err error) bool {
	return err != nil && strings.Contains(err.Error(), "SQLITE_BUSY")
}

func (s *UploadJobStore) FindByID(id string) (*model.UploadJob, error) {
	job := &model.UploadJob{}
	var stage, errorStr, reasonStr, fileID, folderID sql.NullString
	var createdAt, updatedAt string
	var skipNameSizeDedup int

	err := s.db.QueryRow(
		`SELECT id, batch_id, user_id, filename, size_bytes, temp_path, folder_id, skip_name_size_dedup, status, stage, progress, error, reason, file_id, created_at, updated_at FROM upload_jobs WHERE id = ?`,
		id,
	).Scan(&job.ID, &job.BatchID, &job.UserID, &job.Filename, &job.SizeBytes, &job.TempPath, &folderID, &skipNameSizeDedup, &job.Status, &stage, &job.Progress, &errorStr, &reasonStr, &fileID, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find upload job: %w", err)
	}

	if stage.Valid {
		s := model.JobStage(stage.String)
		job.Stage = &s
	}
	if errorStr.Valid {
		job.Error = &errorStr.String
	}
	if reasonStr.Valid {
		job.Reason = &reasonStr.String
	}
	if fileID.Valid {
		job.FileID = &fileID.String
	}
	if folderID.Valid {
		job.FolderID = &folderID.String
	}
	job.SkipNameSizeDedup = skipNameSizeDedup == 1
	job.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	job.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return job, nil
}

func (s *UploadJobStore) UpdateProgress(id string, stage model.JobStage, progress float64) error {
	_, err := s.db.Exec(
		`UPDATE upload_jobs SET stage = ?, progress = ?, updated_at = ? WHERE id = ?`,
		string(stage), progress, time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("update progress: %w", err)
	}
	return nil
}

func (s *UploadJobStore) Complete(id, fileID string) error {
	_, err := s.db.Exec(
		`UPDATE upload_jobs SET status = 'completed', file_id = ?, progress = 1.0, updated_at = ? WHERE id = ?`,
		fileID, time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("complete job: %w", err)
	}
	return nil
}

func (s *UploadJobStore) Fail(id, errorMsg string) error {
	_, err := s.db.Exec(
		`UPDATE upload_jobs SET status = 'failed', error = ?, updated_at = ? WHERE id = ?`,
		errorMsg, time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("fail job: %w", err)
	}
	return nil
}

func (s *UploadJobStore) Skip(id, reason, fileID string) error {
	_, err := s.db.Exec(
		`UPDATE upload_jobs SET status = 'skipped', reason = ?, file_id = ?, progress = 1.0, updated_at = ? WHERE id = ?`,
		reason, fileID, time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("skip job: %w", err)
	}
	return nil
}

func (s *UploadJobStore) SetProcessing(id string) error {
	_, err := s.db.Exec(
		`UPDATE upload_jobs SET status = 'processing', updated_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("set processing: %w", err)
	}
	return nil
}

func (s *UploadJobStore) ListByBatch(batchID string) ([]*model.UploadJob, error) {
	rows, err := s.db.Query(
		`SELECT id, batch_id, user_id, filename, size_bytes, temp_path, folder_id, skip_name_size_dedup, status, stage, progress, error, reason, file_id, created_at, updated_at FROM upload_jobs WHERE batch_id = ? ORDER BY created_at`,
		batchID,
	)
	if err != nil {
		return nil, fmt.Errorf("list upload jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*model.UploadJob
	for rows.Next() {
		job, err := s.scanJob(rows)
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

func (s *UploadJobStore) CountProcessing() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM upload_jobs WHERE status = 'processing'`).Scan(&count)
	return count, err
}

func (s *UploadJobStore) RecoverStuckJobs() (int64, error) {
	result, err := s.db.Exec(
		`UPDATE upload_jobs SET status = 'queued', updated_at = ? WHERE status = 'processing'`,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return 0, fmt.Errorf("recover stuck jobs: %w", err)
	}
	affected, _ := result.RowsAffected()
	return affected, nil
}

func (s *UploadJobStore) ListActiveByUser(userID string) ([]*model.UploadJob, error) {
	cutoff := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339)
	rows, err := s.db.Query(
		`SELECT id, batch_id, user_id, filename, size_bytes, temp_path, folder_id, skip_name_size_dedup, status, stage, progress, error, reason, file_id, created_at, updated_at FROM upload_jobs WHERE user_id = ? AND (status IN ('queued','processing') OR (status IN ('completed','skipped','failed') AND updated_at > ?)) ORDER BY created_at DESC`,
		userID, cutoff,
	)
	if err != nil {
		return nil, fmt.Errorf("list active by user: %w", err)
	}
	defer rows.Close()

	var jobs []*model.UploadJob
	for rows.Next() {
		job, err := s.scanJob(rows)
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}

func (s *UploadJobStore) DeleteByID(id string) error {
	_, err := s.db.Exec(`DELETE FROM upload_jobs WHERE id = ?`, id)
	return err
}

func (s *UploadJobStore) ListAll(limit, offset int, statusFilter string) ([]*model.UploadJob, int, error) {
	var total int
	query := `SELECT COUNT(*) FROM upload_jobs`
	args := []interface{}{}

	if statusFilter != "" {
		query += ` WHERE status = ?`
		args = append(args, statusFilter)
	}

	if err := s.db.QueryRow(query, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count upload jobs: %w", err)
	}

	dataQuery := `SELECT id, batch_id, user_id, filename, size_bytes, temp_path, folder_id, skip_name_size_dedup, status, stage, progress, error, reason, file_id, created_at, updated_at FROM upload_jobs`
	dataArgs := []interface{}{}

	if statusFilter != "" {
		dataQuery += ` WHERE status = ?`
		dataArgs = append(dataArgs, statusFilter)
	}

	dataQuery += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	dataArgs = append(dataArgs, limit, offset)

	rows, err := s.db.Query(dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list upload jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*model.UploadJob
	for rows.Next() {
		job, err := s.scanJob(rows)
		if err != nil {
			continue
		}
		jobs = append(jobs, job)
	}

	return jobs, total, rows.Err()
}

func (s *UploadJobStore) CountByStatus() (map[string]int, error) {
	rows, err := s.db.Query(`SELECT status, COUNT(*) FROM upload_jobs GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("count by status: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{
		"queued":     0,
		"processing": 0,
		"completed":  0,
		"failed":     0,
		"skipped":    0,
	}

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		counts[status] = count
	}

	return counts, rows.Err()
}

func (s *UploadJobStore) Requeue(id string) error {
	result, err := s.db.Exec(
		`UPDATE upload_jobs SET status = 'queued', error = NULL, reason = NULL, progress = 0.0, stage = NULL, updated_at = ? WHERE id = ? AND status IN ('failed', 'skipped')`,
		time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("requeue job: %w", err)
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("job not found or not in retryable state")
	}
	return nil
}

type scannableRow interface {
	Scan(dest ...interface{}) error
}

func (s *UploadJobStore) scanJob(row scannableRow) (*model.UploadJob, error) {
	job := &model.UploadJob{}
	var stage, errorStr, reasonStr, fileID, folderID sql.NullString
	var createdAt, updatedAt string
	var skipNameSizeDedup int

	err := row.Scan(&job.ID, &job.BatchID, &job.UserID, &job.Filename, &job.SizeBytes, &job.TempPath, &folderID, &skipNameSizeDedup, &job.Status, &stage, &job.Progress, &errorStr, &reasonStr, &fileID, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan upload job: %w", err)
	}

	if stage.Valid {
		s := model.JobStage(stage.String)
		job.Stage = &s
	}
	if errorStr.Valid {
		job.Error = &errorStr.String
	}
	if reasonStr.Valid {
		job.Reason = &reasonStr.String
	}
	if fileID.Valid {
		job.FileID = &fileID.String
	}
	if folderID.Valid {
		job.FolderID = &folderID.String
	}
	job.SkipNameSizeDedup = skipNameSizeDedup == 1
	job.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	job.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return job, nil
}
