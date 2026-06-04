package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/drive/drive/internal/model"
)

var timeRFC3339 = "2006-01-02T15:04:05Z07:00"

type fileResponse struct {
	ID           string                `json:"id"`
	UserID       *string               `json:"user_id,omitempty"`
	Filename     string                `json:"filename"`
	OriginalName string                `json:"originalName"`
	Path         string                `json:"path"`
	SizeBytes    int64                 `json:"sizeBytes"`
	MimeType     string                `json:"mimeType"`
	SHA256       *string               `json:"sha256,omitempty"`
	MediaType    string                `json:"mediaType"`
	Width        *int                  `json:"width,omitempty"`
	Height       *int                  `json:"height,omitempty"`
	DurationSec  *float64              `json:"durationSec,omitempty"`
	TakenAt      *string               `json:"takenAt,omitempty"`
	FolderID     *string               `json:"folder_id,omitempty"`
	CreatedAt    string                `json:"createdAt"`
	UpdatedAt    string                `json:"updatedAt,omitempty"`
	DeletedAt    *string               `json:"deletedAt,omitempty"`
	Thumbnails   *thumbnailSetResponse `json:"thumbnails,omitempty"`
	EXIF         interface{}           `json:"exif,omitempty"`
	DownloadInfo *downloadInfoResponse `json:"downloadInfo,omitempty"`
	IsAppManaged bool                  `json:"isAppManaged"`
}

type downloadInfoResponse struct {
	SizeBytes     int64  `json:"sizeBytes"`
	MimeType      string `json:"mimeType"`
	SupportsRange bool   `json:"supportsRange"`
}

type thumbnailSetResponse struct {
	SM         *thumbnailInfoResponse `json:"sm"`
	LG         *thumbnailInfoResponse `json:"lg"`
	MD         *thumbnailInfoResponse `json:"md"`
	XL         *thumbnailInfoResponse `json:"xl,omitempty"`
	Preview    *thumbnailInfoResponse `json:"preview"`
	VideoStill *thumbnailInfoResponse `json:"videoStill,omitempty"`
	VideoProxy *thumbnailInfoResponse `json:"videoProxy,omitempty"`
}

type thumbnailInfoResponse struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

func buildThumbnailSet(fileID string, mediaType model.MediaType) *thumbnailSetResponse {
	ver := fileID
	if len(ver) > 8 {
		ver = ver[:8]
	}
	if mediaType == model.MediaTypeFile {
		return nil
	}
	if mediaType == model.MediaTypeVideo {
		return &thumbnailSetResponse{
			VideoStill: &thumbnailInfoResponse{
				URL:    fmt.Sprintf("/api/v1/thumb/%s/video_still.jpg?v=%s", fileID, ver),
				Width:  600,
				Height: 338,
			},
			VideoProxy: &thumbnailInfoResponse{
				URL:    fmt.Sprintf("/api/v1/video/%s?quality=proxy", fileID),
				Width:  1280,
				Height: 720,
			},
		}
	}
	return &thumbnailSetResponse{
		SM: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/sm.jpg?v=%s", fileID, ver),
			Width:  60,
			Height: 60,
		},
		LG: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/lg.jpg?v=%s", fileID, ver),
			Width:  300,
			Height: 300,
		},
		MD: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/md.jpg?v=%s", fileID, ver),
			Width:  600,
			Height: 600,
		},
		XL: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/xl.jpg?v=%s", fileID, ver),
			Width:  2000,
			Height: 2000,
		},
		Preview: &thumbnailInfoResponse{
			URL:    fmt.Sprintf("/api/v1/thumb/%s/preview.webp?v=%s", fileID, ver),
			Width:  1280,
			Height: 720,
		},
	}
}

func formatDeletedAt(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(timeRFC3339)
	return &s
}

func folderIDFromQuery(r *http.Request) *string {
	v := r.URL.Query().Get("folder_id")
	if v == "" {
		return nil
	}
	s := new(string)
	if v == "root" {
		*s = ""
	} else {
		*s = v
	}
	return s
}
