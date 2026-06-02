package store

import (
	"fmt"

	"github.com/drive/drive/internal/model"
)

type DocumentStore struct {
	db *DB
}

func NewDocumentStore(db *DB) *DocumentStore {
	return &DocumentStore{db: db}
}

func (s *DocumentStore) Create(fileID, content string) error {
	_, err := s.db.Exec(
		`INSERT INTO documents (file_id, content) VALUES (?, ?)`,
		fileID, content,
	)
	if err != nil {
		return fmt.Errorf("insert document: %w", err)
	}
	return nil
}

func (s *DocumentStore) FindByFileID(fileID string) (*model.Document, error) {
	d := &model.Document{}
	err := s.db.QueryRow(
		`SELECT file_id, content FROM documents WHERE file_id = ?`, fileID,
	).Scan(&d.FileID, &d.Content)
	if err != nil {
		return nil, fmt.Errorf("find document: %w", err)
	}
	return d, nil
}

func (s *DocumentStore) UpdateContent(fileID, content string) error {
	_, err := s.db.Exec(
		`UPDATE documents SET content = ? WHERE file_id = ?`,
		content, fileID,
	)
	if err != nil {
		return fmt.Errorf("update document: %w", err)
	}
	return nil
}

func (s *DocumentStore) Delete(fileID string) error {
	_, err := s.db.Exec(`DELETE FROM documents WHERE file_id = ?`, fileID)
	return err
}
