package model

import "time"

type Album struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type AlbumItem struct {
	ID             string    `json:"id" db:"id"`
	AlbumID        string    `json:"album_id" db:"album_id"`
	FileID         string    `json:"file_id" db:"file_id"`
	AddedByUserID  string    `json:"added_by_user_id" db:"added_by_user_id"`
	SortOrder      int       `json:"sort_order" db:"sort_order"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type AlbumShare struct {
	ID                string    `json:"id" db:"id"`
	AlbumID           string    `json:"album_id" db:"album_id"`
	SharedWithUserID  string    `json:"shared_with_user_id" db:"shared_with_user_id"`
	Permission        string    `json:"permission" db:"permission"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

type AlbumWithDetails struct {
	Album       *Album        `json:"album"`
	OwnerName   string        `json:"owner_name"`
	ItemCount   int64         `json:"item_count"`
	CoverFileID *string       `json:"cover_file_id,omitempty"`
	SharedUsers []SharedUser  `json:"shared_users,omitempty"`
}

type SharedUser struct {
	ShareID    string `json:"share_id"`
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
	Permission string `json:"permission"`
}
