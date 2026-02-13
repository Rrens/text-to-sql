package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Rrens/text-to-sql/internal/api/response"
	"github.com/google/uuid"
)

// UploadHandler handles file upload endpoints
type UploadHandler struct {
	uploadDir string
}

// NewUploadHandler creates a new upload handler
func NewUploadHandler(uploadDir string) *UploadHandler {
	// Ensure upload directory exists
	os.MkdirAll(uploadDir, 0755)
	return &UploadHandler{uploadDir: uploadDir}
}

// UploadSQLite handles SQLite file upload
func (h *UploadHandler) UploadSQLite(w http.ResponseWriter, r *http.Request) {
	// Limit upload to 100MB
	r.ParseMultipartForm(100 << 20)

	file, header, err := r.FormFile("file")
	if err != nil {
		response.BadRequest(w, "no file uploaded")
		return
	}
	defer file.Close()

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{".db": true, ".sqlite": true, ".sqlite3": true, ".db3": true}
	if !allowedExts[ext] {
		response.BadRequest(w, "invalid file type. Allowed: .db, .sqlite, .sqlite3, .db3")
		return
	}

	// Generate unique filename to avoid collisions
	uniqueName := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	destPath := filepath.Join(h.uploadDir, uniqueName)

	// Create destination file
	dst, err := os.Create(destPath)
	if err != nil {
		response.InternalError(w, "failed to save file")
		return
	}
	defer dst.Close()

	// Copy uploaded file to destination
	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(destPath) // cleanup on error
		response.InternalError(w, "failed to save file")
		return
	}

	// Return the absolute path for the SQLite adapter
	absPath, _ := filepath.Abs(destPath)

	response.OK(w, map[string]any{
		"file_path":     absPath,
		"original_name": header.Filename,
		"size":          header.Size,
	})
}
