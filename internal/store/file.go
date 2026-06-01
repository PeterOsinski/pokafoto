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
		`INSERT INTO files (id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, folder_id, created_at, updated_at, is_deleted, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		file.ID, file.UserID, file.Filename, file.OriginalName, file.Path, file.SizeBytes, file.MimeType, file.SHA256, file.MediaType, file.Width, file.Height, file.DurationSec, file.TakenAt, file.FolderID, file.CreatedAt.Format(time.RFC3339), file.UpdatedAt.Format(time.RFC3339), boolToInt(file.IsDeleted), nil,
	)
	if err != nil {
		return fmt.Errorf("insert file: %w", err)
	}

	return nil
}

func (s *FileStore) FindByID(id string) (*model.File, error) {
	return s.scanFile(s.db.QueryRow(
		`SELECT id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, folder_id, created_at, updated_at, deleted_at, is_deleted FROM files WHERE id = ?`,
		id,
	))
}

func (s *FileStore) FindBySHA256(userID, sha256 string) (*model.File, error) {
	return s.scanFile(s.db.QueryRow(
		`SELECT id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, folder_id, created_at, updated_at, deleted_at, is_deleted FROM files WHERE sha256 = ? AND user_id = ? AND is_deleted = 0`,
		sha256, userID,
	))
}

func (s *FileStore) FindByNameAndSize(userID, name string, size int64) (*model.File, error) {
	return s.scanFile(s.db.QueryRow(
		`SELECT id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, folder_id, created_at, updated_at, deleted_at, is_deleted FROM files WHERE original_name = ? AND size_bytes = ? AND user_id = ? AND is_deleted = 0 LIMIT 1`,
		name, size, userID,
	))
}

type FileListOptions struct {
	UserID     string
	Path       string
	FolderID   *string
	AllFolders bool
	Cursor     string
	Limit      int
	Sort       string
	Order      string
	MediaType  string
	DateFrom   string
	DateTo     string
	Camera     string
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

	if opts.FolderID != nil {
		if *opts.FolderID == "" {
			conditions = append(conditions, "folder_id IS NULL")
		} else {
			conditions = append(conditions, "folder_id = ?")
			args = append(args, *opts.FolderID)
		}
	} else if !opts.AllFolders {
		conditions = append(conditions, "folder_id IS NULL")
	}

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
		`SELECT files.id, files.user_id, files.filename, files.original_name, files.path, files.size_bytes, files.mime_type, files.sha256, files.media_type, files.width, files.height, files.duration_sec, files.taken_at, files.folder_id, files.created_at, files.updated_at, files.deleted_at, files.is_deleted FROM files WHERE %s ORDER BY %s %s LIMIT ?`,
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
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(`UPDATE files SET is_deleted = 1, deleted_at = ?, updated_at = ? WHERE id = ?`, now, now, id)
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

func (s *FileStore) ListDirs(userID string, allFolders bool) (*DirEntry, error) {
	query := `SELECT path FROM files WHERE user_id = ? AND is_deleted = 0`
	args := []interface{}{userID}
	if !allFolders {
		query += ` AND folder_id IS NULL`
	}
	rows, err := s.db.Query(query, args...)
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

type SearchOptions struct {
	UserID        string
	Query         string
	SizeMin       *int64
	SizeMax       *int64
	CreatedAfter  *string
	CreatedBefore *string
	TakenAfter    *string
	TakenBefore   *string
	Tags          []string
	Limit         int
	Cursor        string
}

func (s *FileStore) SearchEnhanced(opts SearchOptions) (*SearchResult, map[string]string, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}

	args := []interface{}{opts.UserID}
	conditions := []string{"f.user_id = ?", "f.is_deleted = 0"}

	if opts.Query != "" {
		ftsCondition := "f.rowid IN (SELECT rowid FROM files_fts WHERE files_fts MATCH ?)"
		conditions = append(conditions, ftsCondition)
		args = append(args, opts.Query)
	}

	if opts.SizeMin != nil {
		conditions = append(conditions, "f.size_bytes >= ?")
		args = append(args, *opts.SizeMin)
	}
	if opts.SizeMax != nil {
		conditions = append(conditions, "f.size_bytes <= ?")
		args = append(args, *opts.SizeMax)
	}
	if opts.CreatedAfter != nil {
		conditions = append(conditions, "f.created_at >= ?")
		args = append(args, *opts.CreatedAfter)
	}
	if opts.CreatedBefore != nil {
		conditions = append(conditions, "f.created_at <= ?")
		args = append(args, *opts.CreatedBefore)
	}
	if opts.TakenAfter != nil {
		conditions = append(conditions, "f.taken_at >= ?")
		args = append(args, *opts.TakenAfter)
	}
	if opts.TakenBefore != nil {
		conditions = append(conditions, "f.taken_at <= ?")
		args = append(args, *opts.TakenBefore)
	}
	if len(opts.Tags) > 0 {
		placeholders := make([]string, len(opts.Tags))
		for i, tag := range opts.Tags {
			placeholders[i] = "?"
			args = append(args, strings.TrimSpace(strings.ToLower(tag)))
		}
		conditions = append(conditions, fmt.Sprintf(
			"f.id IN (SELECT ft.file_id FROM file_tags ft JOIN tags t ON t.id = ft.tag_id WHERE t.name IN (%s))",
			strings.Join(placeholders, ","),
		))
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`SELECT f.id, f.user_id, f.filename, f.original_name, f.path, f.size_bytes, f.mime_type, f.sha256, f.media_type, f.width, f.height, f.duration_sec, f.taken_at, f.folder_id, f.created_at, f.updated_at, f.deleted_at, f.is_deleted FROM files f WHERE %s ORDER BY f.created_at DESC LIMIT ?`, whereClause)
	allArgs := append([]interface{}{}, args...)
	allArgs = append(allArgs, opts.Limit)

	rows, err := s.db.Query(query, allArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("search enhanced: %w", err)
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

	if rows.Err() != nil {
		return nil, nil, rows.Err()
	}

	folderPaths := make(map[string]string)
	for _, f := range files {
		if f.FolderID != nil && *f.FolderID != "" {
			folderPaths[f.ID] = s.folderPath(*f.FolderID)
		}
	}

	return &SearchResult{Files: files, Total: len(files)}, folderPaths, nil
}

func (s *FileStore) folderPath(folderID string) string {
	path := ""
	current := &folderID
	for i := 0; i < 10 && current != nil; i++ {
		var name string
		var parentID *string
		err := s.db.QueryRow(`SELECT name, parent_id FROM folders WHERE id = ?`, *current).Scan(&name, &parentID)
		if err != nil {
			break
		}
		if path == "" {
			path = name
		} else {
			path = name + "/" + path
		}
		current = parentID
	}
	return path
}

func (s *FileStore) Search(userID, query string, limit int) (*SearchResult, error) {
	if limit <= 0 {
		limit = 50
	}

	fullQuery := `SELECT f.id, f.user_id, f.filename, f.original_name, f.path, f.size_bytes, f.mime_type, f.sha256, f.media_type, f.width, f.height, f.duration_sec, f.taken_at, f.folder_id, f.created_at, f.updated_at, f.deleted_at, f.is_deleted FROM files f
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
	var folderID sql.NullString
	var createdAt, updatedAt string
	var deletedAt sql.NullString
	var isDeleted int

	err := row.Scan(&f.ID, &f.UserID, &f.Filename, &f.OriginalName, &f.Path, &f.SizeBytes, &f.MimeType, &f.SHA256, &f.MediaType, &width, &height, &durationSec, &takenAt, &folderID, &createdAt, &updatedAt, &deletedAt, &isDeleted)
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
	if folderID.Valid {
		f.FolderID = &folderID.String
	}
	f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	f.IsDeleted = isDeleted == 1
	if deletedAt.Valid {
		t, _ := time.Parse(time.RFC3339, deletedAt.String)
		f.DeletedAt = &t
	}

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
	var folderID sql.NullString
	var createdAt, updatedAt string
	var deletedAt sql.NullString
	var isDeleted int

	err := row.Scan(&f.ID, &f.UserID, &f.Filename, &f.OriginalName, &f.Path, &f.SizeBytes, &f.MimeType, &f.SHA256, &f.MediaType, &width, &height, &durationSec, &takenAt, &folderID, &createdAt, &updatedAt, &deletedAt, &isDeleted)
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
	if folderID.Valid {
		f.FolderID = &folderID.String
	}
	f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	f.IsDeleted = isDeleted == 1
	if deletedAt.Valid {
		t, _ := time.Parse(time.RFC3339, deletedAt.String)
		f.DeletedAt = &t
	}

	return f, nil
}

type FileRecord struct {
	ID           string  `json:"id"`
	OriginalName string  `json:"original_name"`
	SizeBytes    int64   `json:"size_bytes"`
}

func (s *FileStore) FindByNameAndSizeBatch(userID string, nameSizes []FileRecord) ([]*model.File, error) {
	if len(nameSizes) == 0 {
		return nil, nil
	}

	placeholders := make([]string, 0, len(nameSizes))
	args := make([]interface{}, 0, len(nameSizes)*2+1)
	for _, ns := range nameSizes {
		placeholders = append(placeholders, "(?, ?)")
		args = append(args, ns.OriginalName, ns.SizeBytes)
	}
	args = append(args, userID)

	query := fmt.Sprintf(
		`SELECT id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, folder_id, created_at, updated_at, deleted_at, is_deleted FROM files WHERE (original_name, size_bytes) IN (%s) AND user_id = ? AND is_deleted = 0`,
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

func (s *FileStore) BatchSoftDelete(userID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids)+3)
	now := time.Now().UTC().Format(time.RFC3339)
	args = append(args, now, now, userID)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(
		`UPDATE files SET is_deleted = 1, deleted_at = ?, updated_at = ? WHERE user_id = ? AND is_deleted = 0 AND id IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("batch soft delete: %w", err)
	}
	return nil
}

func (s *FileStore) BatchMove(userID string, ids []string, folderID *string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids)+2)
	args = append(args, folderID, time.Now().UTC().Format(time.RFC3339), userID)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(
		`UPDATE files SET folder_id = ?, updated_at = ? WHERE user_id = ? AND is_deleted = 0 AND id IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("batch move: %w", err)
	}
	return nil
}

type AdminFileBreakdown struct {
	MediaTypes []MediaTypeBreakdown `json:"media_types"`
	Extensions []ExtensionBreakdown `json:"extensions"`
	TotalSize  int64                `json:"total_size"`
}

type MediaTypeBreakdown struct {
	MediaType string `json:"media_type"`
	Count     int64  `json:"count"`
	SizeBytes int64  `json:"size_bytes"`
}

type ExtensionBreakdown struct {
	Extension string `json:"extension"`
	Count     int64  `json:"count"`
	SizeBytes int64  `json:"size_bytes"`
}

func (s *FileStore) AdminFileBreakdown() (*AdminFileBreakdown, error) {
	b := &AdminFileBreakdown{}

	rows, err := s.db.Query(`SELECT media_type, COUNT(*), COALESCE(SUM(size_bytes), 0) FROM files WHERE is_deleted = 0 GROUP BY media_type ORDER BY media_type`)
	if err != nil {
		return nil, fmt.Errorf("media type breakdown: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var mt MediaTypeBreakdown
		if err := rows.Scan(&mt.MediaType, &mt.Count, &mt.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan media type: %w", err)
		}
		b.MediaTypes = append(b.MediaTypes, mt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows media type: %w", err)
	}

	xrows, err := s.db.Query(`SELECT LOWER(SUBSTR(mime_type, INSTR(mime_type, '/') + 1)) as ext, COUNT(*) as cnt, COALESCE(SUM(size_bytes), 0) FROM files WHERE is_deleted = 0 GROUP BY ext ORDER BY cnt DESC`)
	if err != nil {
		return nil, fmt.Errorf("extension breakdown: %w", err)
	}
	defer xrows.Close()

	for xrows.Next() {
		var eb ExtensionBreakdown
		if err := xrows.Scan(&eb.Extension, &eb.Count, &eb.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan extension: %w", err)
		}
		b.Extensions = append(b.Extensions, eb)
	}
	if err := xrows.Err(); err != nil {
		return nil, fmt.Errorf("rows extension: %w", err)
	}

	if err := s.db.QueryRow(`SELECT COALESCE(SUM(size_bytes), 0) FROM files WHERE is_deleted = 0`).Scan(&b.TotalSize); err != nil {
		return nil, fmt.Errorf("total size: %w", err)
	}

	return b, nil
}

func (s *FileStore) AdminFileBreakdownByUser(userID string) (*AdminFileBreakdown, error) {
	b := &AdminFileBreakdown{}

	rows, err := s.db.Query(`SELECT media_type, COUNT(*), COALESCE(SUM(size_bytes), 0) FROM files WHERE is_deleted = 0 AND user_id = ? GROUP BY media_type ORDER BY media_type`, userID)
	if err != nil {
		return nil, fmt.Errorf("media type breakdown by user: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var mt MediaTypeBreakdown
		if err := rows.Scan(&mt.MediaType, &mt.Count, &mt.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan media type by user: %w", err)
		}
		b.MediaTypes = append(b.MediaTypes, mt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows media type by user: %w", err)
	}

	xrows, err := s.db.Query(`SELECT LOWER(SUBSTR(mime_type, INSTR(mime_type, '/') + 1)) as ext, COUNT(*) as cnt, COALESCE(SUM(size_bytes), 0) FROM files WHERE is_deleted = 0 AND user_id = ? GROUP BY ext ORDER BY cnt DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("extension breakdown by user: %w", err)
	}
	defer xrows.Close()

	for xrows.Next() {
		var eb ExtensionBreakdown
		if err := xrows.Scan(&eb.Extension, &eb.Count, &eb.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan extension by user: %w", err)
		}
		b.Extensions = append(b.Extensions, eb)
	}
	if err := xrows.Err(); err != nil {
		return nil, fmt.Errorf("rows extension by user: %w", err)
	}

	if err := s.db.QueryRow(`SELECT COALESCE(SUM(size_bytes), 0) FROM files WHERE is_deleted = 0 AND user_id = ?`, userID).Scan(&b.TotalSize); err != nil {
		return nil, fmt.Errorf("total size by user: %w", err)
	}

	return b, nil
}

func (s *FileStore) BatchCopy(userID string, ids []string, folderID *string) ([]*model.File, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, userID)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	selectQuery := fmt.Sprintf(
		`SELECT id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, folder_id, created_at, updated_at, deleted_at, is_deleted FROM files WHERE user_id = ? AND is_deleted = 0 AND id IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	rows, err := s.db.Query(selectQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("batch copy select: %w", err)
	}
	defer rows.Close()

	var copies []*model.File
	now := time.Now().UTC().Format(time.RFC3339)

	for rows.Next() {
		src, err := s.scanFileFromRows(rows)
		if err != nil {
			continue
		}

		copy := &model.File{
			ID:           uuid.New().String(),
			UserID:       src.UserID,
			Filename:     src.Filename,
			OriginalName: src.OriginalName,
			Path:         src.Path,
			SizeBytes:    src.SizeBytes,
			MimeType:     src.MimeType,
			SHA256:       src.SHA256,
			MediaType:    src.MediaType,
			Width:        src.Width,
			Height:       src.Height,
			DurationSec:  src.DurationSec,
			TakenAt:      src.TakenAt,
			FolderID:     folderID,
			CreatedAt:    time.Now().UTC(),
			UpdatedAt:    time.Now().UTC(),
		}

		_, err = s.db.Exec(
			`INSERT INTO files (id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, folder_id, created_at, updated_at, is_deleted, deleted_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, NULL)`,
			copy.ID, copy.UserID, copy.Filename, copy.OriginalName, copy.Path, copy.SizeBytes, copy.MimeType, copy.SHA256, copy.MediaType, copy.Width, copy.Height, copy.DurationSec, copy.TakenAt, copy.FolderID, now, now,
		)
		if err != nil {
			continue
		}

		copies = append(copies, copy)
	}

	return copies, rows.Err()
}

func (s *FileStore) FindPhotosMissingThumbnails() ([]*model.File, error) {
	rows, err := s.db.Query(
		`SELECT f.id, f.user_id, f.filename, f.original_name, f.path, f.size_bytes, f.mime_type, f.sha256, f.media_type, f.width, f.height, f.duration_sec, f.taken_at, f.folder_id, f.created_at, f.updated_at, f.deleted_at, f.is_deleted FROM files f WHERE f.media_type IN ('photo', 'video') AND f.is_deleted = 0 AND (NOT EXISTS (SELECT 1 FROM thumbnails t WHERE t.file_id = f.id AND t.size = 'preview') OR NOT EXISTS (SELECT 1 FROM thumbnails t WHERE t.file_id = f.id)) ORDER BY f.created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("find photos missing thumbnails: %w", err)
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		f, err := s.scanFile(rows)
		if err != nil {
			continue
		}
		files = append(files, f)
	}

	return files, rows.Err()
}

func (s *FileStore) CountPhotosMissingThumbnailPreview() (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM files f WHERE f.media_type IN ('photo', 'video') AND f.is_deleted = 0 AND NOT EXISTS (SELECT 1 FROM thumbnails t WHERE t.file_id = f.id AND t.size = 'preview')`,
	).Scan(&count)
	return count, err
}

func (s *FileStore) Restore(id string) error {
	_, err := s.db.Exec(`UPDATE files SET is_deleted = 0, deleted_at = NULL, updated_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339), id)
	return err
}

func (s *FileStore) BatchRestore(userID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids)+2)
	args = append(args, time.Now().UTC().Format(time.RFC3339), userID)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(
		`UPDATE files SET is_deleted = 0, deleted_at = NULL, updated_at = ? WHERE user_id = ? AND is_deleted = 1 AND id IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("batch restore: %w", err)
	}
	return nil
}

func (s *FileStore) ListTrash(opts FileListOptions) ([]*model.File, string, int, error) {
	if opts.Limit <= 0 || opts.Limit > 500 {
		opts.Limit = 100
	}
	if opts.Order == "" {
		opts.Order = "desc"
	}

	var conditions []string
	var args []interface{}

	conditions = append(conditions, "user_id = ?")
	args = append(args, opts.UserID)

	conditions = append(conditions, "is_deleted = 1")

	whereClause := strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM files WHERE %s`, whereClause)
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, "", 0, fmt.Errorf("count trash: %w", err)
	}

	if opts.Cursor != "" {
		cmp := ">"
		if opts.Order == "desc" {
			cmp = "<"
		}
		conditions = append(conditions, fmt.Sprintf("id %s ?", cmp))
		args = append(args, opts.Cursor)
	}

	query := fmt.Sprintf(
		`SELECT id, user_id, filename, original_name, path, size_bytes, mime_type, sha256, media_type, width, height, duration_sec, taken_at, folder_id, created_at, updated_at, deleted_at, is_deleted FROM files WHERE %s ORDER BY deleted_at %s LIMIT ?`,
		whereClause, opts.Order,
	)
	args = append(args, opts.Limit+1)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list trash: %w", err)
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

type TrashStatsResult struct {
	Count     int64 `json:"count"`
	SizeBytes int64 `json:"size_bytes"`
}

func (s *FileStore) TrashStats(userID string) (*TrashStatsResult, error) {
	r := &TrashStatsResult{}
	err := s.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(size_bytes), 0) FROM files WHERE user_id = ? AND is_deleted = 1`,
		userID,
	).Scan(&r.Count, &r.SizeBytes)
	if err != nil {
		return nil, fmt.Errorf("trash stats: %w", err)
	}
	return r, nil
}

type ExpiredFile struct {
	ID       string
	UserID   string
	Filename string
}

func (s *FileStore) GetExpiredFiles(cutoff string, limit int) ([]ExpiredFile, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, filename FROM files WHERE is_deleted = 1 AND deleted_at < ? LIMIT ?`,
		cutoff, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("get expired files: %w", err)
	}
	defer rows.Close()

	var files []ExpiredFile
	for rows.Next() {
		var f ExpiredFile
		if err := rows.Scan(&f.ID, &f.UserID, &f.Filename); err != nil {
			continue
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (s *FileStore) PermanentDeleteByIDs(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`DELETE FROM files WHERE id IN (%s)`, strings.Join(placeholders, ", "))
	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("permanent delete by ids: %w", err)
	}
	return nil
}

func (s *FileStore) BatchPermanentDelete(userID string, ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, userID)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(
		`DELETE FROM files WHERE user_id = ? AND is_deleted = 1 AND id IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("batch permanent delete: %w", err)
	}
	return nil
}
