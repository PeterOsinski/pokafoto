package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRealFS_ReadDir_readsDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("bb"), 0644)

	fs := NewRealFS()
	entries, err := fs.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
}

func TestRealFS_ReadDir_nonexistent(t *testing.T) {
	t.Parallel()
	fs := NewRealFS()
	_, err := fs.ReadDir("/nonexistent/path/xyz")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestRealFS_Remove_deletesFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "to_remove.txt")
	os.WriteFile(path, []byte("data"), 0644)

	fs := NewRealFS()
	if err := fs.Remove(path); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("file should have been removed")
	}
}

func TestRealFS_RemoveAll_deletesDirectoryTree(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	subdir := filepath.Join(dir, "subdir")
	os.MkdirAll(subdir, 0755)
	os.WriteFile(filepath.Join(subdir, "f.txt"), []byte("x"), 0644)

	fs := NewRealFS()
	if err := fs.RemoveAll(subdir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(subdir); !os.IsNotExist(err) {
		t.Error("directory should have been removed")
	}
}

func TestRealFS_Stat_returnsFileInfo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "stat_test.txt")
	os.WriteFile(path, []byte("hello world"), 0644)

	fs := NewRealFS()
	info, err := fs.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 11 {
		t.Errorf("expected size 11, got %d", info.Size())
	}
	if info.IsDir() {
		t.Error("expected not a directory")
	}
}

func TestRealFS_Stat_nonexistent(t *testing.T) {
	t.Parallel()
	fs := NewRealFS()
	_, err := fs.Stat("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestRealFS_Statfs_returnsStorageInfo(t *testing.T) {
	t.Parallel()
	fs := NewRealFS()
	total, bsize, free := fs.Statfs("/")
	if total == 0 {
		t.Error("expected non-zero total")
	}
	if bsize == 0 {
		t.Error("expected non-zero block size")
	}
	if free >= total && total > 0 {
		t.Error("free should be less than or equal to total")
	}
}

func TestRealFS_Statfs_nonexistentPath(t *testing.T) {
	t.Parallel()
	fs := NewRealFS()
	total, _, _ := fs.Statfs("/nonexistent/path/abc123")
	if total != 0 {
		t.Error("expected total=0 for nonexistent path")
	}
}

func TestRealFS_CreateTemp_createsFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	fs := NewRealFS()
	f, err := fs.CreateTemp(dir, "test-*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	if info, _ := os.Stat(f.Name()); info == nil {
		t.Error("temp file should exist")
	}
}

func TestRealFS_MkdirAll_createsDirectories(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")
	fs := NewRealFS()
	if err := fs.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(nested); info == nil || !info.IsDir() {
		t.Error("nested directory should exist and be a directory")
	}
}

func TestRealFS_Walk_walksDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("b"), 0644)

	fs := NewRealFS()
	count := 0
	err := fs.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count < 3 {
		t.Errorf("expected at least 3 entries (dir, sub, a.txt, b.txt), got %d", count)
	}
}

func TestRealFS_Open_opensFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "open_test.txt")
	os.WriteFile(path, []byte("content"), 0644)

	fs := NewRealFS()
	f, err := fs.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
}

func TestRealFS_Open_nonexistent(t *testing.T) {
	t.Parallel()
	fs := NewRealFS()
	_, err := fs.Open("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestRealFS_Create_createsNewFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "new_file.txt")

	fs := NewRealFS()
	f, err := fs.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("new content"))
	f.Close()

	data, _ := os.ReadFile(path)
	if string(data) != "new content" {
		t.Errorf("expected 'new content', got '%s'", string(data))
	}
}

func TestRealFS_ReadFile_readsContent(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "read_test.txt")
	os.WriteFile(path, []byte("sample data"), 0644)

	fs := NewRealFS()
	data, err := fs.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "sample data" {
		t.Errorf("expected 'sample data', got '%s'", string(data))
	}
}

func TestRealFS_ReadFile_nonexistent(t *testing.T) {
	t.Parallel()
	fs := NewRealFS()
	_, err := fs.ReadFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
