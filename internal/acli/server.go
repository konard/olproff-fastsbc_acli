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
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type SSHServerConfig struct {
	Address        string
	Signer         ssh.Signer
	Authenticator  *StaticAuthenticator
	Engine         *Engine
	Backend        BackendClient
	Auditor        Auditor
	CommandTimeout time.Duration
	PromptHostname string
}

type SSHServer struct {
	cfg      SSHServerConfig
	listener net.Listener
	done     chan struct{}
	once     sync.Once
}

func NewSSHServer(cfg SSHServerConfig) *SSHServer {
	return &SSHServer{cfg: cfg, done: make(chan struct{})}
}

func LoadOrGenerateHostSigner(path string) (ssh.Signer, error) {
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		return ssh.ParsePrivateKey(data)
	}
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(privateKey)
}

func (s *SSHServer) ListenAndServe() error {
	if s.cfg.Address == "" {
		s.cfg.Address = ":22"
	}
	if s.cfg.Signer == nil {
		return errors.New("ssh host signer is required")
	}
	if s.cfg.Authenticator == nil {
		return errors.New("ssh authenticator is required")
	}
	if s.cfg.Engine == nil {
		return errors.New("ACLI engine is required")
	}

	serverConfig := &ssh.ServerConfig{
		PasswordCallback:  s.cfg.Authenticator.PasswordCallback,
		PublicKeyCallback: s.cfg.Authenticator.PublicKeyCallback,
		ServerVersion:     "SSH-2.0-fastsbc_acli",
	}
	serverConfig.AddHostKey(s.cfg.Signer)

	listener, err := net.Listen("tcp", s.cfg.Address)
	if err != nil {
		return err
	}
	s.listener = listener
	defer close(s.done)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return context.Canceled
			}
			return err
		}
		go s.handleConn(conn, serverConfig)
	}
}

func (s *SSHServer) Shutdown(ctx context.Context) error {
	var err error
	s.once.Do(func() {
		if s.listener != nil {
			err = s.listener.Close()
		}
		select {
		case <-s.done:
		case <-ctx.Done():
			err = ctx.Err()
		}
	})
	return err
}

func (s *SSHServer) handleConn(conn net.Conn, serverConfig *ssh.ServerConfig) {
	sshConn, channels, requests, err := ssh.NewServerConn(conn, serverConfig)
	if err != nil {
		slog.Warn("ssh handshake failed", "remote", conn.RemoteAddr(), "error", err)
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(requests)

	identity := identityFromPermissions(sshConn.Permissions, sshConn.User())
	clientIP, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	if clientIP == "" {
		clientIP = conn.RemoteAddr().String()
	}
	for newChannel := range channels {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "session channels only")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}
		go s.handleSession(channel, requests, identity, clientIP)
	}
}

func (s *SSHServer) handleSession(channel ssh.Channel, requests <-chan *ssh.Request, identity UserIdentity, clientIP string) {
	defer channel.Close()
	session := s.cfg.Engine.NewSession(identity, clientIP)
	acceptedShell := make(chan bool, 1)
	go func() {
		sent := false
		for request := range requests {
			switch request.Type {
			case "pty-req":
				request.Reply(true, nil)
			case "shell":
				request.Reply(true, nil)
				if !sent {
					acceptedShell <- true
					sent = true
				}
			case "exec":
				request.Reply(false, nil)
			default:
				request.Reply(false, nil)
			}
		}
		if !sent {
			acceptedShell <- false
		}
	}()
	if ok := <-acceptedShell; !ok {
		return
	}
	if err := RunTextSession(context.Background(), channel, channel, s.cfg.Engine, session); err != nil && !errors.Is(err, io.EOF) {
		fmt.Fprintf(channel.Stderr(), "%s", FormatTerminalError(err))
	}
}

func identityFromPermissions(permissions *ssh.Permissions, fallbackUser string) UserIdentity {
	if permissions == nil {
		return UserIdentity{Username: fallbackUser, Role: RoleReadOnly}
	}
	role := ParseRole(permissions.Extensions["role"])
	if role == "" {
		role = RoleReadOnly
	}
	username := permissions.Extensions["username"]
	if username == "" {
		username = fallbackUser
	}
	return UserIdentity{Username: username, Role: role}
}
