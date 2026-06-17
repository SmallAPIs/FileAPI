package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseConfigYAML(t *testing.T) {
	data := []byte(`host: 127.0.0.1
port: 9443

allowed_roots:
  - /home/me
  - /data

allowed_origins:
  - "https://app.example.com"
  - '*'

cert_file: /tmp/cert.pem
key_file: /tmp/key.pem
`)

	cfg, err := parseConfigYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != 9443 {
		t.Fatalf("unexpected host/port: %s %d", cfg.Host, cfg.Port)
	}
	if len(cfg.AllowedRoots) != 2 || cfg.AllowedRoots[0] != "/home/me" {
		t.Fatalf("unexpected roots: %v", cfg.AllowedRoots)
	}
	if len(cfg.AllowedOrigins) != 2 || cfg.AllowedOrigins[1] != "*" {
		t.Fatalf("unexpected origins: %v", cfg.AllowedOrigins)
	}
	if cfg.CertFile != "/tmp/cert.pem" || cfg.KeyFile != "/tmp/key.pem" {
		t.Fatalf("unexpected cert paths: %s %s", cfg.CertFile, cfg.KeyFile)
	}
}

func TestMarshalConfigYAMLRoundTrip(t *testing.T) {
	original := &Config{
		Host:           "127.0.0.1",
		Port:           8443,
		AllowedRoots:   []string{"/home/me"},
		AllowedOrigins: []string{"*"},
	}

	data, err := marshalConfigYAML(original)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := parseConfigYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Host != original.Host || parsed.Port != original.Port {
		t.Fatalf("round trip host/port mismatch: %+v", parsed)
	}
	if len(parsed.AllowedRoots) != 1 || parsed.AllowedRoots[0] != "/home/me" {
		t.Fatalf("round trip roots mismatch: %v", parsed.AllowedRoots)
	}
}

func TestLoadIgnoresCommentsAndBlankLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `# dev config

host: 127.0.0.1
port: 9000

allowed_roots:
  - /tmp

allowed_origins:
  - "*"
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 9000 {
		t.Fatalf("expected port 9000, got %d", cfg.Port)
	}
	if !strings.HasSuffix(cfg.AllowedRoots[0], "/tmp") {
		t.Fatalf("unexpected root: %v", cfg.AllowedRoots)
	}
}
