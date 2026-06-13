// Package handlers implements REST endpoints for files and system actions.
package handlers

import (
	"encoding/json"
	"net/http"
)

// APIError is the error object in JSON responses.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error *APIError   `json:"error,omitempty"`
}

// WriteOK sends a successful JSON envelope.
func WriteOK(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, envelope{OK: true, Data: data})
}

// WriteError sends a failed JSON envelope with the given HTTP status.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, envelope{
		OK: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
