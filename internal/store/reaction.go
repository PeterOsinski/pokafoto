package store

import (
	"fmt"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type ReactionStore struct {
	db *DB
}

func NewReactionStore(db *DB) *ReactionStore {
	return &ReactionStore{db: db}
}

func (s *ReactionStore) Toggle(commentID, userID, emoji string) (added bool, err error) {
	var existingID string
	err = s.db.QueryRow(
		`SELECT id FROM reactions WHERE comment_id = ? AND user_id = ? AND emoji = ?`,
		commentID, userID, emoji,
	).Scan(&existingID)
	if err == nil {
		_, err = s.db.Exec(`DELETE FROM reactions WHERE id = ?`, existingID)
		if err != nil {
			return false, fmt.Errorf("delete reaction: %w", err)
		}
		return false, nil
	}

	r := &model.Reaction{
		ID:        uuid.New().String(),
		CommentID: commentID,
		UserID:    userID,
		Emoji:     emoji,
		CreatedAt: time.Now().UTC(),
	}
	_, err = s.db.Exec(
		`INSERT INTO reactions (id, comment_id, user_id, emoji, created_at) VALUES (?, ?, ?, ?, ?)`,
		r.ID, r.CommentID, r.UserID, r.Emoji, r.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return false, fmt.Errorf("insert reaction: %w", err)
	}

	return true, nil
}

func (s *ReactionStore) FindByCommentID(commentID, viewerUserID string) ([]model.ReactionGroup, error) {
	rows, err := s.db.Query(
		`SELECT emoji, COUNT(*) as cnt,
			(SELECT COUNT(*) > 0 FROM reactions r2 WHERE r2.comment_id = ? AND r2.user_id = ? AND r2.emoji = r.emoji) as has_mine
		 FROM reactions r WHERE r.comment_id = ? GROUP BY emoji ORDER BY cnt DESC`,
		commentID, viewerUserID, commentID,
	)
	if err != nil {
		return nil, fmt.Errorf("find reactions: %w", err)
	}
	defer rows.Close()

	var groups []model.ReactionGroup
	for rows.Next() {
		g := model.ReactionGroup{}
		if err := rows.Scan(&g.Emoji, &g.Count, &g.HasMine); err != nil {
			continue
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (s *ReactionStore) Remove(commentID, userID, emoji string) error {
	_, err := s.db.Exec(
		`DELETE FROM reactions WHERE comment_id = ? AND user_id = ? AND emoji = ?`,
		commentID, userID, emoji,
	)
	if err != nil {
		return fmt.Errorf("delete reaction: %w", err)
	}
	return nil
}
