package config

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// parseConfigYAML unmarshals the small subset of YAML used by FileAPI config files.
func parseConfigYAML(data []byte) (*Config, error) {
	cfg := &Config{}
	var listKey string

	for lineNo, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "- ") {
			if listKey == "" {
				return nil, fmt.Errorf("line %d: list item without key", lineNo+1)
			}
			item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			item = unquoteYAMLString(item)
			switch listKey {
			case "allowed_roots":
				cfg.AllowedRoots = append(cfg.AllowedRoots, item)
			case "allowed_origins":
				cfg.AllowedOrigins = append(cfg.AllowedOrigins, item)
			default:
				return nil, fmt.Errorf("line %d: unexpected list for %q", lineNo+1, listKey)
			}
			continue
		}

		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("line %d: invalid syntax", lineNo+1)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if value == "" {
			listKey = key
			continue
		}
		listKey = ""

		switch key {
		case "host":
			cfg.Host = unquoteYAMLString(value)
		case "port":
			port, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid port: %w", lineNo+1, err)
			}
			cfg.Port = port
		case "cert_file":
			cfg.CertFile = unquoteYAMLString(value)
		case "key_file":
			cfg.KeyFile = unquoteYAMLString(value)
		default:
			return nil, fmt.Errorf("line %d: unknown key %q", lineNo+1, key)
		}
	}

	return cfg, nil
}

func marshalConfigYAML(cfg *Config) ([]byte, error) {
	var buf bytes.Buffer
	writeScalar := func(key, value string) {
		buf.WriteString(key)
		buf.WriteString(": ")
		buf.WriteString(value)
		buf.WriteByte('\n')
	}
	writeList := func(key string, items []string) {
		buf.WriteString(key)
		buf.WriteString(":\n")
		for _, item := range items {
			buf.WriteString("  - ")
			buf.WriteString(item)
			buf.WriteByte('\n')
		}
	}

	writeScalar("host", cfg.Host)
	writeScalar("port", strconv.Itoa(cfg.Port))
	buf.WriteByte('\n')
	writeList("allowed_roots", cfg.AllowedRoots)
	buf.WriteByte('\n')
	writeList("allowed_origins", cfg.AllowedOrigins)
	buf.WriteByte('\n')
	if cfg.CertFile != "" {
		writeScalar("cert_file", cfg.CertFile)
	}
	if cfg.KeyFile != "" {
		writeScalar("key_file", cfg.KeyFile)
	}

	return buf.Bytes(), nil
}

func unquoteYAMLString(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
