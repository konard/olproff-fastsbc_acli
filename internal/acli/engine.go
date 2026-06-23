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
	"errors"
	"strings"
	"time"
)

type EngineConfig struct {
	Backend        BackendClient
	Auditor        Auditor
	CommandTimeout time.Duration
	PromptHostname string
}

type Engine struct {
	root           *CommandNode
	configureRoot  *CommandNode
	backend        BackendClient
	auditor        Auditor
	commandTimeout time.Duration
	promptHostname string
}

func NewEngine(cfg EngineConfig) *Engine {
	backend := cfg.Backend
	if backend == nil {
		backend = NewMockBackendClient()
	}
	auditor := cfg.Auditor
	if auditor == nil {
		auditor = NopAuditor{}
	}
	timeout := cfg.CommandTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	hostname := cfg.PromptHostname
	if hostname == "" {
		hostname = "fastsbc"
	}
	return &Engine{
		root:           NewRootCommandTree(),
		configureRoot:  NewConfigureCommandTree(),
		backend:        backend,
		auditor:        auditor,
		commandTimeout: timeout,
		promptHostname: hostname,
	}
}

func (e *Engine) NewSession(identity UserIdentity, clientIP string) *Session {
	if identity.Role == "" {
		identity.Role = RoleReadOnly
	}
	if identity.Username == "" {
		identity.Username = "unknown"
	}
	return &Session{
		Identity:       identity,
		ClientIP:       clientIP,
		Candidate:      NewCandidateConfig(),
		Backend:        e.backend,
		Auditor:        e.auditor,
		PromptHostname: e.promptHostname,
	}
}

func (e *Engine) Execute(ctx context.Context, session *Session, input string) (CommandResult, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return CommandResult{}, nil
	}
	root := e.rootForSession(session)
	if strings.HasSuffix(trimmed, "?") {
		tokens, err := Tokenize(strings.TrimSpace(strings.TrimSuffix(trimmed, "?")))
		if err != nil {
			return CommandResult{}, err
		}
		return CommandResult{Output: HelpFor(root, tokens, session.Identity.Role)}, nil
	}

	tokens, err := Tokenize(trimmed)
	if err != nil {
		return CommandResult{}, err
	}
	resolved, err := ResolveCommand(root, tokens, session.Identity.Role)
	if err != nil {
		return CommandResult{}, err
	}

	callCtx, cancel := context.WithTimeout(ctx, e.commandTimeout)
	defer cancel()

	result, err := resolved.Node.Handler(callCtx, session, input, resolved.Argv)
	if resolved.Node.Mutating {
		e.recordAudit(session, input, result.TraceID, auditStatus(err))
	}
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return CommandResult{TraceID: result.TraceID}, NewBackendUnavailable()
		}
		return CommandResult{TraceID: result.TraceID}, err
	}
	return result, nil
}

func (e *Engine) Complete(session *Session, input string) []string {
	tokens, err := Tokenize(input)
	if err != nil {
		return nil
	}
	if strings.TrimSpace(input) == "" {
		return Complete(e.rootForSession(session), []Token{{Text: "", Start: 0}}, session.Identity.Role)
	}
	if strings.HasSuffix(input, " ") {
		tokens = append(tokens, Token{Text: "", Start: len(input)})
	}
	return Complete(e.rootForSession(session), tokens, session.Identity.Role)
}

func (e *Engine) rootForSession(session *Session) *CommandNode {
	if session.InConfigureMode() {
		return e.configureRoot
	}
	return e.root
}

func (e *Engine) recordAudit(session *Session, rawCommand string, traceID string, status string) {
	e.auditor.Record(AuditEvent{
		Timestamp:       time.Now().UTC(),
		ClientIP:        session.ClientIP,
		Username:        session.Identity.Username,
		Role:            session.Identity.Role,
		RawCommand:      rawCommand,
		ExecutionStatus: status,
		GRPCTraceID:     traceID,
	})
}

func handleConfigureTerminal(_ context.Context, session *Session, _ string, _ []string) (CommandResult, error) {
	session.Path = []string{"configure"}
	return CommandResult{}, nil
}

func handleEnable(_ context.Context, _ *Session, _ string, _ []string) (CommandResult, error) {
	return CommandResult{Output: "already in privileged mode\n"}, nil
}

func handleExit(_ context.Context, session *Session, _ string, _ []string) (CommandResult, error) {
	if session.InConfigureMode() {
		session.Path = nil
		return CommandResult{}, nil
	}
	return CommandResult{Exit: true}, nil
}

func handleCancel(_ context.Context, session *Session, _ string, _ []string) (CommandResult, error) {
	session.Candidate.Reset()
	session.Path = nil
	return CommandResult{Output: "candidate configuration discarded\n", TraceID: newTraceID()}, nil
}

func handleSetHostname(_ context.Context, session *Session, _ string, argv []string) (CommandResult, error) {
	session.Candidate.Set("system.hostname", argv[0])
	return CommandResult{Output: "staged system hostname\n", TraceID: newTraceID()}, nil
}

func handleSetInterfaceIPAddress(_ context.Context, session *Session, _ string, argv []string) (CommandResult, error) {
	session.Candidate.Set("network_interface.ip_address", argv[0])
	return CommandResult{Output: "staged network-interface ip-address\n", TraceID: newTraceID()}, nil
}

func handleDone(ctx context.Context, session *Session, input string, _ []string) (CommandResult, error) {
	traceID := newTraceID()
	changes := session.Candidate.Snapshot()
	if len(changes) == 0 {
		return CommandResult{Output: "no candidate changes to commit\n", TraceID: traceID}, nil
	}
	response, err := session.Backend.CommitCandidate(ctx, BackendRequest{
		TraceID:          traceID,
		Username:         session.Identity.Username,
		Role:             session.Identity.Role,
		RawCommand:       input,
		CandidateChanges: changes,
	})
	if err != nil {
		return CommandResult{TraceID: traceID}, err
	}
	session.Candidate.Reset()
	return CommandResult{Output: response.Output, TraceID: response.TraceID}, nil
}

func handleShowConfiguration(ctx context.Context, session *Session, input string, _ []string) (CommandResult, error) {
	traceID := newTraceID()
	response, err := session.Backend.ShowState(ctx, BackendRequest{
		TraceID:          traceID,
		Username:         session.Identity.Username,
		Role:             session.Identity.Role,
		RawCommand:       input,
		Argv:             []string{"configuration"},
		CandidateChanges: session.Candidate.Snapshot(),
	})
	if err != nil {
		return CommandResult{TraceID: traceID}, err
	}
	if session.Candidate.Len() == 0 {
		return CommandResult{Output: response.Output, TraceID: response.TraceID}, nil
	}
	var builder strings.Builder
	builder.WriteString(response.Output)
	builder.WriteString("candidate changes:\n")
	for key, value := range session.Candidate.Snapshot() {
		builder.WriteString("  ")
		builder.WriteString(key)
		builder.WriteString(" = ")
		builder.WriteString(value)
		builder.WriteString("\n")
	}
	return CommandResult{Output: builder.String(), TraceID: response.TraceID}, nil
}

func handleShowHealth(ctx context.Context, session *Session, input string, _ []string) (CommandResult, error) {
	traceID := newTraceID()
	response, err := session.Backend.ShowState(ctx, BackendRequest{
		TraceID:    traceID,
		Username:   session.Identity.Username,
		Role:       session.Identity.Role,
		RawCommand: input,
		Argv:       []string{"health"},
	})
	if err != nil {
		return CommandResult{TraceID: traceID}, err
	}
	return CommandResult{Output: response.Output, TraceID: response.TraceID}, nil
}
