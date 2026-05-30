package handler

import (
	"sync"
	"time"
)

type operationStatus struct {
	Running     bool   `json:"running"`
	Stage       string `json:"stage,omitempty"`
	Progress    int    `json:"progress,omitempty"`
	Total       int    `json:"total,omitempty"`
	Message     string `json:"message,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	CompletedAt string `json:"completed_at,omitempty"`
	Error       string `json:"error,omitempty"`
}

type opTracker struct {
	mu     sync.Mutex
	status *operationStatus
}

func (t *opTracker) update(stage, message string, progress, total int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now().Format(time.RFC3339)
	if t.status == nil {
		t.status = &operationStatus{Running: true, StartedAt: now}
	}
	t.status.Running = true
	t.status.Stage = stage
	t.status.Message = message
	t.status.Progress = progress
	t.status.Total = total
}

func (t *opTracker) error(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.status == nil {
		t.status = &operationStatus{StartedAt: time.Now().Format(time.RFC3339)}
	}
	t.status.Running = false
	t.status.Error = msg
	t.status.CompletedAt = time.Now().Format(time.RFC3339)
}

func (t *opTracker) complete(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.status == nil {
		t.status = &operationStatus{StartedAt: time.Now().Format(time.RFC3339)}
	}
	t.status.Running = false
	t.status.Message = msg
	t.status.CompletedAt = time.Now().Format(time.RFC3339)
}

func (t *opTracker) clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.status = nil
}

func (t *opTracker) get() *operationStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.status
}
