package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/SmallAPIs/FileAPI/internal/filesystem"
)

// FilesHandler serves file CRUD endpoints.
type FilesHandler struct {
	fs *filesystem.Service
}

// NewFilesHandler creates a handler backed by the filesystem service.
func NewFilesHandler(fs *filesystem.Service) *FilesHandler {
	return &FilesHandler{fs: fs}
}

type createFileRequest struct {
	Path       string `json:"path"`
	Content    string `json:"content"`
	CreateDirs bool   `json:"create_dirs"`
}

type editFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Mode    string `json:"mode"`
}

// Read handles GET /files?path=...
func (h *FilesHandler) Read(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	info, err := h.fs.Read(path)
	if err != nil {
		writeFileError(w, err)
		return
	}
	WriteOK(w, info)
}

// Create handles POST /files.
func (h *FilesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	info, err := h.fs.Create(req.Path, req.Content, req.CreateDirs)
	if err != nil {
		writeFileError(w, err)
		return
	}
	WriteOK(w, info)
}

// Edit handles PATCH /files.
func (h *FilesHandler) Edit(w http.ResponseWriter, r *http.Request) {
	var req editFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_json", "request body must be valid JSON")
		return
	}

	info, err := h.fs.Edit(req.Path, req.Content, req.Mode)
	if err != nil {
		writeFileError(w, err)
		return
	}
	WriteOK(w, info)
}

// Delete handles DELETE /files?path=...
func (h *FilesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if err := h.fs.Delete(path); err != nil {
		writeFileError(w, err)
		return
	}
	WriteOK(w, map[string]string{"path": path, "deleted": "true"})
}

func writeFileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, filesystem.ErrPathNotAllowed):
		WriteError(w, http.StatusForbidden, "path_not_allowed", err.Error())
	case errors.Is(err, filesystem.ErrPathTraversal):
		WriteError(w, http.StatusBadRequest, "path_traversal", err.Error())
	case errors.Is(err, filesystem.ErrNotFound):
		WriteError(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, filesystem.ErrBinaryContent):
		WriteError(w, http.StatusUnsupportedMediaType, "binary_content", err.Error())
	case errors.Is(err, filesystem.ErrTooLarge):
		WriteError(w, http.StatusRequestEntityTooLarge, "file_too_large", err.Error())
	default:
		WriteError(w, http.StatusBadRequest, "file_error", err.Error())
	}
}
