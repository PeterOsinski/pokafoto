package model

import "time"

type Reaction struct {
	ID        string    `json:"id" db:"id"`
	CommentID string    `json:"comment_id" db:"comment_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Emoji     string    `json:"emoji" db:"emoji"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type ReactionGroup struct {
	Emoji  string `json:"emoji"`
	Count  int64  `json:"count"`
	HasMine bool   `json:"has_mine"`
}
