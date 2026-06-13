package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/SmallAPIs/FileAPI/internal/platform"
)

// SystemHandler serves desktop integration endpoints.
type SystemHandler struct {
	desktop platform.Desktop
}

// NewSystemHandler creates a handler backed by the platform desktop service.
func NewSystemHandler(desktop platform.Desktop) *SystemHandler {
	return &SystemHandler{desktop: desktop}
}

type openAppRequest struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type openURLRequest struct {
	URL string `json:"url"`
}

// OpenApp handles POST /system/open-app.
func (h *SystemHandler) OpenApp(w http.ResponseWriter, r *http.Request) {
	var req openAppRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON")
		return
	}

	target := strings.TrimSpace(req.Path)
	if target == "" {
		target = strings.TrimSpace(req.Name)
	}
	if target == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "either name or path is required")
		return
	}

	if err := h.desktop.OpenApp(target); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	WriteOK(w, map[string]string{"opened": target})
}

// OpenURL handles POST /system/open-url.
func (h *SystemHandler) OpenURL(w http.ResponseWriter, r *http.Request) {
	var req openURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "request body must be valid JSON")
		return
	}

	rawURL := strings.TrimSpace(req.URL)
	if rawURL == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "url is required")
		return
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_URL", "url must be a valid absolute URL")
		return
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		WriteError(w, http.StatusBadRequest, "INVALID_URL", "only http and https URLs are allowed")
		return
	}

	if err := h.desktop.OpenURL(rawURL); err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	WriteOK(w, map[string]string{"url": rawURL})
}
