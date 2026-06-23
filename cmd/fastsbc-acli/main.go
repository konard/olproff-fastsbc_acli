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

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/olproff/fastsbc_acli/internal/acli"
)

func main() {
	cfg, err := acli.LoadServiceConfigFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %v\n", err)
		os.Exit(2)
	}

	auditor, err := acli.NewFileAuditLogger(cfg.AuditLogPath, cfg.AuditQueueSize)
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit logger error: %v\n", err)
		os.Exit(2)
	}
	defer auditor.Close(context.Background())

	signer, err := acli.LoadOrGenerateHostSigner(cfg.HostKeyPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "host key error: %v\n", err)
		os.Exit(2)
	}

	backend := acli.NewMockBackendClient()
	engine := acli.NewEngine(acli.EngineConfig{
		Backend:        backend,
		Auditor:        auditor,
		CommandTimeout: cfg.BackendTimeout,
		PromptHostname: cfg.PromptHostname,
	})

	server := acli.NewSSHServer(acli.SSHServerConfig{
		Address:        cfg.ListenAddress,
		Signer:         signer,
		Authenticator:  acli.NewStaticAuthenticator(cfg.Users),
		Engine:         engine,
		Backend:        backend,
		Auditor:        auditor,
		CommandTimeout: cfg.BackendTimeout,
		PromptHostname: cfg.PromptHostname,
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("starting fastsbc_acli ssh listener", "address", cfg.ListenAddress)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutting down fastsbc_acli")
	case err := <-errCh:
		if err != nil && !errors.Is(err, context.Canceled) {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if err := server.Shutdown(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "shutdown error: %v\n", err)
		os.Exit(1)
	}
}
