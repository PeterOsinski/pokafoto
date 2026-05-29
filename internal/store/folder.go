package store

import (
	"fmt"
	"time"

	"github.com/drive/drive/internal/model"
	"github.com/google/uuid"
)

type FolderStore struct {
	db *DB
}

func NewFolderStore(db *DB) *FolderStore {
	return &FolderStore{db: db}
}

func (s *FolderStore) Create(userID, name string, parentID *string) (*model.Folder, error) {
	f := &model.Folder{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      name,
		ParentID:  parentID,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	_, err := s.db.Exec(
		`INSERT INTO folders (id, user_id, name, parent_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		f.ID, f.UserID, f.Name, f.ParentID, f.CreatedAt.Format(time.RFC3339), f.UpdatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return nil, fmt.Errorf("insert folder: %w", err)
	}

	return f, nil
}

func (s *FolderStore) FindByID(id string) (*model.Folder, error) {
	f := &model.Folder{}
	var parentID *string
	var createdAt, updatedAt string

	err := s.db.QueryRow(
		`SELECT id, user_id, name, parent_id, created_at, updated_at FROM folders WHERE id = ?`, id,
	).Scan(&f.ID, &f.UserID, &f.Name, &parentID, &createdAt, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("find folder: %w", err)
	}

	f.ParentID = parentID
	f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return f, nil
}

func (s *FolderStore) ListByUser(userID string) ([]*model.Folder, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, name, parent_id, created_at, updated_at FROM folders WHERE user_id = ? ORDER BY name`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list folders: %w", err)
	}
	defer rows.Close()

	var folders []*model.Folder
	for rows.Next() {
		f := &model.Folder{}
		var parentID *string
		var createdAt, updatedAt string
		if err := rows.Scan(&f.ID, &f.UserID, &f.Name, &parentID, &createdAt, &updatedAt); err != nil {
			continue
		}
		f.ParentID = parentID
		f.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		f.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		folders = append(folders, f)
	}

	return folders, rows.Err()
}

func (s *FolderStore) ListTree(userID string) (*model.FolderTreeNode, error) {
	allFolders, err := s.ListByUser(userID)
	if err != nil {
		return nil, err
	}

	fileCounts, err := s.folderFileCounts(userID)
	if err != nil {
		return nil, err
	}

	folderMap := make(map[string]*model.Folder)
	for _, f := range allFolders {
		folderMap[f.ID] = f
	}

	nodeMap := make(map[string]*model.FolderTreeNode)
	root := &model.FolderTreeNode{
		Children: []*model.FolderTreeNode{},
	}

	for _, f := range allFolders {
		node := &model.FolderTreeNode{
			Folder:    f,
			FileCount: fileCounts[f.ID],
			Children:  []*model.FolderTreeNode{},
		}
		nodeMap[f.ID] = node
	}

	for _, f := range allFolders {
		node := nodeMap[f.ID]
		if f.ParentID == nil || *f.ParentID == "" {
			root.Children = append(root.Children, node)
		} else if parent, ok := nodeMap[*f.ParentID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}

	var aggregateCounts func(node *model.FolderTreeNode)
	aggregateCounts = func(node *model.FolderTreeNode) {
		total := node.FileCount
		for _, child := range node.Children {
			aggregateCounts(child)
			total += child.FileCount
		}
		node.FileCount = total
	}
	for _, child := range root.Children {
		aggregateCounts(child)
	}

	return root, nil
}

func (s *FolderStore) folderFileCounts(userID string) (map[string]int64, error) {
	rows, err := s.db.Query(
		`SELECT folder_id, COUNT(*) FROM files WHERE user_id = ? AND folder_id IS NOT NULL AND is_deleted = 0 GROUP BY folder_id`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("folder file counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var folderID string
		var count int64
		if err := rows.Scan(&folderID, &count); err != nil {
			continue
		}
		counts[folderID] = count
	}
	return counts, rows.Err()
}

func (s *FolderStore) UpdateName(id, name string) error {
	_, err := s.db.Exec(
		`UPDATE folders SET name = ?, updated_at = ? WHERE id = ?`,
		name, time.Now().UTC().Format(time.RFC3339), id,
	)
	if err != nil {
		return fmt.Errorf("update folder name: %w", err)
	}
	return nil
}

func (s *FolderStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM folders WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete folder: %w", err)
	}
	return nil
}
