package server

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"log/slog"
)

type spaHandler struct {
	root       string
	staticPath string
	indexPath  string
}

func newSPAHandler(webDir string) *spaHandler {
	return &spaHandler{
		root:       webDir,
		staticPath: "assets/",
		indexPath:  "index.html",
	}
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(h.root, r.URL.Path)

	if strings.HasPrefix(r.URL.Path, "/assets/") || strings.HasPrefix(r.URL.Path, "/vite.") {
		if _, err := os.Stat(path); err == nil {
			http.FileServer(http.Dir(h.root)).ServeHTTP(w, r)
			return
		}
	}

	indexPath := filepath.Join(h.root, h.indexPath)
	indexContent, err := os.ReadFile(indexPath)
	if err != nil {
		slog.Error("failed to read index.html", "error", err)
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Page not found")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexContent)
}

func (s *Server) serveSPA() http.Handler {
	webDir := s.cfg.WebDistPath()
	for _, dir := range []string{webDir, "/app/web/dist", "web/dist"} {
		if _, err := os.Stat(dir); err == nil {
			slog.Info("serving SPA", "path", dir)
			return newSPAHandler(dir)
		}
	}
	slog.Warn("SPA not found, API-only mode")
	return nil
}
