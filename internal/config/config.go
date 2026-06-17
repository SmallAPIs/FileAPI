// Package config loads FileAPI agent settings from YAML and environment.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	defaultHost = "127.0.0.1"
	defaultPort = 8443
)

// Config holds runtime settings for the local FileAPI agent.
type Config struct {
	Host           string
	Port           int
	AllowedRoots   []string
	AllowedOrigins []string
	CertFile       string
	KeyFile        string
	ConfigDir      string
	ConfigPath     string
}

// DefaultConfig returns settings suitable for local development.
func DefaultConfig() *Config {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	configDir := defaultConfigDir()
	cfg := &Config{
		Host:           defaultHost,
		Port:           defaultPort,
		AllowedRoots:   []string{home},
		AllowedOrigins: []string{"*"},
		ConfigDir:      configDir,
		ConfigPath:     filepath.Join(configDir, "config.yaml"),
		CertFile:       filepath.Join(configDir, "cert.pem"),
		KeyFile:        filepath.Join(configDir, "key.pem"),
	}
	return cfg
}

func defaultConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("APPDATA")
		if base == "" {
			home, _ := os.UserHomeDir()
			base = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(base, "FileAPI")
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "FileAPI"
		}
		return filepath.Join(home, "Library", "Application Support", "FileAPI")
	default:
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".config", "fileapi")
		}
		return filepath.Join(home, ".config", "fileapi")
	}
}

// Load reads configuration from path, falling back to defaults for missing fields.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	if path == "" {
		path = cfg.ConfigPath
	}
	cfg.ConfigPath = path
	cfg.ConfigDir = filepath.Dir(path)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	parsed, err := parseConfigYAML(data)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	mergeConfig(cfg, parsed)

	cfg.applyDefaults()
	return cfg, nil
}

func mergeConfig(dst, src *Config) {
	if src.Host != "" {
		dst.Host = src.Host
	}
	if src.Port != 0 {
		dst.Port = src.Port
	}
	if len(src.AllowedRoots) > 0 {
		dst.AllowedRoots = src.AllowedRoots
	}
	if len(src.AllowedOrigins) > 0 {
		dst.AllowedOrigins = src.AllowedOrigins
	}
	if src.CertFile != "" {
		dst.CertFile = src.CertFile
	}
	if src.KeyFile != "" {
		dst.KeyFile = src.KeyFile
	}
}

func (c *Config) applyDefaults() {
	if c.Host == "" {
		c.Host = defaultHost
	}
	if c.Port == 0 {
		c.Port = defaultPort
	}
	if len(c.AllowedRoots) == 0 {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		c.AllowedRoots = []string{home}
	}
	if len(c.AllowedOrigins) == 0 {
		c.AllowedOrigins = []string{"*"}
	}
	if c.CertFile == "" {
		c.CertFile = filepath.Join(c.ConfigDir, "cert.pem")
	}
	if c.KeyFile == "" {
		c.KeyFile = filepath.Join(c.ConfigDir, "key.pem")
	}
}

// Save writes the current configuration to disk.
func (c *Config) Save() error {
	if err := os.MkdirAll(c.ConfigDir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := marshalConfigYAML(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(c.ConfigPath, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// ListenAddr returns the host:port bind address.
func (c *Config) ListenAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// BaseURL returns the HTTPS API base URL.
func (c *Config) BaseURL() string {
	return fmt.Sprintf("https://%s:%d/api/v1", c.Host, c.Port)
}
