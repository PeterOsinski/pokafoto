package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type FileStore struct {
	db *DB
}

func NewFileStore(db *DB) *FileStore {
	return &FileStore{db: db}
}

func (s *FileStore) Create(file *model.File) error {
	file.ID = uuid.New().String()
	file.CreatedAt = time.Now().UTC()
	file.UpdatedAt = time.Now().UTC()

	_, err := s.db.Exec(
		`INSERT INTO files (id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, created_at, updated_at, is_deleted) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		file.ID, file.UserID, file.Filename, file.OriginalName, file.Path, file.SizeBytes, file.MimeType, file.SHA256, file.MediaType, file.Width, file.Height, file.DurationSec, file.TakenAt, file.CreatedAt.Format(time.RFC3339), file.UpdatedAt.Format(time.RFC3339), boolToInt(file.IsDeleted),
	)
	if err != nil {
		return fmt.Errorf("insert file: %w", err)
	}

	return nil
}

func (s *FileStore) FindByID(id string) (*model.File, error) {
	return s.scanFile(s.db.QueryRow(
		`SELECT id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, created_at, updated_at, is_deleted FROM files WHERE id = ?`,
		id,
	))
}

func (s *FileStore) FindBySHA256(sha256 string) (*model.File, error) {
	return s.scanFile(s.db.QueryRow(
		`SELECT id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, created_at, updated_at, is_deleted FROM files WHERE sha256 = ?`,
		sha256,
	))
}

func (s *FileStore) FindByNameAndSize(name string, size int64) (*model.File, error) {
	return s.scanFile(s.db.QueryRow(
		`SELECT id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, created_at, updated_at, is_deleted FROM files WHERE original_name = ? AND size_bytes = ? AND is_deleted = 0 LIMIT 1`,
		name, size,
	))
}

type FileListOptions struct {
	UserID    string
	Path      string
	Cursor    string
	Limit     int
	Sort      string
	Order     string
	MediaType string
	DateFrom  string
	DateTo    string
	Camera    string
}

func (s *FileStore) List(opts FileListOptions) ([]*model.File, string, int, error) {
	if opts.Limit <= 0 || opts.Limit > 500 {
		opts.Limit = 100
	}
	if opts.Sort == "" {
		opts.Sort = "taken_at"
	}
	if opts.Order == "" {
		opts.Order = "desc"
	}

	var conditions []string
	var args []interface{}

	conditions = append(conditions, "user_id = ?")
	args = append(args, opts.UserID)

	conditions = append(conditions, "is_deleted = 0")

	if opts.Path != "" {
		conditions = append(conditions, "path = ?")
		args = append(args, opts.Path)
	}

	if opts.MediaType != "" {
		conditions = append(conditions, "media_type = ?")
		args = append(args, opts.MediaType)
	}

	if opts.DateFrom != "" {
		conditions = append(conditions, "taken_at >= ?")
		args = append(args, opts.DateFrom)
	}

	if opts.DateTo != "" {
		conditions = append(conditions, "taken_at <= ?")
		args = append(args, opts.DateTo)
	}

	if opts.Camera != "" {
		conditions = append(conditions, "exif.camera_model LIKE ?")
		args = append(args, "%"+opts.Camera+"%")
	}

	whereClause := strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM files WHERE %s`, whereClause)
	if opts.Camera != "" {
		countQuery = fmt.Sprintf(`SELECT COUNT(*) FROM files LEFT JOIN exif ON files.id = exif.file_id WHERE %s`, whereClause)
	}
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, "", 0, fmt.Errorf("count files: %w", err)
	}

	if opts.Cursor != "" {
		cmp := ">"
		if opts.Order == "desc" {
			cmp = "<"
		}
		conditions = append(conditions, fmt.Sprintf("files.id %s ?", cmp))
		args = append(args, opts.Cursor)
	}

	orderBy := fmt.Sprintf("files.%s", opts.Sort)
	if opts.Sort == "filename" || opts.Sort == "size" {
		orderBy = fmt.Sprintf("files.%s", opts.Sort)
	}

	query := fmt.Sprintf(
		`SELECT files.id, files.user_id, files.filename, files.original_name, files.path, files.size_bytes, files.mime_type, files.sha256, files.media_type, files.width, files.height, files.duration_sec, files.taken_at, files.created_at, files.updated_at, files.is_deleted FROM files WHERE %s ORDER BY %s %s LIMIT ?`,
		whereClause, orderBy, opts.Order,
	)
	args = append(args, opts.Limit+1)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		f, err := s.scanFileFromRows(rows)
		if err != nil {
			return nil, "", 0, err
		}
		files = append(files, f)
	}
	if err := rows.Err(); err != nil {
		return nil, "", 0, err
	}

	var nextCursor string
	if len(files) > opts.Limit {
		files = files[:opts.Limit]
		nextCursor = files[len(files)-1].ID
	}

	return files, nextCursor, total, nil
}

func (s *FileStore) SoftDelete(id string) error {
	_, err := s.db.Exec(`UPDATE files SET is_deleted = 1, updated_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *FileStore) PermanentDelete(id string) error {
	_, err := s.db.Exec(`DELETE FROM files WHERE id = ?`, id)
	return err
}

type StatsResult struct {
	TotalFiles    int64
	TotalPhotos   int64
	TotalVideos   int64
	TotalSize     int64
	PhotosWithGPS int64
	DateOldest    *string
	DateNewest    *string
}

func (s *FileStore) Stats(userID string) (*StatsResult, error) {
	r := &StatsResult{}

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM files WHERE user_id = ? AND is_deleted = 0`, userID).Scan(&r.TotalFiles); err != nil {
		return nil, err
	}
	s.db.QueryRow(`SELECT COUNT(*) FROM files WHERE user_id = ? AND media_type = 'photo' AND is_deleted = 0`, userID).Scan(&r.TotalPhotos)
	s.db.QueryRow(`SELECT COUNT(*) FROM files WHERE user_id = ? AND media_type = 'video' AND is_deleted = 0`, userID).Scan(&r.TotalVideos)
	s.db.QueryRow(`SELECT COALESCE(SUM(size_bytes), 0) FROM files WHERE user_id = ? AND is_deleted = 0`, userID).Scan(&r.TotalSize)
	s.db.QueryRow(`SELECT COUNT(*) FROM files f INNER JOIN exif e ON f.id = e.file_id WHERE f.user_id = ? AND e.gps_latitude IS NOT NULL AND f.is_deleted = 0`, userID).Scan(&r.PhotosWithGPS)
	s.db.QueryRow(`SELECT MIN(taken_at) FROM files WHERE user_id = ? AND taken_at IS NOT NULL AND is_deleted = 0`, userID).Scan(&r.DateOldest)
	s.db.QueryRow(`SELECT MAX(taken_at) FROM files WHERE user_id = ? AND taken_at IS NOT NULL AND is_deleted = 0`, userID).Scan(&r.DateNewest)

	return r, nil
}

type DirEntry struct {
	Path      string      `json:"path"`
	Name      string      `json:"name"`
	FileCount int64       `json:"fileCount"`
	Children  []*DirEntry `json:"children,omitempty"`
}

func (s *FileStore) ListDirs(userID string) (*DirEntry, error) {
	rows, err := s.db.Query(`SELECT path FROM files WHERE user_id = ? AND is_deleted = 0`, userID)
	if err != nil {
		return nil, fmt.Errorf("query dirs: %w", err)
	}
	defer rows.Close()

	root := &DirEntry{Path: "", Name: "root", Children: []*DirEntry{}}
	pathCount := make(map[string]int64)

	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}
		pathCount[path]++
		parts := strings.Split(strings.Trim(path, "/"), "/")
		curr := root
		for i, part := range parts {
			if part == "" {
				break
			}
			found := false
			for _, c := range curr.Children {
				if c.Name == part {
					curr = c
					found = true
					break
				}
			}
			if !found {
				child := &DirEntry{Path: strings.Join(parts[:i+1], "/"), Name: part}
				curr.Children = append(curr.Children, child)
				curr = child
			}
		}
	}

	var collect func(*DirEntry)
	collect = func(e *DirEntry) {
		count := pathCount[e.Path]
		for _, c := range e.Children {
			collect(c)
			count += c.FileCount
		}
		e.FileCount = count
	}
	collect(root)

	return root, nil
}

type SearchResult struct {
	Files []*model.File
	Total int
}

func (s *FileStore) Search(userID, query string, limit int) (*SearchResult, error) {
	if limit <= 0 {
		limit = 50
	}

	fullQuery := `SELECT f.id, f.user_id, f.filename, f.original_name, f.path, f.size_bytes, f.mime_type, f.sha256, f.media_type, f.width, f.height, f.duration_sec, f.taken_at, f.created_at, f.updated_at, f.is_deleted FROM files f
		INNER JOIN files_fts ON f.rowid = files_fts.rowid
		WHERE files_fts MATCH ? AND f.user_id = ? AND f.is_deleted = 0
		ORDER BY rank LIMIT ?`

	rows, err := s.db.Query(fullQuery, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		f, err := s.scanFileFromRows(rows)
		if err != nil {
			continue
		}
		files = append(files, f)
	}

	return &SearchResult{Files: files, Total: len(files)}, rows.Err()
}

type TimelineGroup struct {
	Period       string `json:"period"`
	Label        string `json:"label"`
	Count        int    `json:"count"`
	ThumbnailURL string `json:"thumbnailUrl"`
	StartDate    string `json:"startDate"`
	EndDate      string `json:"endDate"`
}

func (s *FileStore) Timeline(userID, granularity string) ([]TimelineGroup, error) {
	if granularity == "" {
		granularity = "month"
	}

	var format, labelFormat string
	switch granularity {
	case "year":
		format = "%Y"
		labelFormat = "2006"
	case "day":
		format = "%Y-%m-%d"
		labelFormat = "January 2, 2006"
	default:
		format = "%Y-%m"
		labelFormat = "January 2006"
	}

	query := fmt.Sprintf(`SELECT strftime('%s', taken_at, 'localtime') as period, COUNT(*) as count, MIN(taken_at) as start_date FROM files WHERE user_id = ? AND taken_at IS NOT NULL AND is_deleted = 0 GROUP BY period ORDER BY period DESC`, format)

	rows, err := s.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("timeline: %w", err)
	}
	defer rows.Close()

	var groups []TimelineGroup
	for rows.Next() {
		var g TimelineGroup
		var startDate string
		if err := rows.Scan(&g.Period, &g.Count, &startDate); err != nil {
			continue
		}
		g.StartDate = startDate + "T00:00:00Z"
		t, _ := time.Parse(labelFormat, g.Period)
		if granularity == "month" {
			t, _ = time.Parse("2006-01", g.Period)
		} else if granularity == "year" {
			t, _ = time.Parse("2006", g.Period)
		}
		g.Label = t.Format(labelFormat)
		groups = append(groups, g)
	}

	return groups, rows.Err()
}

func (s *FileStore) scanFile(row interface{ Scan(dest ...interface{}) error }) (*model.File, error) {
	f := &model.File{}
	var width, height sql.NullInt64
	var durationSec sql.NullFloat64
	var takenAt sql.NullString
	var createdAt, updatedAt string
	var isDeleted int

	err := row.Scan(&f.ID, &f.UserID, &f.Filename, &f.OriginalName, &f.Path, &f.SizeBytes, &f.MimeType, &f.SHA256, &f.MediaType, &width, &height, &durationSec, &takenAt, &createdAt, &updatedAt, &isDeleted)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	if width.Valid {
		w := int(width.Int64)
		f.Width = &w
	}
	if height.Valid {
		h := int(height.Int64)
		f.Height = &h
	}
	if durationSec.Valid {
		f.DurationSec = &durationSec.Float64
	}
	if takenAt.Valid {
		f.TakenAt = &takenAt.String
	}
	f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	f.IsDeleted = isDeleted == 1

	return f, nil
}

type scannable interface {
	Scan(dest ...interface{}) error
}

func (s *FileStore) scanFileFromRows(row scannable) (*model.File, error) {
	f := &model.File{}
	var width, height sql.NullInt64
	var durationSec sql.NullFloat64
	var takenAt sql.NullString
	var createdAt, updatedAt string
	var isDeleted int

	err := row.Scan(&f.ID, &f.UserID, &f.Filename, &f.OriginalName, &f.Path, &f.SizeBytes, &f.MimeType, &f.SHA256, &f.MediaType, &width, &height, &durationSec, &takenAt, &createdAt, &updatedAt, &isDeleted)
	if err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	if width.Valid {
		w := int(width.Int64)
		f.Width = &w
	}
	if height.Valid {
		h := int(height.Int64)
		f.Height = &h
	}
	if durationSec.Valid {
		f.DurationSec = &durationSec.Float64
	}
	if takenAt.Valid {
		f.TakenAt = &takenAt.String
	}
	f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	f.IsDeleted = isDeleted == 1

	return f, nil
}

type FileRecord struct {
	ID           string  `json:"id"`
	OriginalName string  `json:"original_name"`
	SizeBytes    int64   `json:"size_bytes"`
}

func (s *FileStore) FindByNameAndSizeBatch(nameSizes []FileRecord) ([]*model.File, error) {
	if len(nameSizes) == 0 {
		return nil, nil
	}

	placeholders := make([]string, 0, len(nameSizes))
	args := make([]interface{}, 0, len(nameSizes)*2)
	for _, ns := range nameSizes {
		placeholders = append(placeholders, "(?, ?)")
		args = append(args, ns.OriginalName, ns.SizeBytes)
	}

	query := fmt.Sprintf(
		`SELECT id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, created_at, updated_at, is_deleted FROM files WHERE (original_name, size_bytes) IN (%s) AND is_deleted = 0`,
		strings.Join(placeholders, ", "),
	)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("find by name and size batch: %w", err)
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		f, err := s.scanFileFromRows(rows)
		if err != nil {
			continue
		}
		files = append(files, f)
	}

	return files, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
