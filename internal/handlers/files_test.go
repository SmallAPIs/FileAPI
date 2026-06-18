package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SmallAPIs/FileAPI/internal/filesystem"
)

func newTestFilesHandler(t *testing.T) (*FilesHandler, string) {
	t.Helper()
	root := t.TempDir()
	fs, err := filesystem.NewService([]string{root}, 1024*1024, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}
	return NewFilesHandler(fs, 1024*1024, 1024*1024), root
}

func TestCreateIncludeContentFalse(t *testing.T) {
	h, root := newTestFilesHandler(t)
	path := filepath.Join(root, "note.txt")

	body := `{"path":` + jsonString(path) + `,"content":"hello","include_content":false}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status: got %d body %s", rec.Code, rec.Body.String())
	}

	var resp envelope
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected data type: %T", resp.Data)
	}
	if _, hasContent := data["content"]; hasContent {
		t.Fatalf("expected content omitted, got %v", data["content"])
	}
}

func TestCreateRejectsOversizedBody(t *testing.T) {
	root := t.TempDir()
	fs, err := filesystem.NewService([]string{root}, 1024*1024, 128)
	if err != nil {
		t.Fatal(err)
	}
	h := NewFilesHandler(fs, 1024*1024, 128)

	body := bytes.NewBufferString(`{"path":"` + filepath.Join(root, "x.txt") + `","content":"`)
	body.WriteString(strings.Repeat("a", 256))
	body.WriteString(`"}`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/files", body)
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", rec.Code)
	}
}

func TestReadRawStreamsContent(t *testing.T) {
	h, root := newTestFilesHandler(t)
	path := filepath.Join(root, "raw.txt")
	if _, err := h.fs.Create(path, "plain text", false, true); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/raw?path="+path, nil)
	rec := httptest.NewRecorder()
	h.ReadRaw(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d body %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "plain text" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
	if rec.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", rec.Header().Get("Content-Type"))
	}
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
