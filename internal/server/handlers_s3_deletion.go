package server

import (
	"github.com/drive/drive/internal/service"
)

func (c *FileCtl) enqueueS3Deletion(fileID, userID, filename string) {
	refs, err := c.ThumbnailStore.FindThumbnailRefsByFileID(fileID)
	if err != nil {
		return
	}

	thumbs := make([]service.S3ThumbItem, len(refs))
	for i, ref := range refs {
		thumbs[i] = service.S3ThumbItem{Size: ref.Size, Format: ref.Format}
	}

	task := &service.S3DeleteTask{
		FileID:   fileID,
		UserID:   userID,
		Filename: filename,
		Thumbs:   thumbs,
	}
	c.S3DeletionPool.Enqueue(task)
}
