package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	chunkSize     = 5 * 1024 * 1024
	maxRetries    = 3
	smallFileSize = 5 * 1024 * 1024
)

type importCmd struct {
	serverURL   string
	token       string
	client      *http.Client
	concurrency int
	dryRun      bool
	folderCache map[string]string
	mu          sync.Mutex
	stats       importStats
}

type importStats struct {
	mu       sync.Mutex
	total    int
	uploaded int
	skipped  int
	failed   int
}

type folderTreeResponse struct {
	Folder   *folderTreeItem       `json:"folder"`
	Children []*folderTreeResponse `json:"children,omitempty"`
}

type folderTreeItem struct {
	ID        string  `json:"id"`
	UserID    string  `json:"user_id"`
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type fileItemResponse struct {
	ID           string  `json:"id"`
	OriginalName string  `json:"originalName"`
	SizeBytes    int64   `json:"sizeBytes"`
	FolderID     *string `json:"folder_id,omitempty"`
}

type fileListResponse struct {
	Items      []fileItemResponse `json:"items"`
	NextCursor string             `json:"nextCursor"`
	Total      int                `json:"total"`
}

type fileTask struct {
	localPath string
	filename  string
	size      int64
	folderID  string
}

func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	source := fs.String("source", "", "Local directory to import")
	server := fs.String("server", "http://localhost:8080", "Drive server URL")
	username := fs.String("username", "", "Login username")
	password := fs.String("password", "", "Login password")
	target := fs.String("target", "/", "Target folder path in Drive")
	concurrency := fs.Int("concurrency", 3, "Max parallel file uploads")
	dryRun := fs.Bool("dry-run", false, "Walk and report without uploading")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *source == "" {
		return fmt.Errorf("--source is required")
	}
	if *username == "" {
		return fmt.Errorf("--username is required")
	}
	if *password == "" {
		return fmt.Errorf("--password is required")
	}

	cmd := &importCmd{
		serverURL:   strings.TrimRight(*server, "/"),
		client:      &http.Client{Timeout: 5 * time.Minute},
		concurrency: *concurrency,
		dryRun:      *dryRun,
		folderCache: make(map[string]string),
	}

	if err := cmd.authenticate(*username, *password); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Printf("Resolving target folder %s...", *target)
	targetFolderID, err := cmd.resolveTargetFolder(*target)
	if err != nil {
		fmt.Println()
		return fmt.Errorf("resolve target folder: %w", err)
	}
	fmt.Printf(" done (folder_id=%s)\n", targetFolderID)

	absSource, err := filepath.Abs(*source)
	if err != nil {
		return fmt.Errorf("resolve source path: %w", err)
	}

	return cmd.walkAndImport(absSource, targetFolderID)
}

func (c *importCmd) authenticate(username, password string) error {
	body := map[string]string{"username": username, "password": password}
	payload, _ := json.Marshal(body)

	resp, err := c.doRequest("POST", "/api/v1/auth/login", payload)
	if err != nil {
		return err
	}

	var authResp struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(resp, &authResp); err != nil {
		return fmt.Errorf("parse auth response: %w", err)
	}

	c.token = authResp.AccessToken
	return nil
}

func (c *importCmd) doRequest(method, path string, body []byte) ([]byte, error) {
	url := c.serverURL + path
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error %d: %s", resp.StatusCode, string(respBody))
			continue
		}

		if resp.StatusCode >= 400 {
			var errResp struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error.Message != "" {
				return nil, fmt.Errorf("%s: %s", errResp.Error.Code, errResp.Error.Message)
			}
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (c *importCmd) resolveTargetFolder(targetPath string) (string, error) {
	targetPath = filepath.ToSlash(filepath.Clean(targetPath))
	if targetPath == "." || targetPath == "/" {
		return "", nil
	}

	parts := strings.Split(strings.Trim(targetPath, "/"), "/")

	treeResp, err := c.doRequest("GET", "/api/v1/folders", nil)
	if err != nil {
		return "", fmt.Errorf("list folders: %w", err)
	}

	var tree folderTreeResponse
	if err := json.Unmarshal(treeResp, &tree); err != nil {
		return "", fmt.Errorf("parse folder tree: %w", err)
	}

	var currentParentID *string

	for _, part := range parts {
		cacheKey := currentParentKey(currentParentID, part)
		if id, ok := c.folderCache[cacheKey]; ok {
			currentParentID = &id
			continue
		}

		var found *string
		for _, child := range tree.Children {
			if child.Folder != nil && strings.EqualFold(child.Folder.Name, part) &&
				parentIDsMatch(child.Folder.ParentID, currentParentID) {
				found = &child.Folder.ID
				break
			}
		}

		if found == nil {
			folderID, err := c.createFolder(currentParentID, part)
			if err != nil {
				return "", fmt.Errorf("create folder %q: %w", part, err)
			}
			found = &folderID
		}

		c.mu.Lock()
		c.folderCache[cacheKey] = *found
		c.mu.Unlock()
		currentParentID = found
	}

	if currentParentID == nil {
		return "", fmt.Errorf("empty target path")
	}
	return *currentParentID, nil
}

func currentParentKey(parentID *string, name string) string {
	pid := "root"
	if parentID != nil {
		pid = *parentID
	}
	return pid + "/" + strings.ToLower(name)
}

func parentIDsMatch(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func (c *importCmd) createFolder(parentID *string, name string) (string, error) {
	if c.dryRun {
		fmt.Printf("  [dry-run] would create folder: %s\n", name)
		return "dry-run-folder-id", nil
	}

	req := map[string]interface{}{"name": name}
	if parentID != nil {
		req["parent_id"] = *parentID
	}
	payload, _ := json.Marshal(req)

	resp, err := c.doRequest("POST", "/api/v1/folders", payload)
	if err != nil {
		return "", err
	}

	var folder folderTreeItem
	if err := json.Unmarshal(resp, &folder); err != nil {
		return "", fmt.Errorf("parse folder response: %w", err)
	}

	return folder.ID, nil
}

func (c *importCmd) walkAndImport(source string, targetFolderID string) error {
	fmt.Printf("Walking %s...\n", source)

	var allTasks []fileTask

	err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("walk error", "path", path, "error", err)
			return nil
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		dirRel := filepath.Dir(relPath)
		filename := filepath.Base(path)
		var folderID string

		if dirRel == "." {
			folderID = targetFolderID
		} else {
			parts := strings.Split(filepath.ToSlash(dirRel), "/")
			currentParentID := targetFolderID
			for _, part := range parts {
				cacheKey := currentParentKey(nonNilPtr(currentParentID), part)
				c.mu.Lock()
				cached, ok := c.folderCache[cacheKey]
				c.mu.Unlock()
				if ok {
					currentParentID = cached
					continue
				}

				fid, err := c.createFolder(nonNilStrPtr(currentParentID), part)
				if err != nil {
					return fmt.Errorf("create folder %q: %w", part, err)
				}
				c.mu.Lock()
				c.folderCache[cacheKey] = fid
				c.mu.Unlock()
				currentParentID = fid
			}
			folderID = currentParentID
		}

		allTasks = append(allTasks, fileTask{
			localPath: path,
			filename:  filename,
			size:      info.Size(),
			folderID:  folderID,
		})

		return nil
	})

	if err != nil {
		return err
	}

	c.stats.total = len(allTasks)
	sem := make(chan struct{}, c.concurrency)
	var wg sync.WaitGroup

	for i, task := range allTasks {
		wg.Add(1)
		go func(idx int, t fileTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			code := c.processFile(t, idx)
			c.stats.mu.Lock()
			switch code {
			case resultUploaded:
				c.stats.uploaded++
			case resultSkipped:
				c.stats.skipped++
			case resultFailed:
				c.stats.failed++
			}
			c.stats.mu.Unlock()
		}(i, task)
	}

	wg.Wait()
	return nil
}

func nonNilPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nonNilStrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

type resultCode int

const (
	resultUploaded resultCode = iota
	resultSkipped
	resultFailed
)

func (c *importCmd) processFile(t fileTask, idx int) resultCode {
	index := fmt.Sprintf("[%d/%d]", idx+1, c.stats.total)

	if c.dryRun {
		exists, err := c.fileExistsInFolder(t.folderID, t.filename, t.size)
		if err != nil {
			fmt.Printf("  %s %s — error checking existence: %v\n", index, t.filename, err)
		} else if exists {
			fmt.Printf("  %s %s — skipped (already exists)\n", index, t.filename)
		} else {
			fmt.Printf("  %s %s — would upload (%s)\n", index, t.filename, humanSize(t.size))
		}
		return resultSkipped
	}

	exists, err := c.fileExistsInFolder(t.folderID, t.filename, t.size)
	if err != nil {
		fmt.Printf("  %s %s — error checking: %v\n", index, t.filename, err)
		return resultFailed
	}
	if exists {
		fmt.Printf("  %s %s — skipped (already exists)\n", index, t.filename)
		return resultSkipped
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
		}
		if t.size <= smallFileSize {
			lastErr = c.uploadSmallFile(t)
		} else {
			lastErr = c.uploadChunkedFile(t)
		}
		if lastErr == nil {
			fmt.Printf("  %s %s — uploaded (%s)\n", index, t.filename, humanSize(t.size))
			return resultUploaded
		}
		if attempt > 0 {
			fmt.Printf("  %s %s — retry %d/%d: %v\n", index, t.filename, attempt, maxRetries, lastErr)
		}
	}

	fmt.Printf("  %s %s — failed: %v\n", index, t.filename, lastErr)
	return resultFailed
}

func (c *importCmd) uploadSmallFile(t fileTask) error {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	part, err := w.CreateFormFile("files", t.filename)
	if err != nil {
		return err
	}

	f, err := os.Open(t.localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(part, f); err != nil {
		return err
	}

	if t.folderID != "" {
		w.WriteField("folder_id", t.folderID)
	}
	w.Close()

	url := c.serverURL + "/api/v1/upload"
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (c *importCmd) uploadChunkedFile(t fileTask) error {
	f, err := os.Open(t.localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	totalChunks := int((t.size + chunkSize - 1) / chunkSize)
	firstChunkSize := int64(chunkSize)
	if t.size < firstChunkSize {
		firstChunkSize = t.size
	}

	firstChunk := make([]byte, firstChunkSize)
	if _, err := io.ReadFull(f, firstChunk); err != nil {
		return err
	}

	resumeToken, uploadID, err := c.sendChunk(t.filename, t.size, totalChunks, 0, t.folderID, firstChunk, nil)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)
	errCh := make(chan error, totalChunks-1)

	for i := 1; i < totalChunks; i++ {
		wg.Add(1)
		go func(chunkIdx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			start := int64(chunkIdx) * chunkSize
			end := start + chunkSize
			if end > t.size {
				end = t.size
			}
			chunkData := make([]byte, end-start)
			_ = readAtFull(f, chunkData, start)

			if _, _, err := c.sendChunk(t.filename, t.size, totalChunks, chunkIdx, t.folderID, chunkData, &resumeToken); err != nil {
				errCh <- err
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for e := range errCh {
		if e != nil {
			return e
		}
	}

	completeBody := map[string]string{"upload_id": uploadID}
	payload, _ := json.Marshal(completeBody)
	if _, err := c.doRequest("POST", "/api/v1/upload/chunk/"+resumeToken+"/complete", payload); err != nil {
		return err
	}

	return c.pollUploadCompletion(uploadID)
}

func (c *importCmd) sendChunk(filename string, totalSize int64, totalChunks, chunkIdx int, folderID string, data []byte, resumeToken *string) (string, string, error) {
	sha256Sum := sha256.Sum256(data)
	sha256Hex := fmt.Sprintf("%x", sha256Sum)

	url := c.serverURL + "/api/v1/upload/chunk"
	req, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Filename", filename)
	req.Header.Set("X-Total-Size", fmt.Sprintf("%d", totalSize))
	req.Header.Set("X-Total-Chunks", fmt.Sprintf("%d", totalChunks))
	req.Header.Set("X-Chunk-Index", fmt.Sprintf("%d", chunkIdx))
	req.Header.Set("X-Chunk-Size", fmt.Sprintf("%d", len(data)))
	req.Header.Set("X-Chunk-SHA256", sha256Hex)
	if folderID != "" {
		req.Header.Set("X-Folder-ID", folderID)
	}
	if resumeToken != nil && *resumeToken != "" {
		req.Header.Set("X-Resume-Token", *resumeToken)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("chunk upload HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var chunkResp struct {
		UploadID    string `json:"upload_id"`
		ResumeToken string `json:"resume_token"`
	}
	json.Unmarshal(respBody, &chunkResp)

	return chunkResp.ResumeToken, chunkResp.UploadID, nil
}

func (c *importCmd) pollUploadCompletion(uploadID string) error {
	maxPolls := 120

	for i := 0; i < maxPolls; i++ {
		time.Sleep(1 * time.Second)

		resp, err := c.doRequest("GET", "/api/v1/upload/active", nil)
		if err != nil {
			continue
		}

		var activeResp struct {
			Jobs []struct {
				JobID  string `json:"job_id"`
				Status string `json:"status"`
				Error  string `json:"error,omitempty"`
			} `json:"jobs"`
		}
		if err := json.Unmarshal(resp, &activeResp); err != nil {
			continue
		}

		for _, job := range activeResp.Jobs {
			if job.JobID == uploadID {
				switch job.Status {
				case "completed", "skipped":
					return nil
				case "failed":
					if job.Error != "" {
						return fmt.Errorf("upload failed: %s", job.Error)
					}
					return fmt.Errorf("upload failed")
				}
			}
		}
	}

	return fmt.Errorf("timed out waiting for upload completion")
}

func (c *importCmd) fileExistsInFolder(folderID string, filename string, size int64) (bool, error) {
	cursor := ""
	for {
		path := "/api/v1/files"
		if cursor != "" {
			path += "?cursor=" + cursor
		}
		if folderID != "" {
			sep := "?"
			if strings.Contains(path, "?") {
				sep = "&"
			}
			path += sep + "folder_id=" + folderID
		}

		resp, err := c.doRequest("GET", path, nil)
		if err != nil {
			return false, err
		}

		var list fileListResponse
		if err := json.Unmarshal(resp, &list); err != nil {
			return false, fmt.Errorf("parse file list: %w", err)
		}

		for _, item := range list.Items {
			if item.OriginalName == filename && item.SizeBytes == size {
				return true, nil
			}
		}

		if list.NextCursor == "" {
			return false, nil
		}
		cursor = list.NextCursor
	}
}

func readAtFull(f *os.File, buf []byte, offset int64) error {
	n, err := f.ReadAt(buf, offset)
	if err != nil {
		return err
	}
	if n < len(buf) {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func humanSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
