package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type FileBackend struct {
	baseDir string
	mu      sync.RWMutex
}

func NewFileBackend(baseDir string) (*FileBackend, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	return &FileBackend{baseDir: baseDir}, nil
}

// getFilePath builds a safe file path from the key.
// Protects against path traversal: keys with '..' or absolute paths are rejected.
func (f *FileBackend) getFilePath(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("storage: empty key")
	}
	if filepath.IsAbs(key) {
		return "", fmt.Errorf("storage: absolute key not allowed: %q", key)
	}
	clean := filepath.Clean(key)
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("storage: key escapes base directory: %q", key)
	}
	return filepath.Join(f.baseDir, clean+".json"), nil
}

func (f *FileBackend) Get(ctx context.Context, key string) ([]byte, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	path, err := f.getFilePath(key)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (f *FileBackend) Set(ctx context.Context, key string, value []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	path, err := f.getFilePath(key)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path, value, 0644)
}

func (f *FileBackend) Delete(ctx context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	path, err := f.getFilePath(key)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}