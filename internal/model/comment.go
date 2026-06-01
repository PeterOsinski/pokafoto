package model

import "time"

type Comment struct {
	ID        string    `json:"id" db:"id"`
	FileID    string    `json:"file_id" db:"file_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CommentWithUser struct {
	Comment  *Comment    `json:"comment"`
	Username string      `json:"username"`
	Reactions []ReactionGroup `json:"reactions,omitempty"`
}
