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
	"fmt"
	"sync"
)

type BackendRequest struct {
	TraceID          string
	Username         string
	Role             Role
	RawCommand       string
	Argv             []string
	CandidateChanges map[string]string
}

type BackendResponse struct {
	TraceID string
	Output  string
}

type BackendClient interface {
	ExecuteCommand(ctx context.Context, request BackendRequest) (BackendResponse, error)
	CommitCandidate(ctx context.Context, request BackendRequest) (BackendResponse, error)
	ShowState(ctx context.Context, request BackendRequest) (BackendResponse, error)
}

type MockBackendClient struct {
	mu       sync.Mutex
	shows    map[string]string
	commits  []BackendRequest
	commands []BackendRequest
	err      error
}

func NewMockBackendClient() *MockBackendClient {
	return &MockBackendClient{
		shows: map[string]string{
			"configuration": "running configuration is available from the backend\n",
			"health":        "backend health: ok\n",
		},
	}
}

func (b *MockBackendClient) SetError(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.err = err
}

func (b *MockBackendClient) Commits() []BackendRequest {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]BackendRequest(nil), b.commits...)
}

func (b *MockBackendClient) Commands() []BackendRequest {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]BackendRequest(nil), b.commands...)
}

func (b *MockBackendClient) ExecuteCommand(_ context.Context, request BackendRequest) (BackendResponse, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return BackendResponse{TraceID: request.TraceID}, b.err
	}
	b.commands = append(b.commands, cloneRequest(request))
	return BackendResponse{TraceID: request.TraceID, Output: "command accepted\n"}, nil
}

func (b *MockBackendClient) CommitCandidate(_ context.Context, request BackendRequest) (BackendResponse, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return BackendResponse{TraceID: request.TraceID}, b.err
	}
	b.commits = append(b.commits, cloneRequest(request))
	return BackendResponse{
		TraceID: request.TraceID,
		Output:  fmt.Sprintf("committed %d candidate change(s)\n", len(request.CandidateChanges)),
	}, nil
}

func (b *MockBackendClient) ShowState(_ context.Context, request BackendRequest) (BackendResponse, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.err != nil {
		return BackendResponse{TraceID: request.TraceID}, b.err
	}
	topic := "health"
	if len(request.Argv) > 0 {
		topic = request.Argv[0]
	}
	output, ok := b.shows[topic]
	if !ok {
		output = fmt.Sprintf("no mock output configured for %s\n", topic)
	}
	return BackendResponse{TraceID: request.TraceID, Output: output}, nil
}

func cloneRequest(request BackendRequest) BackendRequest {
	copied := request
	copied.Argv = append([]string(nil), request.Argv...)
	copied.CandidateChanges = copyStringMap(request.CandidateChanges)
	return copied
}
