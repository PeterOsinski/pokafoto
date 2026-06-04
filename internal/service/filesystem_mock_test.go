package service

import (
	"os"
	"testing"
)

func TestMockFS_Stat_existingFile(t *testing.T) {
	fs := NewMockFS()
	fs.AddFile("/tmp/test.txt", []byte("hello"))

	info, err := fs.Stat("/tmp/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 5 {
		t.Errorf("expected size 5, got %d", info.Size())
	}
	if info.IsDir() {
		t.Error("expected not a directory")
	}
}

func TestMockFS_Stat_missingFile(t *testing.T) {
	fs := NewMockFS()

	_, err := fs.Stat("/nonexistent/file")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMockFS_Stat_removedFile(t *testing.T) {
	fs := NewMockFS()
	fs.AddFile("/tmp/removed.txt", []byte("data"))
	fs.Remove("/tmp/removed.txt")

	_, err := fs.Stat("/tmp/removed.txt")
	if err == nil {
		t.Fatal("expected error for removed file")
	}
}

func TestMockFS_ReadDir_empty(t *testing.T) {
	fs := NewMockFS()
	entries, err := fs.ReadDir("/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestMockFS_ReadDir_withFiles(t *testing.T) {
	fs := NewMockFS()
	fs.AddFile("/tmp/a.txt", []byte("a"))
	fs.AddFile("/tmp/b.txt", []byte("bb"))

	entries, err := fs.ReadDir("/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestMockFS_ReadDir_sorted(t *testing.T) {
	fs := NewMockFS()
	fs.AddFile("/tmp/c.txt", []byte("c"))
	fs.AddFile("/tmp/a.txt", []byte("a"))

	entries, _ := fs.ReadDir("/tmp")
	if entries[0].Name() > entries[1].Name() {
		t.Errorf("expected sorted entries, got %s before %s", entries[0].Name(), entries[1].Name())
	}
}

func TestMockFS_RemoveAll(t *testing.T) {
	fs := NewMockFS()
	fs.AddFile("/tmp/subdir/a.txt", []byte("a"))
	fs.AddFile("/tmp/subdir/b.txt", []byte("b"))
	fs.AddFile("/tmp/other.txt", []byte("other"))

	fs.RemoveAll("/tmp/subdir")

	_, err := fs.Stat("/tmp/subdir/a.txt")
	if err == nil {
		t.Error("expected error for removed file a.txt")
	}
	_, err = fs.Stat("/tmp/subdir/b.txt")
	if err == nil {
		t.Error("expected error for removed file b.txt")
	}

	_, err = fs.Stat("/tmp/other.txt")
	if err != nil {
		t.Errorf("other.txt should still exist: %v", err)
	}
}

func TestMockFS_Walk(t *testing.T) {
	fs := NewMockFS()
	fs.AddFile("/tmp/a.txt", []byte("a"))
	fs.AddFile("/tmp/b.txt", []byte("bb"))

	count := 0
	err := fs.Walk("/tmp", func(path string, info os.FileInfo, err error) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("expected 2 files walked, got %d", count)
	}
}

func TestMockFS_ReadFile(t *testing.T) {
	fs := NewMockFS()
	fs.AddFile("/tmp/test.txt", []byte("hello world"))

	data, err := fs.ReadFile("/tmp/test.txt")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", string(data))
	}
}

func TestMockFS_ReadFile_missing(t *testing.T) {
	fs := NewMockFS()

	_, err := fs.ReadFile("/missing.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestMockFS_MkdirAll(t *testing.T) {
	fs := NewMockFS()
	fs.MkdirAll("/tmp/newdir", 0o755)

	info, err := fs.Stat("/tmp/newdir")
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestMockFS_Statfs(t *testing.T) {
	fs := NewMockFS()
	total, bsize, free := fs.Statfs("/")
	if total == 0 {
		t.Error("expected non-zero total")
	}
	if bsize == 0 {
		t.Error("expected non-zero block size")
	}
	if free >= total {
		t.Error("free should be less than total")
	}
}
