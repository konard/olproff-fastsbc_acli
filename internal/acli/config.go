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
	"os"
	"strconv"
	"strings"
	"time"
)

type ServiceConfig struct {
	ListenAddress  string
	BackendAddress string
	BackendTimeout time.Duration
	AuditLogPath   string
	AuditQueueSize int
	HostKeyPath    string
	PromptHostname string
	Users          []UserCredential
}

func LoadServiceConfigFromEnv() (ServiceConfig, error) {
	cfg := ServiceConfig{
		ListenAddress:  envOrDefault("FASTSBC_ACLI_LISTEN", ":22"),
		BackendAddress: envOrDefault("FASTSBC_ACLI_BACKEND", "127.0.0.1:50051"),
		AuditLogPath:   envOrDefault("FASTSBC_ACLI_AUDIT_LOG", "audit/fastsbc_acli.jsonl"),
		HostKeyPath:    os.Getenv("FASTSBC_ACLI_HOST_KEY"),
		PromptHostname: envOrDefault("FASTSBC_ACLI_PROMPT_HOSTNAME", "fastsbc"),
	}

	timeoutText := envOrDefault("FASTSBC_ACLI_BACKEND_TIMEOUT", "5s")
	timeout, err := time.ParseDuration(timeoutText)
	if err != nil {
		return ServiceConfig{}, fmt.Errorf("FASTSBC_ACLI_BACKEND_TIMEOUT: %w", err)
	}
	cfg.BackendTimeout = timeout

	queueText := envOrDefault("FASTSBC_ACLI_AUDIT_QUEUE_SIZE", "1024")
	queueSize, err := strconv.Atoi(queueText)
	if err != nil || queueSize <= 0 {
		return ServiceConfig{}, fmt.Errorf("FASTSBC_ACLI_AUDIT_QUEUE_SIZE must be a positive integer")
	}
	cfg.AuditQueueSize = queueSize

	users, err := parseUsers(os.Getenv("FASTSBC_ACLI_USERS"))
	if err != nil {
		return ServiceConfig{}, err
	}
	cfg.Users = users
	return cfg, nil
}

func envOrDefault(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func parseUsers(value string) ([]UserCredential, error) {
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("FASTSBC_ACLI_USERS must define at least one user as username:password:role")
	}
	parts := strings.Split(value, ",")
	users := make([]UserCredential, 0, len(parts))
	for _, part := range parts {
		fields := strings.Split(part, ":")
		if len(fields) != 3 {
			return nil, fmt.Errorf("invalid FASTSBC_ACLI_USERS entry %q", part)
		}
		role := ParseRole(fields[2])
		if role == "" {
			return nil, fmt.Errorf("invalid role %q for user %q", fields[2], fields[0])
		}
		users = append(users, UserCredential{
			Username: fields[0],
			Password: fields[1],
			Role:     role,
		})
	}
	return users, nil
}
