// Package filesystem performs sandboxed file operations for the local agent.
package filesystem

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const maxReadSize = 10 * 1024 * 1024 // 10 MB

var (
	ErrPathNotAllowed = errors.New("path is outside allowed roots")
	ErrPathTraversal  = errors.New("path traversal is not allowed")
	ErrNotFound       = errors.New("file not found")
	ErrBinaryContent  = errors.New("file appears to be binary or is not valid UTF-8")
	ErrTooLarge       = errors.New("file exceeds maximum read size")
)

// FileInfo describes a file returned by read operations.
type FileInfo struct {
	Path       string    `json:"path"`
	Content    string    `json:"content"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
}

// Service performs path-validated filesystem operations.
type Service struct {
	allowedRoots []string
}

// NewService creates a filesystem service constrained to allowedRoots.
func NewService(allowedRoots []string) (*Service, error) {
	if len(allowedRoots) == 0 {
		return nil, errors.New("at least one allowed root is required")
	}

	resolved := make([]string, 0, len(allowedRoots))
	for _, root := range allowedRoots {
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("resolve root %q: %w", root, err)
		}
		resolved = append(resolved, filepath.Clean(abs))
	}

	return &Service{allowedRoots: resolved}, nil
}

// Read returns UTF-8 text content for a sandboxed path.
func (s *Service) Read(path string) (*FileInfo, error) {
	abs, err := s.resolve(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("path is a directory")
	}
	if info.Size() > maxReadSize {
		return nil, ErrTooLarge
	}

	data, err := os.ReadFile(abs)
	if err != nil {
		return nil, err
	}
	if !utf8.Valid(data) {
		return nil, ErrBinaryContent
	}

	return &FileInfo{
		Path:       abs,
		Content:    string(data),
		Size:       info.Size(),
		ModifiedAt: info.ModTime(),
	}, nil
}

// Create writes a new file at path, optionally creating parent directories.
func (s *Service) Create(path, content string, createDirs bool) (*FileInfo, error) {
	abs, err := s.resolve(path)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(abs); err == nil {
		return nil, fmt.Errorf("file already exists")
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	if createDirs {
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return nil, err
		}
	}

	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return nil, err
	}

	return s.Read(abs)
}

// Edit updates file content by overwrite or append.
func (s *Service) Edit(path, content, mode string) (*FileInfo, error) {
	abs, err := s.resolve(path)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(abs); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	switch mode {
	case "", "overwrite":
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			return nil, err
		}
	case "append":
		f, err := os.OpenFile(abs, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if _, err := io.WriteString(f, content); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported edit mode %q", mode)
	}

	return s.Read(abs)
}

// Delete removes a file at path.
func (s *Service) Delete(path string) error {
	abs, err := s.resolve(path)
	if err != nil {
		return err
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("path is a directory")
	}

	return os.Remove(abs)
}

func (s *Service) resolve(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	if strings.Contains(path, "\x00") {
		return "", fmt.Errorf("invalid path")
	}

	cleaned := filepath.Clean(path)
	if strings.Contains(cleaned, "..") {
		return "", ErrPathTraversal
	}

	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	for _, root := range s.allowedRoots {
		if pathUnderRoot(abs, root) {
			return abs, nil
		}
	}

	return "", ErrPathNotAllowed
}

func pathUnderRoot(path, root string) bool {
	if path == root {
		return true
	}
	sep := string(os.PathSeparator)
	return strings.HasPrefix(path, root+sep)
}
