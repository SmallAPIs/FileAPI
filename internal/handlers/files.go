package handlers

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/SmallAPIs/FileAPI/internal/filesystem"
)

const jsonBodyOverhead = 512 * 1024 // room for path/mode fields beyond content

// FilesHandler serves file CRUD endpoints.
type FilesHandler struct {
	fs           *filesystem.Service
	maxReadSize  int64
	maxWriteSize int64
	maxBodySize  int64
}

// NewFilesHandler creates a handler backed by the filesystem service.
func NewFilesHandler(fs *filesystem.Service, maxReadSize, maxWriteSize int64) *FilesHandler {
	if maxReadSize <= 0 {
		maxReadSize = 10 * 1024 * 1024
	}
	if maxWriteSize <= 0 {
		maxWriteSize = maxReadSize
	}
	return &FilesHandler{
		fs:           fs,
		maxReadSize:  maxReadSize,
		maxWriteSize: maxWriteSize,
		maxBodySize:  maxWriteSize + jsonBodyOverhead,
	}
}

type createFileRequest struct {
	Path           string `json:"path"`
	Content        string `json:"content"`
	CreateDirs     bool   `json:"create_dirs"`
	IncludeContent *bool  `json:"include_content"`
}

type editFileRequest struct {
	Path           string `json:"path"`
	Content        string `json:"content"`
	Mode           string `json:"mode"`
	IncludeContent *bool  `json:"include_content"`
}

func includeContentValue(v *bool) bool {
	if v == nil {
		return true
	}
	return *v
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

// ReadRaw handles GET /files/raw?path=... with optional gzip compression.
func (h *FilesHandler) ReadRaw(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	info, err := h.fs.Stat(path)
	if err != nil {
		writeFileError(w, err)
		return
	}
	if info.Size > h.maxReadSize {
		WriteError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", filesystem.ErrTooLarge.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-File-Path", info.Path)
	w.Header().Set("X-File-Size", strconv.FormatInt(info.Size, 10))

	var writer io.Writer = w
	if acceptsGzip(r) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		writer = gz
	}

	w.WriteHeader(http.StatusOK)

	if _, err := h.fs.StreamRaw(path, writer); err != nil {
		// Response may already be partially written; best-effort logging only.
		return
	}
}

// Create handles POST /files.
func (h *FilesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createFileRequest
	if err := decodeJSONBody(w, r, h.maxBodySize, &req); err != nil {
		return
	}

	info, err := h.fs.Create(req.Path, req.Content, req.CreateDirs, includeContentValue(req.IncludeContent))
	if err != nil {
		writeFileError(w, err)
		return
	}
	WriteCreated(w, info)
}

// Edit handles PATCH /files.
func (h *FilesHandler) Edit(w http.ResponseWriter, r *http.Request) {
	var req editFileRequest
	if err := decodeJSONBody(w, r, h.maxBodySize, &req); err != nil {
		return
	}

	info, err := h.fs.Edit(req.Path, req.Content, req.Mode, includeContentValue(req.IncludeContent))
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

func decodeJSONBody(w http.ResponseWriter, r *http.Request, maxBytes int64, dest interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	if err := json.NewDecoder(r.Body).Decode(dest); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			WriteError(w, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "request body exceeds maximum size")
			return err
		}
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON")
		return err
	}
	return nil
}

func writeFileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, filesystem.ErrPathNotAllowed), errors.Is(err, filesystem.ErrPathTraversal):
		WriteError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, filesystem.ErrNotFound):
		WriteError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, filesystem.ErrBinaryContent):
		WriteError(w, http.StatusUnsupportedMediaType, "BINARY_NOT_SUPPORTED", err.Error())
	case errors.Is(err, filesystem.ErrTooLarge), errors.Is(err, filesystem.ErrContentTooLarge):
		WriteError(w, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", err.Error())
	default:
		if strings.Contains(err.Error(), "already exists") {
			WriteError(w, http.StatusConflict, "ALREADY_EXISTS", err.Error())
			return
		}
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
}

func acceptsGzip(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
}
