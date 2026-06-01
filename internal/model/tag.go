package model

import "time"

type Tag struct {
	ID   string `json:"id" db:"id"`
	Name string `json:"name" db:"name"`
}

type TagWithCount struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type FileTag struct {
	FileID         string    `json:"file_id" db:"file_id"`
	TagID          string    `json:"tag_id" db:"tag_id"`
	AddedByUserID  string    `json:"added_by_user_id" db:"added_by_user_id"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}
