package model

import "time"

type Folder struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	Name      string    `json:"name" db:"name"`
	ParentID  *string   `json:"parent_id,omitempty" db:"parent_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type FolderTreeNode struct {
	Folder    *Folder             `json:"folder"`
	FileCount int64               `json:"fileCount"`
	HasShares bool                `json:"hasShares"`
	Children  []*FolderTreeNode   `json:"children,omitempty"`
}
