package config

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// parseConfigYAML unmarshals the small subset of YAML used by FileAPI config files.
// It tolerates inline comments and unknown keys (both are silently ignored).
func parseConfigYAML(data []byte) (*Config, error) {
	cfg := &Config{}
	var listKey string
	listKeyKnown := false

	for lineNo, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, "- ") {
			if listKey == "" {
				return nil, fmt.Errorf("line %d: list item without key", lineNo+1)
			}
			if !listKeyKnown {
				// List items under an unrecognised key are silently skipped.
				continue
			}
			item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			item = stripInlineComment(item)
			item = unquoteYAMLString(item)
			switch listKey {
			case "allowed_roots":
				cfg.AllowedRoots = append(cfg.AllowedRoots, item)
			case "allowed_origins":
				cfg.AllowedOrigins = append(cfg.AllowedOrigins, item)
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
			// Bare key introduces a list.
			switch key {
			case "allowed_roots", "allowed_origins":
				listKey = key
				listKeyKnown = true
			default:
				// Unknown list key: track it so list items can be skipped.
				listKey = key
				listKeyKnown = false
			}
			continue
		}
		listKey = ""
		listKeyKnown = false

		value = stripInlineComment(value)

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
		case "max_read_size_bytes":
			v, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid max_read_size_bytes: %w", lineNo+1, err)
			}
			cfg.MaxReadSizeBytes = v
		case "max_write_size_bytes":
			v, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid max_write_size_bytes: %w", lineNo+1, err)
			}
			cfg.MaxWriteSizeBytes = v
		case "read_timeout_seconds":
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid read_timeout_seconds: %w", lineNo+1, err)
			}
			cfg.ReadTimeoutSeconds = v
		case "write_timeout_seconds":
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid write_timeout_seconds: %w", lineNo+1, err)
			}
			cfg.WriteTimeoutSeconds = v
		case "idle_timeout_seconds":
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid idle_timeout_seconds: %w", lineNo+1, err)
			}
			cfg.IdleTimeoutSeconds = v
		case "read_header_timeout_seconds":
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("line %d: invalid read_header_timeout_seconds: %w", lineNo+1, err)
			}
			cfg.ReadHeaderTimeoutSeconds = v
		default:
			// Unknown key: silently ignore for forward-compatibility.
		}
	}

	return cfg, nil
}

func marshalConfigYAML(cfg *Config) ([]byte, error) {
	var buf bytes.Buffer
	writeScalar := func(key, value string) {
		buf.WriteString(key)
		buf.WriteString(": ")
		buf.WriteString(quoteIfNeeded(value))
		buf.WriteByte('\n')
	}
	writeInt := func(key string, value int64) {
		buf.WriteString(key)
		buf.WriteString(": ")
		buf.WriteString(strconv.FormatInt(value, 10))
		buf.WriteByte('\n')
	}
	writeList := func(key string, items []string) {
		buf.WriteString(key)
		buf.WriteString(":\n")
		for _, item := range items {
			buf.WriteString("  - ")
			buf.WriteString(quoteIfNeeded(item))
			buf.WriteByte('\n')
		}
	}

	writeScalar("host", cfg.Host)
	writeInt("port", int64(cfg.Port))
	buf.WriteByte('\n')
	writeList("allowed_roots", cfg.AllowedRoots)
	buf.WriteByte('\n')
	writeList("allowed_origins", cfg.AllowedOrigins)
	buf.WriteByte('\n')
	writeInt("max_read_size_bytes", cfg.MaxReadSizeBytes)
	writeInt("max_write_size_bytes", cfg.MaxWriteSizeBytes)
	buf.WriteByte('\n')
	writeInt("read_timeout_seconds", int64(cfg.ReadTimeoutSeconds))
	writeInt("write_timeout_seconds", int64(cfg.WriteTimeoutSeconds))
	writeInt("idle_timeout_seconds", int64(cfg.IdleTimeoutSeconds))
	writeInt("read_header_timeout_seconds", int64(cfg.ReadHeaderTimeoutSeconds))
	if cfg.CertFile != "" {
		buf.WriteByte('\n')
		writeScalar("cert_file", cfg.CertFile)
	}
	if cfg.KeyFile != "" {
		writeScalar("key_file", cfg.KeyFile)
	}

	return buf.Bytes(), nil
}

// stripInlineComment removes a trailing YAML inline comment from value,
// respecting quoted strings so that a '#' inside quotes is not stripped.
func stripInlineComment(value string) string {
	if len(value) == 0 {
		return value
	}
	// Quoted string: find the closing quote and return just the quoted portion.
	if value[0] == '"' {
		if end := strings.Index(value[1:], `"`); end >= 0 {
			return value[:end+2]
		}
		return value // unclosed quote; return as-is
	}
	if value[0] == '\'' {
		if end := strings.Index(value[1:], "'"); end >= 0 {
			return value[:end+2]
		}
		return value
	}
	// Unquoted: strip from the first ' #' sequence.
	if idx := strings.Index(value, " #"); idx >= 0 {
		return strings.TrimSpace(value[:idx])
	}
	return value
}

// quoteIfNeeded wraps value in double-quotes when it contains characters that
// have special meaning in YAML (e.g. '*' starts an alias, ' #' starts a comment).
func quoteIfNeeded(s string) string {
	if len(s) == 0 {
		return `""`
	}
	needsQuote := strings.ContainsAny(s[:1], "*&!|>`") ||
		strings.Contains(s, " #")
	if !needsQuote {
		return s
	}
	// Escape backslash and double-quote, then wrap.
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// unquoteYAMLString strips surrounding quotes (single or double) from a YAML
// scalar value.  Basic escape sequences (\\ and \") inside double-quoted
// strings are expanded.
func unquoteYAMLString(value string) string {
	if len(value) < 2 {
		return value
	}
	if value[0] == '"' && value[len(value)-1] == '"' {
		inner := value[1 : len(value)-1]
		inner = strings.ReplaceAll(inner, `\"`, `"`)
		inner = strings.ReplaceAll(inner, `\\`, `\`)
		return inner
	}
	if value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}
	return value
}
