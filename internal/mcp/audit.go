package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type AuditEvent struct {
	At         time.Time `json:"at"`
	Tool       string    `json:"tool"`
	Arguments  any       `json:"arguments,omitempty"`
	DurationMS int64     `json:"duration_ms"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
}

type AuditSink interface {
	Record(context.Context, AuditEvent) error
}

type FileAuditSink struct {
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
}

func NewFileAuditSink(path string) (*FileAuditSink, error) {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil, fmt.Errorf("audit log path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create audit log dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open audit log: %w", err)
	}
	return &FileAuditSink{
		file: file,
		enc:  json.NewEncoder(file),
	}, nil
}

func (s *FileAuditSink) Record(_ context.Context, event AuditEvent) error {
	if s == nil || s.file == nil || s.enc == nil {
		return fmt.Errorf("audit sink is not initialized")
	}
	if event.At.IsZero() {
		event.At = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.enc.Encode(event); err != nil {
		return fmt.Errorf("encode audit event: %w", err)
	}
	if err := s.file.Sync(); err != nil {
		return fmt.Errorf("sync audit log: %w", err)
	}
	return nil
}

func (s *FileAuditSink) Close() error {
	if s == nil || s.file == nil {
		return nil
	}
	return s.file.Close()
}
