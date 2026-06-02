package model

type Document struct {
	FileID  string `json:"file_id" db:"file_id"`
	Content string `json:"content" db:"content"`
}
