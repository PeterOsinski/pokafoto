package server

import (
	"testing"

	"github.com/drive/drive/internal/worker"
)

func TestWorkerJobToMsg_shouldIncludeFolderIDWhenSet(t *testing.T) {
	folderID := "folder-uuid-123"
	job := &worker.UploadJob{
		JobID:    "job-1",
		Filename: "test.jpg",
		Status:   worker.JobCompleted,
		Progress: 1.0,
		FileID:   "file-1",
		FolderID: &folderID,
	}

	msg := workerJobToMsg(job)

	if v, ok := msg["folder_id"]; !ok {
		t.Error("expected folder_id to be present")
	} else if v != folderID {
		t.Errorf("expected folder_id to be %q, got %v", folderID, v)
	}
}

func TestWorkerJobToMsg_shouldOmitFolderIDWhenNil(t *testing.T) {
	job := &worker.UploadJob{
		JobID:    "job-2",
		Filename: "test.jpg",
		Status:   worker.JobCompleted,
		Progress: 1.0,
		FileID:   "file-2",
		FolderID: nil,
	}

	msg := workerJobToMsg(job)

	if _, ok := msg["folder_id"]; ok {
		t.Error("expected folder_id to be absent when nil")
	}
}

func TestWorkerJobToMsg_shouldIncludeFileIDWhenPresent(t *testing.T) {
	job := &worker.UploadJob{
		JobID:    "job-3",
		Filename: "test.jpg",
		Status:   worker.JobCompleted,
		Progress: 1.0,
		FileID:   "file-3",
	}

	msg := workerJobToMsg(job)

	if v, ok := msg["file_id"]; !ok {
		t.Error("expected file_id to be present")
	} else if v != "file-3" {
		t.Errorf("expected file_id %q, got %v", "file-3", v)
	}
}
