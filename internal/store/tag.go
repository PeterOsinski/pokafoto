package store

import (
	"fmt"
	"strings"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type TagStore struct {
	db *DB
}

func NewTagStore(db *DB) *TagStore {
	return &TagStore{db: db}
}

func (s *TagStore) FindOrCreate(name string) (*model.Tag, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return nil, fmt.Errorf("tag name cannot be empty")
	}

	tag := &model.Tag{}
	err := s.db.QueryRow(`SELECT id, name FROM tags WHERE name = ?`, name).Scan(&tag.ID, &tag.Name)
	if err == nil {
		return tag, nil
	}

	tag = &model.Tag{
		ID:   uuid.New().String(),
		Name: name,
	}
	_, err = s.db.Exec(`INSERT OR IGNORE INTO tags (id, name) VALUES (?, ?)`, tag.ID, tag.Name)
	if err != nil {
		return nil, fmt.Errorf("insert tag: %w", err)
	}

	if tag.ID == "" {
		err = s.db.QueryRow(`SELECT id, name FROM tags WHERE name = ?`, name).Scan(&tag.ID, &tag.Name)
		if err != nil {
			return nil, fmt.Errorf("find tag after insert: %w", err)
		}
	}

	return tag, nil
}

func (s *TagStore) Search(prefix string) ([]*model.Tag, error) {
	rows, err := s.db.Query(
		`SELECT id, name FROM tags WHERE name LIKE ? ORDER BY name LIMIT 20`,
		strings.ToLower(prefix)+"%",
	)
	if err != nil {
		return nil, fmt.Errorf("search tags: %w", err)
	}
	defer rows.Close()

	var tags []*model.Tag
	for rows.Next() {
		t := &model.Tag{}
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			continue
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *TagStore) AddToFile(fileID, tagID, userID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO file_tags (file_id, tag_id, added_by_user_id, created_at) VALUES (?, ?, ?, ?)`,
		fileID, tagID, userID, now,
	)
	if err != nil {
		return fmt.Errorf("add tag to file: %w", err)
	}
	return nil
}

func (s *TagStore) RemoveFromFile(fileID, tagID string) error {
	_, err := s.db.Exec(`DELETE FROM file_tags WHERE file_id = ? AND tag_id = ?`, fileID, tagID)
	if err != nil {
		return fmt.Errorf("remove tag from file: %w", err)
	}
	return nil
}

func (s *TagStore) FindByFileID(fileID string) ([]*model.Tag, error) {
	rows, err := s.db.Query(
		`SELECT t.id, t.name FROM tags t JOIN file_tags ft ON t.id = ft.tag_id WHERE ft.file_id = ? ORDER BY t.name`,
		fileID,
	)
	if err != nil {
		return nil, fmt.Errorf("find file tags: %w", err)
	}
	defer rows.Close()

	var tags []*model.Tag
	for rows.Next() {
		t := &model.Tag{}
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			continue
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *TagStore) ListWithCount(userID string) ([]model.TagWithCount, error) {
	rows, err := s.db.Query(
		`SELECT t.id, t.name, COUNT(ft.file_id) as cnt
		 FROM tags t
		 JOIN file_tags ft ON t.id = ft.tag_id
		 JOIN files f ON ft.file_id = f.id
		 WHERE f.user_id = ? AND f.is_deleted = 0
		 GROUP BY t.id, t.name
		 ORDER BY COUNT(ft.file_id) DESC, t.name`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list tags with count: %w", err)
	}
	defer rows.Close()

	var tags []model.TagWithCount
	for rows.Next() {
		t := model.TagWithCount{}
		if err := rows.Scan(&t.ID, &t.Name, &t.Count); err != nil {
			continue
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
