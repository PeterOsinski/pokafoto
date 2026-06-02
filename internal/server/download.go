package server

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) handleBatchDownloadReal(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)

	var req struct {
		FileIDs []string `json:"file_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if len(req.FileIDs) == 0 {
		writeError(w, http.StatusBadRequest, "NO_FILES", "No file IDs provided")
		return
	}

	if len(req.FileIDs) > 100 {
		writeError(w, http.StatusBadRequest, "TOO_MANY", "Maximum 100 files per batch download")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"drive-download.zip\"")
	w.WriteHeader(http.StatusOK)

	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, fileID := range req.FileIDs {
		file, err := s.fileStore.FindByID(fileID)
		if err != nil || file == nil || file.UserID != userID {
			continue
		}

		entryName := file.OriginalName
		writer, err := zw.Create(entryName)
		if err != nil {
			continue
		}

		if file.IsAppManaged {
			doc, err := s.docStore.FindByFileID(fileID)
			if err != nil {
				continue
			}
			io.Copy(writer, strings.NewReader(doc.Content))
			continue
		}

		filePath := filepath.Join(s.cfg.OriginalsDir(), file.UserID, file.Filename)
		f, err := os.Open(filePath)
		if err != nil {
			continue
		}

		io.Copy(writer, f)
		f.Close()
	}
}
