package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenURLValidation(t *testing.T) {
	h := NewSystemHandler(&fakeDesktop{})

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"valid https", `{"url":"https://example.com"}`, http.StatusOK},
		{"invalid scheme", `{"url":"file:///etc/passwd"}`, http.StatusBadRequest},
		{"missing url", `{}`, http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/system/open-url", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()
			h.OpenURL(rec, req)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status %d, body %s", rec.Code, rec.Body.String())
			}
		})
	}
}

type fakeDesktop struct{}

func (f *fakeDesktop) OpenURL(url string) error { return nil }
func (f *fakeDesktop) OpenApp(nameOrPath string) error { return nil }
