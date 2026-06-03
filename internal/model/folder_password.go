package model

import "time"

type FolderPassword struct {
	ID           string    `json:"id" db:"id"`
	FolderID     string    `json:"folder_id" db:"folder_id"`
	PasswordHash string    `json:"-" db:"password_hash"`
	PasswordHint string    `json:"password_hint" db:"password_hint"`
	ExpiresAt    time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
