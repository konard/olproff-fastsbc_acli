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
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunTextSessionFormatsInvalidInput(t *testing.T) {
	engine := NewEngine(EngineConfig{})
	session := engine.NewSession(UserIdentity{Username: "viewer", Role: RoleReadOnly}, "192.0.2.20")
	input := strings.NewReader("unknown\nexit\n")
	var output bytes.Buffer

	if err := RunTextSession(context.Background(), input, &output, engine, session); err != nil {
		t.Fatalf("RunTextSession returned error: %v", err)
	}
	if !strings.Contains(output.String(), "% Invalid input detected at '^' marker.") {
		t.Fatalf("invalid input was not formatted for terminal output:\n%s", output.String())
	}
}
