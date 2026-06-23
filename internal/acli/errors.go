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
	"fmt"
	"strings"
)

type CLIError struct {
	Message string
	Input   string
	Column  int
}

func (e *CLIError) Error() string {
	return e.Message
}

func (e *CLIError) TerminalString() string {
	if e.Input == "" {
		return e.Message + "\n"
	}
	column := e.Column
	if column < 0 {
		column = 0
	}
	return fmt.Sprintf("%s\n%s^\n%s\n", e.Input, strings.Repeat(" ", column), e.Message)
}

func NewIncompleteCommand() *CLIError {
	return &CLIError{Message: "% Incomplete command"}
}

func NewAmbiguousCommand() *CLIError {
	return &CLIError{Message: "% Ambiguous command"}
}

func NewInvalidInput(input string, column int) *CLIError {
	return &CLIError{
		Message: "% Invalid input detected at '^' marker.",
		Input:   input,
		Column:  column,
	}
}

func NewBackendUnavailable() *CLIError {
	return &CLIError{Message: "% Backend Unavailable"}
}

func FormatTerminalError(err error) string {
	if err == nil {
		return ""
	}
	if cliErr, ok := err.(*CLIError); ok {
		return cliErr.TerminalString()
	}
	return "% " + err.Error() + "\n"
}
