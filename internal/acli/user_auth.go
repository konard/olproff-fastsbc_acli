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
	"fmt"

	"golang.org/x/crypto/ssh"
)

type UserCredential struct {
	Username       string
	Password       string
	Role           Role
	AuthorizedKeys []ssh.PublicKey
}

type StaticAuthenticator struct {
	users map[string]UserCredential
}

func NewStaticAuthenticator(users []UserCredential) *StaticAuthenticator {
	byName := make(map[string]UserCredential, len(users))
	for _, user := range users {
		byName[user.Username] = user
	}
	return &StaticAuthenticator{users: byName}
}

func (a *StaticAuthenticator) PasswordCallback(meta ssh.ConnMetadata, password []byte) (*ssh.Permissions, error) {
	user, ok := a.users[meta.User()]
	if !ok || user.Password == "" || user.Password != string(password) {
		return nil, fmt.Errorf("password authentication failed")
	}
	return userPermissions(user), nil
}

func (a *StaticAuthenticator) PublicKeyCallback(meta ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	user, ok := a.users[meta.User()]
	if !ok {
		return nil, fmt.Errorf("public key authentication failed")
	}
	for _, authorizedKey := range user.AuthorizedKeys {
		if bytes.Equal(authorizedKey.Marshal(), key.Marshal()) {
			return userPermissions(user), nil
		}
	}
	return nil, fmt.Errorf("public key authentication failed")
}

func userPermissions(user UserCredential) *ssh.Permissions {
	return &ssh.Permissions{
		Extensions: map[string]string{
			"username": user.Username,
			"role":     string(user.Role),
		},
	}
}
