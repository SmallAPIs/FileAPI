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
)

var (
	ErrPathNotAllowed = errors.New("path is outside allowed roots")
	ErrPathTraversal  = errors.New("path traversal is not allowed")
	ErrNotFound       = errors.New("file not found")
	ErrBinaryContent  = errors.New("file appears to be binary or is not valid UTF-8")
	ErrTooLarge       = errors.New("file exceeds maximum read size")
	ErrContentTooLarge = errors.New("content exceeds maximum write size")
)

// FileInfo describes a file returned by read operations.
type FileInfo struct {
	Path       string    `json:"path"`
	Content    string    `json:"content,omitempty"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
}

// Service performs path-validated filesystem operations.
type Service struct {
	allowedRoots  []string
	maxReadSize   int64
	maxWriteSize  int64
}

// NewService creates a filesystem service constrained to allowedRoots.
func NewService(allowedRoots []string, maxReadSize, maxWriteSize int64) (*Service, error) {
	if len(allowedRoots) == 0 {
		return nil, errors.New("at least one allowed root is required")
	}
	if maxReadSize <= 0 {
		maxReadSize = 10 * 1024 * 1024
	}
	if maxWriteSize <= 0 {
		maxWriteSize = maxReadSize
	}

	resolved := make([]string, 0, len(allowedRoots))
	for _, root := range allowedRoots {
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("resolve root %q: %w", root, err)
		}
		resolved = append(resolved, filepath.Clean(abs))
	}

	return &Service{
		allowedRoots: resolved,
		maxReadSize:  maxReadSize,
		maxWriteSize: maxWriteSize,
	}, nil
}

// Read returns UTF-8 text content for a sandboxed path.
func (s *Service) Read(path string) (*FileInfo, error) {
	abs, info, err := s.resolveFile(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > s.maxReadSize {
		return nil, ErrTooLarge
	}

	data, err := readValidatedUTF8(abs, s.maxReadSize)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Path:       abs,
		Content:    string(data),
		Size:       info.Size(),
		ModifiedAt: info.ModTime(),
	}, nil
}

// Stat returns file metadata without reading content.
func (s *Service) Stat(path string) (*FileInfo, error) {
	abs, info, err := s.resolveFile(path)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Path:       abs,
		Size:       info.Size(),
		ModifiedAt: info.ModTime(),
	}, nil
}

// StreamRaw validates UTF-8 and streams file content to w.
func (s *Service) StreamRaw(path string, w io.Writer) (*FileInfo, error) {
	abs, info, err := s.resolveFile(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > s.maxReadSize {
		return nil, ErrTooLarge
	}

	if err := streamValidatedUTF8(abs, s.maxReadSize, w); err != nil {
		return nil, err
	}

	return &FileInfo{
		Path:       abs,
		Size:       info.Size(),
		ModifiedAt: info.ModTime(),
	}, nil
}

// Create writes a new file at path, optionally creating parent directories.
func (s *Service) Create(path, content string, createDirs, includeContent bool) (*FileInfo, error) {
	if int64(len(content)) > s.maxWriteSize {
		return nil, ErrContentTooLarge
	}

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

	if includeContent {
		return s.Read(abs)
	}
	return s.Stat(abs)
}

// Edit updates file content by overwrite or append.
func (s *Service) Edit(path, content, mode string, includeContent bool) (*FileInfo, error) {
	if int64(len(content)) > s.maxWriteSize {
		return nil, ErrContentTooLarge
	}

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

	if includeContent {
		return s.Read(abs)
	}
	return s.Stat(abs)
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

func (s *Service) resolveFile(path string) (string, os.FileInfo, error) {
	abs, err := s.resolve(path)
	if err != nil {
		return "", nil, err
	}

	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, ErrNotFound
		}
		return "", nil, err
	}
	if info.IsDir() {
		return "", nil, fmt.Errorf("path is a directory")
	}

	return abs, info, nil
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
