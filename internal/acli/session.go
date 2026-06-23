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
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

type UserIdentity struct {
	Username string
	Role     Role
}

type Session struct {
	Identity       UserIdentity
	ClientIP       string
	Path           []string
	Candidate      *CandidateConfig
	Backend        BackendClient
	Auditor        Auditor
	PromptHostname string
}

func (s *Session) InConfigureMode() bool {
	return len(s.Path) > 0 && s.Path[0] == "configure"
}

func (s *Session) Prompt() string {
	host := s.PromptHostname
	if host == "" {
		host = "fastsbc"
	}
	if s.InConfigureMode() {
		return fmt.Sprintf("%s(%s)# ", host, strings.Join(s.Path, "/"))
	}
	return host + "# "
}

func RunTextSession(ctx context.Context, reader io.Reader, writer io.Writer, engine *Engine, session *Session) error {
	scanner := bufio.NewScanner(reader)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if _, err := io.WriteString(writer, session.Prompt()); err != nil {
			return err
		}
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return err
			}
			return io.EOF
		}
		result, err := engine.Execute(ctx, session, scanner.Text())
		if err != nil {
			if _, writeErr := io.WriteString(writer, FormatTerminalError(err)); writeErr != nil {
				return writeErr
			}
			continue
		}
		if result.Output != "" {
			if _, err := io.WriteString(writer, result.Output); err != nil {
				return err
			}
		}
		if result.Exit {
			return nil
		}
	}
}
