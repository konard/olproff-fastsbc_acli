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

import "sync"

type CandidateConfig struct {
	mu      sync.RWMutex
	changes map[string]string
}

func NewCandidateConfig() *CandidateConfig {
	return &CandidateConfig{changes: make(map[string]string)}
}

func (c *CandidateConfig) Set(path string, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changes[path] = value
}

func (c *CandidateConfig) Snapshot() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return copyStringMap(c.changes)
}

func (c *CandidateConfig) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.changes = make(map[string]string)
}

func (c *CandidateConfig) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.changes)
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
