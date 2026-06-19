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

func TestParseConfigYAMLNewFields(t *testing.T) {
	data := []byte(`host: 127.0.0.1
port: 8443
max_read_size_bytes: 5242880
max_write_size_bytes: 5242880
read_timeout_seconds: 15
write_timeout_seconds: 60
idle_timeout_seconds: 30
read_header_timeout_seconds: 5
`)
	cfg, err := parseConfigYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MaxReadSizeBytes != 5242880 {
		t.Fatalf("unexpected max_read_size_bytes: %d", cfg.MaxReadSizeBytes)
	}
	if cfg.ReadTimeoutSeconds != 15 {
		t.Fatalf("unexpected read_timeout_seconds: %d", cfg.ReadTimeoutSeconds)
	}
	if cfg.ReadHeaderTimeoutSeconds != 5 {
		t.Fatalf("unexpected read_header_timeout_seconds: %d", cfg.ReadHeaderTimeoutSeconds)
	}
}

func TestParseConfigYAMLInlineComments(t *testing.T) {
	data := []byte(`host: 127.0.0.1 # bind address
port: 9000 # main port
allowed_roots:
  - /tmp # scratch
allowed_origins:
  - "*" # allow all
`)
	cfg, err := parseConfigYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "127.0.0.1" {
		t.Fatalf("inline comment leaked into host: %q", cfg.Host)
	}
	if cfg.Port != 9000 {
		t.Fatalf("inline comment broke port parsing: %d", cfg.Port)
	}
	if len(cfg.AllowedRoots) != 1 || cfg.AllowedRoots[0] != "/tmp" {
		t.Fatalf("inline comment leaked into root: %v", cfg.AllowedRoots)
	}
	if len(cfg.AllowedOrigins) != 1 || cfg.AllowedOrigins[0] != "*" {
		t.Fatalf("inline comment broke origin: %v", cfg.AllowedOrigins)
	}
}

func TestParseConfigYAMLUnknownKeysIgnored(t *testing.T) {
	data := []byte(`host: 127.0.0.1
port: 8443
log_level: debug
future_field: some_value
unknown_list:
  - item1
  - item2
`)
	cfg, err := parseConfigYAML(data)
	if err != nil {
		t.Fatalf("unknown keys should be silently ignored, got error: %v", err)
	}
	if cfg.Host != "127.0.0.1" || cfg.Port != 8443 {
		t.Fatalf("unexpected host/port after unknown keys: %s %d", cfg.Host, cfg.Port)
	}
}

func TestMarshalConfigYAMLRoundTrip(t *testing.T) {
	original := &Config{
		Host:                     "127.0.0.1",
		Port:                     8443,
		AllowedRoots:             []string{"/home/me"},
		AllowedOrigins:           []string{"*"},
		MaxReadSizeBytes:         5242880,
		MaxWriteSizeBytes:        5242880,
		ReadTimeoutSeconds:       15,
		WriteTimeoutSeconds:      60,
		IdleTimeoutSeconds:       30,
		ReadHeaderTimeoutSeconds: 5,
	}

	data, err := marshalConfigYAML(original)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := parseConfigYAML(data)
	if err != nil {
		t.Fatalf("round-trip failed to parse: %v\nYAML:\n%s", err, data)
	}
	if parsed.Host != original.Host || parsed.Port != original.Port {
		t.Fatalf("round trip host/port mismatch: %+v", parsed)
	}
	if len(parsed.AllowedRoots) != 1 || parsed.AllowedRoots[0] != "/home/me" {
		t.Fatalf("round trip roots mismatch: %v", parsed.AllowedRoots)
	}
	if len(parsed.AllowedOrigins) != 1 || parsed.AllowedOrigins[0] != "*" {
		t.Fatalf("round trip origins mismatch: %v", parsed.AllowedOrigins)
	}
	if parsed.MaxReadSizeBytes != original.MaxReadSizeBytes {
		t.Fatalf("round trip max_read_size_bytes mismatch: %d", parsed.MaxReadSizeBytes)
	}
	if parsed.ReadTimeoutSeconds != original.ReadTimeoutSeconds {
		t.Fatalf("round trip read_timeout_seconds mismatch: %d", parsed.ReadTimeoutSeconds)
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
