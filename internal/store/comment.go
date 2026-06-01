package store

import (
	"fmt"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type CommentStore struct {
	db *DB
}

func NewCommentStore(db *DB) *CommentStore {
	return &CommentStore{db: db}
}

func (s *CommentStore) Create(fileID, userID, content string) (*model.Comment, error) {
	c := &model.Comment{
		ID:        uuid.New().String(),
		FileID:    fileID,
		UserID:    userID,
		Content:   content,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err := s.db.Exec(
		`INSERT INTO comments (id, file_id, user_id, content, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		c.ID, c.FileID, c.UserID, c.Content, c.CreatedAt.Format(time.RFC3339), c.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("insert comment: %w", err)
	}

	return c, nil
}

func (s *CommentStore) FindByFileID(fileID string) ([]*model.Comment, error) {
	rows, err := s.db.Query(
		`SELECT id, file_id, user_id, content, created_at, updated_at FROM comments WHERE file_id = ? ORDER BY created_at ASC`, fileID,
	)
	if err != nil {
		return nil, fmt.Errorf("find comments: %w", err)
	}
	defer rows.Close()

	var comments []*model.Comment
	for rows.Next() {
		c := &model.Comment{}
		var createdAt, updatedAt string
		if err := rows.Scan(&c.ID, &c.FileID, &c.UserID, &c.Content, &createdAt, &updatedAt); err != nil {
			continue
		}
		c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (s *CommentStore) FindByID(id string) (*model.Comment, error) {
	c := &model.Comment{}
	var createdAt, updatedAt string
	err := s.db.QueryRow(
		`SELECT id, file_id, user_id, content, created_at, updated_at FROM comments WHERE id = ?`, id,
	).Scan(&c.ID, &c.FileID, &c.UserID, &c.Content, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("find comment: %w", err)
	}
	c.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	c.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return c, nil
}

func (s *CommentStore) Update(id, userID, content string) error {
	_, err := s.db.Exec(
		`UPDATE comments SET content = ?, updated_at = ? WHERE id = ? AND user_id = ?`,
		content, time.Now().UTC().Format(time.RFC3339), id, userID,
	)
	if err != nil {
		return fmt.Errorf("update comment: %w", err)
	}
	return nil
}

func (s *CommentStore) Delete(id, userID string) error {
	_, err := s.db.Exec(`DELETE FROM comments WHERE id = ? AND user_id = ?`, id, userID)
	if err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	return nil
}
