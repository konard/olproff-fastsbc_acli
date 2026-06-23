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
	"strings"
	"testing"
)

func TestCandidateCommitIsIsolatedAndAudited(t *testing.T) {
	backend := NewMockBackendClient()
	auditor := &MemoryAuditor{}
	engine := NewEngine(EngineConfig{Backend: backend, Auditor: auditor})
	session := engine.NewSession(UserIdentity{Username: "admin", Role: RoleAdmin}, "192.0.2.10")

	if _, err := engine.Execute(context.Background(), session, "configure terminal"); err != nil {
		t.Fatalf("enter configure mode: %v", err)
	}
	if _, err := engine.Execute(context.Background(), session, "system hostname sbc-a"); err != nil {
		t.Fatalf("stage hostname: %v", err)
	}
	result, err := engine.Execute(context.Background(), session, "done")
	if err != nil {
		t.Fatalf("commit candidate: %v", err)
	}
	if !strings.Contains(result.Output, "committed 1 candidate change") {
		t.Fatalf("unexpected commit output: %q", result.Output)
	}

	commits := backend.Commits()
	if len(commits) != 1 {
		t.Fatalf("got %d commits, want 1", len(commits))
	}
	if got := commits[0].CandidateChanges["system.hostname"]; got != "sbc-a" {
		t.Fatalf("candidate hostname = %q, want sbc-a", got)
	}
	if session.Candidate.Len() != 0 {
		t.Fatalf("candidate changes were not cleared after commit")
	}
	if len(auditor.Events) != 2 {
		t.Fatalf("got %d audit events, want 2", len(auditor.Events))
	}
	if auditor.Events[1].ExecutionStatus != "success" || auditor.Events[1].GRPCTraceID == "" {
		t.Fatalf("audit event missing success status or trace ID: %+v", auditor.Events[1])
	}
}

func TestReadOnlyRoleCannotExecuteOrCompleteAdminCommands(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	session := engine.NewSession(UserIdentity{Username: "viewer", Role: RoleReadOnly}, "192.0.2.11")

	if _, err := engine.Execute(context.Background(), session, "configure terminal"); err == nil {
		t.Fatalf("read-only user unexpectedly entered configuration mode")
	}
	completions := engine.Complete(session, "c")
	if len(completions) != 0 {
		t.Fatalf("read-only completions expose admin command: %v", completions)
	}

	completions = engine.Complete(session, "show ")
	if len(completions) != 2 || completions[0] != "configuration" || completions[1] != "health" {
		t.Fatalf("show completions = %v, want [configuration health]", completions)
	}
}

func TestHelpAndPromptFollowCurrentContext(t *testing.T) {
	engine := NewEngine(EngineConfig{PromptHostname: "sbc1"})
	session := engine.NewSession(UserIdentity{Username: "admin", Role: RoleAdmin}, "192.0.2.12")

	result, err := engine.Execute(context.Background(), session, "show ?")
	if err != nil {
		t.Fatalf("show help: %v", err)
	}
	if !strings.Contains(result.Output, "configuration") || !strings.Contains(result.Output, "health") {
		t.Fatalf("help output missing show children: %q", result.Output)
	}

	if _, err := engine.Execute(context.Background(), session, "conf term"); err != nil {
		t.Fatalf("abbreviated configure terminal failed: %v", err)
	}
	if got, want := session.Prompt(), "sbc1(configure)# "; got != want {
		t.Fatalf("prompt = %q, want %q", got, want)
	}
}
