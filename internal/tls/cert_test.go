package tls

import (
	"path/filepath"
	"testing"
)

func TestEnsureCreatesCert(t *testing.T) {
	dir := t.TempDir()
	certFile := filepath.Join(dir, "cert.pem")
	keyFile := filepath.Join(dir, "key.pem")

	if err := Ensure(certFile, keyFile); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	if err := Ensure(certFile, keyFile); err != nil {
		t.Fatalf("second ensure: %v", err)
	}
}
