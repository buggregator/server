package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// AttachmentStore provides file storage for attachments.
// Supports in-memory (default) and filesystem modes.
type AttachmentStore struct {
	mu    sync.RWMutex
	mode  string            // "memory" or "filesystem"
	dir   string            // filesystem base directory
	files map[string][]byte // in-memory: path → content
}

// NewAttachmentStore creates an attachment store.
// mode: "memory" (default) or "filesystem"
// dir: base directory for filesystem mode (ignored for memory)
func NewAttachmentStore(mode, dir string) *AttachmentStore {
	if mode == "" {
		mode = "memory"
	}
	return &AttachmentStore{
		mode:  mode,
		dir:   dir,
		files: make(map[string][]byte),
	}
}

// Store saves attachment content. path format: "{eventUUID}/{filename}"
func (s *AttachmentStore) Store(path string, content []byte) error {
	if s.mode == "filesystem" {
		fullPath := filepath.Join(s.dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return err
		}
		return os.WriteFile(fullPath, content, 0644)
	}

	// In-memory.
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[path] = content
	return nil
}

// Get retrieves attachment content.
func (s *AttachmentStore) Get(path string) ([]byte, error) {
	if s.mode == "filesystem" {
		return os.ReadFile(filepath.Join(s.dir, path))
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.files[path]
	if !ok {
		return nil, fmt.Errorf("attachment not found: %s", path)
	}
	return data, nil
}

// GetReader returns a reader for the attachment.
func (s *AttachmentStore) GetReader(path string) (io.ReadCloser, error) {
	if s.mode == "filesystem" {
		return os.Open(filepath.Join(s.dir, path))
	}

	data, err := s.Get(path)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(io.NewSectionReader(
		&bytesReaderAt{data}, 0, int64(len(data)),
	)), nil
}

// DeleteByEvent removes all files for an event.
func (s *AttachmentStore) DeleteByEvent(eventUUID string) {
	prefix := eventUUID + "/"

	if s.mode == "filesystem" {
		os.RemoveAll(filepath.Join(s.dir, eventUUID))
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for path := range s.files {
		if len(path) > len(prefix) && path[:len(prefix)] == prefix {
			delete(s.files, path)
		}
	}
}

type bytesReaderAt struct {
	data []byte
}

func (b *bytesReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n = copy(p, b.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return
}
