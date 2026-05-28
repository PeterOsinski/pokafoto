package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type SessionStore struct {
	db *DB
}

func NewSessionStore(db *DB) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) Create(userID string, expiresAt time.Time) (*model.Session, error) {
	session := &model.Session{
		ID:           uuid.New().String(),
		UserID:       userID,
		RefreshToken: uuid.New().String(),
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now().UTC(),
	}

	_, err := s.db.Exec(
		`INSERT INTO sessions (id, user_id, refresh_token, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		session.ID, session.UserID, session.RefreshToken, session.ExpiresAt.Format(time.RFC3339), session.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("insert session: %w", err)
	}

	return session, nil
}

func (s *SessionStore) FindByRefreshToken(token string) (*model.Session, error) {
	sess := &model.Session{}
	var expiresAt, createdAt string

	err := s.db.QueryRow(
		`SELECT id, user_id, refresh_token, expires_at, created_at FROM sessions WHERE refresh_token = ?`,
		token,
	).Scan(&sess.ID, &sess.UserID, &sess.RefreshToken, &expiresAt, &createdAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find session by token: %w", err)
	}

	sess.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	sess.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return sess, nil
}

func (s *SessionStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *SessionStore) DeleteByRefreshToken(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE refresh_token = ?`, token)
	if err != nil {
		return fmt.Errorf("delete session by token: %w", err)
	}
	return nil
}

func (s *SessionStore) DeleteByUserID(userID string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	if err != nil {
		return fmt.Errorf("delete sessions by user id: %w", err)
	}
	return nil
}
