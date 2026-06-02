package model

import "time"

type SharePermission string

const (
	ShareRead      SharePermission = "read"
	ShareReadUpload SharePermission = "read_upload"
	ShareReadWrite  SharePermission = "read_write"
)

type FolderShare struct {
	ID               string          `json:"id" db:"id"`
	FolderID         string          `json:"folder_id" db:"folder_id"`
	Token            string          `json:"token" db:"token"`
	Permissions      SharePermission `json:"permissions" db:"permissions"`
	IncludeSubdirs   bool            `json:"include_subdirs" db:"include_subdirs"`
	UploadLimitBytes *int64          `json:"upload_limit_bytes,omitempty" db:"upload_limit_bytes"`
	ExpiresAt        *time.Time      `json:"expires_at,omitempty" db:"expires_at"`
	HasPassword      bool            `json:"has_password" db:"has_password"`
	PasswordHash     *string         `json:"-" db:"password_hash"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}
