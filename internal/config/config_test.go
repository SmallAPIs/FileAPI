package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Host != "127.0.0.1" {
		t.Fatalf("unexpected host: %s", cfg.Host)
	}
	if cfg.Port != 8443 {
		t.Fatalf("unexpected port: %d", cfg.Port)
	}
	if len(cfg.AllowedRoots) != 1 {
		t.Fatalf("expected one allowed root")
	}
	if cfg.MaxReadSizeBytes != defaultMaxReadSize {
		t.Fatalf("unexpected max read size: %d", cfg.MaxReadSizeBytes)
	}
	if cfg.WriteTimeoutSeconds != int(defaultWriteTimeout/time.Second) {
		t.Fatalf("unexpected write timeout: %d", cfg.WriteTimeoutSeconds)
	}
}

func TestLoadAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Port = 9443
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Port != 9443 {
		t.Fatalf("expected port 9443, got %d", loaded.Port)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file missing: %v", err)
	}
}
