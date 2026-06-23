// fastsbc_cli - Administration Command Line Interface (ACLI) Service for SBC.
// Copyright (C) 2026 fastsbc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package acli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// AuditEvent contains the mandatory state-changing command audit fields.
type AuditEvent struct {
	Timestamp       time.Time `json:"timestamp"`
	ClientIP        string    `json:"client_ip"`
	Username        string    `json:"username"`
	Role            Role      `json:"role"`
	RawCommand      string    `json:"raw_command"`
	ExecutionStatus string    `json:"execution_status"`
	GRPCTraceID     string    `json:"grpc_trace_id"`
}

type Auditor interface {
	Record(event AuditEvent)
	Close(ctx context.Context) error
}

type NopAuditor struct{}

func (NopAuditor) Record(AuditEvent) {}

func (NopAuditor) Close(context.Context) error {
	return nil
}

type MemoryAuditor struct {
	mu     sync.Mutex
	Events []AuditEvent
}

func (a *MemoryAuditor) Record(event AuditEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Events = append(a.Events, event)
}

func (a *MemoryAuditor) Close(context.Context) error {
	return nil
}

type FileAuditLogger struct {
	events  chan AuditEvent
	done    chan struct{}
	file    *os.File
	dropped atomic.Uint64
	once    sync.Once
}

func NewFileAuditLogger(path string, queueSize int) (*FileAuditLogger, error) {
	if path == "" {
		path = "audit/fastsbc_acli.jsonl"
	}
	if queueSize <= 0 {
		queueSize = 1024
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, err
	}

	logger := &FileAuditLogger{
		events: make(chan AuditEvent, queueSize),
		done:   make(chan struct{}),
		file:   file,
	}
	go logger.writeLoop()
	return logger, nil
}

func (l *FileAuditLogger) Record(event AuditEvent) {
	select {
	case l.events <- event:
	default:
		l.dropped.Add(1)
	}
}

func (l *FileAuditLogger) Dropped() uint64 {
	return l.dropped.Load()
}

func (l *FileAuditLogger) Close(ctx context.Context) error {
	var err error
	l.once.Do(func() {
		close(l.events)
		select {
		case <-l.done:
		case <-ctx.Done():
			err = ctx.Err()
			return
		}
		err = l.file.Close()
	})
	return err
}

func (l *FileAuditLogger) writeLoop() {
	encoder := json.NewEncoder(l.file)
	for event := range l.events {
		if err := encoder.Encode(event); err != nil {
			l.dropped.Add(1)
		}
	}
	close(l.done)
}

func auditStatus(err error) string {
	if err == nil {
		return "success"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline_exceeded"
	}
	return "fail"
}
