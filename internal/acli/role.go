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

import "strings"

type Role string

const (
	RoleReadOnly Role = "read_only"
	RoleOperator Role = "operator"
	RoleAdmin    Role = "admin"
)

func ParseRole(value string) Role {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(RoleReadOnly), "readonly", "read-only":
		return RoleReadOnly
	case string(RoleOperator):
		return RoleOperator
	case string(RoleAdmin):
		return RoleAdmin
	default:
		return ""
	}
}

func (r Role) Allows(required Role) bool {
	return roleRank(r) >= roleRank(required)
}

func roleRank(role Role) int {
	switch role {
	case RoleAdmin:
		return 30
	case RoleOperator:
		return 20
	case RoleReadOnly:
		return 10
	default:
		return 0
	}
}
