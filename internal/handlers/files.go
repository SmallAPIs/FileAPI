package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON")
		return
	}

	info, err := h.fs.Create(req.Path, req.Content, req.CreateDirs)
	if err != nil {
		writeFileError(w, err)
		return
	}
	WriteCreated(w, info)
}

// Edit handles PATCH /files.
func (h *FilesHandler) Edit(w http.ResponseWriter, r *http.Request) {
	var req editFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON")
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
	case errors.Is(err, filesystem.ErrPathNotAllowed), errors.Is(err, filesystem.ErrPathTraversal):
		WriteError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, filesystem.ErrNotFound):
		WriteError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, filesystem.ErrBinaryContent):
		WriteError(w, http.StatusUnsupportedMediaType, "BINARY_NOT_SUPPORTED", err.Error())
	case errors.Is(err, filesystem.ErrTooLarge):
		WriteError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", err.Error())
	default:
		if strings.Contains(err.Error(), "already exists") {
			WriteError(w, http.StatusConflict, "ALREADY_EXISTS", err.Error())
			return
		}
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
}
