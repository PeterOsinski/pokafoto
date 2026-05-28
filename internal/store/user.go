package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserStore struct {
	db *DB
}

func NewUserStore(db *DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(username, password string, role model.UserRole, displayName *string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
		DisplayName:  displayName,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	_, err = s.db.Exec(
		`INSERT INTO users (id, username, password_hash, role, display_name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.PasswordHash, user.Role, user.DisplayName, user.CreatedAt.Format(time.RFC3339), user.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

func (s *UserStore) FindByUsername(username string) (*model.User, error) {
	user := &model.User{}
	var displayName sql.NullString
	var createdAt, updatedAt string

	err := s.db.QueryRow(
		`SELECT id, username, password_hash, role, display_name, created_at, updated_at FROM users WHERE username = ?`,
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &displayName, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by username: %w", err)
	}

	if displayName.Valid {
		user.DisplayName = &displayName.String
	}
	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return user, nil
}

func (s *UserStore) FindByID(id string) (*model.User, error) {
	user := &model.User{}
	var displayName sql.NullString
	var createdAt, updatedAt string

	err := s.db.QueryRow(
		`SELECT id, username, password_hash, role, display_name, created_at, updated_at FROM users WHERE id = ?`,
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &displayName, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	if displayName.Valid {
		user.DisplayName = &displayName.String
	}
	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return user, nil
}

func (s *UserStore) List() ([]*model.User, error) {
	rows, err := s.db.Query(`SELECT id, username, role, display_name, created_at, updated_at FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		var displayName sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &displayName, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		if displayName.Valid {
			u.DisplayName = &displayName.String
		}
		u.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		u.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		users = append(users, u)
	}
	return users, rows.Err()
}

func (s *UserStore) UpdateRole(id string, role model.UserRole) error {
	_, err := s.db.Exec(`UPDATE users SET role = ?, updated_at = ? WHERE id = ?`, role, time.Now().UTC().Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("update user role: %w", err)
	}
	return nil
}

func (s *UserStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *UserStore) Count() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	return count, err
}
