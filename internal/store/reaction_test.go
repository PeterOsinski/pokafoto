package store

import (
	"testing"
)

func TestReactionStore_Toggle_shouldAddReaction(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	file := createTestFile(t, fs, u.ID, "test.jpg")
	commentStore := NewCommentStore(db)
	c, _ := commentStore.Create(file.ID, u.ID, "Nice")

	reactionStore := NewReactionStore(db)
	added, err := reactionStore.Toggle(c.ID, u.ID, "👍")
	if err != nil {
		t.Fatalf("Toggle() error = %v", err)
	}
	if !added {
		t.Error("expected reaction to be added")
	}
}

func TestReactionStore_Toggle_shouldRemoveReaction(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	file := createTestFile(t, fs, u.ID, "test.jpg")
	commentStore := NewCommentStore(db)
	c, _ := commentStore.Create(file.ID, u.ID, "Nice")

	reactionStore := NewReactionStore(db)
	reactionStore.Toggle(c.ID, u.ID, "👍")
	added, err := reactionStore.Toggle(c.ID, u.ID, "👍")
	if err != nil {
		t.Fatalf("Toggle() error = %v", err)
	}
	if added {
		t.Error("expected reaction to be removed on toggle")
	}
}

func TestReactionStore_FindByCommentID_shouldReturnReactions(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	file := createTestFile(t, fs, u.ID, "test.jpg")
	commentStore := NewCommentStore(db)
	c, _ := commentStore.Create(file.ID, u.ID, "Nice")

	reactionStore := NewReactionStore(db)
	reactionStore.Toggle(c.ID, u.ID, "👍")
	reactionStore.Toggle(c.ID, u.ID, "❤️")

	reactions, err := reactionStore.FindByCommentID(c.ID, u.ID)
	if err != nil {
		t.Fatalf("FindByCommentID() error = %v", err)
	}
	if len(reactions) != 2 {
		t.Errorf("expected 2 reaction groups, got %d", len(reactions))
	}
}

func TestReactionStore_Remove_shouldDeleteReaction(t *testing.T) {
	t.Parallel()
	db := OpenTestDB(t)

	us := NewUserStore(db)
	u := createTestUser(t, us)
	fs := NewFileStore(db)
	file := createTestFile(t, fs, u.ID, "test.jpg")
	commentStore := NewCommentStore(db)
	c, _ := commentStore.Create(file.ID, u.ID, "Nice")

	reactionStore := NewReactionStore(db)
	reactionStore.Toggle(c.ID, u.ID, "🔥")

	if err := reactionStore.Remove(c.ID, u.ID, "🔥"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	reactions, _ := reactionStore.FindByCommentID(c.ID, u.ID)
	if len(reactions) != 0 {
		t.Errorf("expected 0 reactions after removal, got %d", len(reactions))
	}
}
