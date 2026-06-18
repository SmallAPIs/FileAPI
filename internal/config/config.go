// Package config loads FileAPI agent settings from YAML and environment.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultHost              = "127.0.0.1"
	defaultPort              = 8443
	defaultMaxReadSize       = 10 * 1024 * 1024 // 10 MB
	defaultMaxWriteSize      = 10 * 1024 * 1024 // 10 MB
	defaultReadTimeout       = 30 * time.Second
	defaultWriteTimeout      = 120 * time.Second
	defaultIdleTimeout       = 60 * time.Second
	defaultReadHeaderTimeout = 10 * time.Second
)

// Config holds runtime settings for the local FileAPI agent.
type Config struct {
	Host           string   `yaml:"host"`
	Port           int      `yaml:"port"`
	AllowedRoots   []string `yaml:"allowed_roots"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	CertFile       string   `yaml:"cert_file"`
	KeyFile        string   `yaml:"key_file"`

	MaxReadSizeBytes  int64 `yaml:"max_read_size_bytes"`
	MaxWriteSizeBytes int64 `yaml:"max_write_size_bytes"`

	ReadTimeoutSeconds       int `yaml:"read_timeout_seconds"`
	WriteTimeoutSeconds      int `yaml:"write_timeout_seconds"`
	IdleTimeoutSeconds       int `yaml:"idle_timeout_seconds"`
	ReadHeaderTimeoutSeconds int `yaml:"read_header_timeout_seconds"`

	ConfigDir  string `yaml:"-"`
	ConfigPath string `yaml:"-"`
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

		MaxReadSizeBytes:  defaultMaxReadSize,
		MaxWriteSizeBytes: defaultMaxWriteSize,

		ReadTimeoutSeconds:       int(defaultReadTimeout / time.Second),
		WriteTimeoutSeconds:      int(defaultWriteTimeout / time.Second),
		IdleTimeoutSeconds:       int(defaultIdleTimeout / time.Second),
		ReadHeaderTimeoutSeconds: int(defaultReadHeaderTimeout / time.Second),
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
		cfg.applyDefaults()
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	return cfg, nil
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
	if c.MaxReadSizeBytes <= 0 {
		c.MaxReadSizeBytes = defaultMaxReadSize
	}
	if c.MaxWriteSizeBytes <= 0 {
		c.MaxWriteSizeBytes = defaultMaxWriteSize
	}
	if c.ReadTimeoutSeconds <= 0 {
		c.ReadTimeoutSeconds = int(defaultReadTimeout / time.Second)
	}
	if c.WriteTimeoutSeconds <= 0 {
		c.WriteTimeoutSeconds = int(defaultWriteTimeout / time.Second)
	}
	if c.IdleTimeoutSeconds <= 0 {
		c.IdleTimeoutSeconds = int(defaultIdleTimeout / time.Second)
	}
	if c.ReadHeaderTimeoutSeconds <= 0 {
		c.ReadHeaderTimeoutSeconds = int(defaultReadHeaderTimeout / time.Second)
	}
}

// ReadTimeout returns the HTTP read timeout.
func (c *Config) ReadTimeout() time.Duration {
	return time.Duration(c.ReadTimeoutSeconds) * time.Second
}

// WriteTimeout returns the HTTP write timeout.
func (c *Config) WriteTimeout() time.Duration {
	return time.Duration(c.WriteTimeoutSeconds) * time.Second
}

// IdleTimeout returns the HTTP idle timeout.
func (c *Config) IdleTimeout() time.Duration {
	return time.Duration(c.IdleTimeoutSeconds) * time.Second
}

// ReadHeaderTimeout returns the HTTP read-header timeout.
func (c *Config) ReadHeaderTimeout() time.Duration {
	return time.Duration(c.ReadHeaderTimeoutSeconds) * time.Second
}

// Save writes the current configuration to disk.
func (c *Config) Save() error {
	if err := os.MkdirAll(c.ConfigDir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
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
