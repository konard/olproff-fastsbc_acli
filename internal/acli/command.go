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
	"fmt"
	"net"
	"sort"
	"strings"
)

type CommandResult struct {
	Output  string
	TraceID string
	Exit    bool
}

type CommandHandler func(ctx context.Context, session *Session, input string, argv []string) (CommandResult, error)

type ArgSpec struct {
	Name     string
	Help     string
	Validate func(string) error
}

type CommandNode struct {
	Name     string
	Help     string
	MinRole  Role
	Mutating bool
	Args     []ArgSpec
	Children []*CommandNode
	Handler  CommandHandler
}

type ResolvedCommand struct {
	Node *CommandNode
	Argv []string
}

func NewRootCommandTree() *CommandNode {
	return &CommandNode{
		Name:    "",
		MinRole: RoleReadOnly,
		Children: []*CommandNode{
			{
				Name:    "show",
				Help:    "display operational state",
				MinRole: RoleReadOnly,
				Children: []*CommandNode{
					{Name: "configuration", Help: "display running configuration", MinRole: RoleReadOnly, Handler: handleShowConfiguration},
					{Name: "health", Help: "display backend health", MinRole: RoleReadOnly, Handler: handleShowHealth},
				},
			},
			{
				Name:    "configure",
				Help:    "enter configuration mode",
				MinRole: RoleAdmin,
				Children: []*CommandNode{
					{Name: "terminal", Help: "configure from the terminal", MinRole: RoleAdmin, Handler: handleConfigureTerminal},
				},
			},
			{Name: "enable", Help: "enter privileged mode", MinRole: RoleAdmin, Handler: handleEnable},
			{Name: "exit", Help: "close the ACLI session", MinRole: RoleReadOnly, Handler: handleExit},
		},
	}
}

func NewConfigureCommandTree() *CommandNode {
	return &CommandNode{
		Name:    "configure",
		MinRole: RoleAdmin,
		Children: []*CommandNode{
			{
				Name:    "system",
				Help:    "configure system attributes",
				MinRole: RoleAdmin,
				Children: []*CommandNode{
					{
						Name:     "hostname",
						Help:     "set the system hostname",
						MinRole:  RoleAdmin,
						Mutating: true,
						Args: []ArgSpec{
							{Name: "name", Help: "hostname value", Validate: validateName},
						},
						Handler: handleSetHostname,
					},
				},
			},
			{
				Name:    "network-interface",
				Help:    "configure a network interface",
				MinRole: RoleAdmin,
				Children: []*CommandNode{
					{
						Name:    "ip-address",
						Help:    "stage an interface IP address",
						MinRole: RoleAdmin,
						Args: []ArgSpec{
							{Name: "address", Help: "IPv4 or IPv6 address", Validate: validateIP},
						},
						Mutating: true,
						Handler:  handleSetInterfaceIPAddress,
					},
				},
			},
			{Name: "done", Help: "commit the candidate configuration", MinRole: RoleAdmin, Mutating: true, Handler: handleDone},
			{Name: "commit", Help: "commit the candidate configuration", MinRole: RoleAdmin, Mutating: true, Handler: handleDone},
			{Name: "cancel", Help: "discard candidate changes", MinRole: RoleAdmin, Mutating: true, Handler: handleCancel},
			{Name: "exit", Help: "return to the parent mode", MinRole: RoleReadOnly, Handler: handleExit},
			{
				Name:    "show",
				Help:    "display state while in configuration mode",
				MinRole: RoleReadOnly,
				Children: []*CommandNode{
					{Name: "configuration", Help: "display candidate and running configuration", MinRole: RoleReadOnly, Handler: handleShowConfiguration},
					{Name: "health", Help: "display backend health", MinRole: RoleReadOnly, Handler: handleShowHealth},
				},
			},
		},
	}
}

func ResolveCommand(root *CommandNode, tokens []Token, role Role) (ResolvedCommand, error) {
	if len(tokens) == 0 {
		return ResolvedCommand{}, NewIncompleteCommand()
	}
	node := root
	argv := make([]string, 0)
	for index, token := range tokens {
		matches := node.matchingChildren(token.Text, role)
		switch len(matches) {
		case 0:
			if len(node.Args) == 0 {
				return ResolvedCommand{}, NewInvalidInput(tokensToInput(tokens), token.Start)
			}
			remaining := tokens[index:]
			if len(remaining) != len(node.Args) {
				return ResolvedCommand{}, NewIncompleteCommand()
			}
			for argIndex, argToken := range remaining {
				spec := node.Args[argIndex]
				if spec.Validate != nil {
					if err := spec.Validate(argToken.Text); err != nil {
						return ResolvedCommand{}, NewInvalidInput(tokensToInput(tokens), argToken.Start)
					}
				}
				argv = append(argv, argToken.Text)
			}
			return completeResolution(node, argv)
		case 1:
			node = matches[0]
		default:
			return ResolvedCommand{}, NewAmbiguousCommand()
		}
	}
	return completeResolution(node, argv)
}

func HelpFor(root *CommandNode, tokens []Token, role Role) string {
	node := root
	for _, token := range tokens {
		if token.Text == "" {
			continue
		}
		matches := node.matchingChildren(token.Text, role)
		if len(matches) != 1 {
			return formatHelp(node.visibleChildren(role))
		}
		node = matches[0]
	}
	if len(node.Children) > 0 {
		return formatHelp(node.visibleChildren(role))
	}
	if len(node.Args) > 0 {
		var lines []string
		for _, arg := range node.Args {
			lines = append(lines, fmt.Sprintf("  <%s>\t%s", arg.Name, arg.Help))
		}
		return strings.Join(lines, "\n") + "\n"
	}
	return node.Help + "\n"
}

func Complete(root *CommandNode, tokens []Token, role Role) []string {
	node := root
	for index, token := range tokens {
		last := index == len(tokens)-1
		matches := node.matchingChildren(token.Text, role)
		if last {
			out := make([]string, 0, len(matches))
			for _, match := range matches {
				out = append(out, match.Name)
			}
			sort.Strings(out)
			return out
		}
		if len(matches) != 1 {
			return nil
		}
		node = matches[0]
	}
	return nil
}

func completeResolution(node *CommandNode, argv []string) (ResolvedCommand, error) {
	if node.Handler == nil {
		return ResolvedCommand{}, NewIncompleteCommand()
	}
	if len(argv) != len(node.Args) {
		return ResolvedCommand{}, NewIncompleteCommand()
	}
	return ResolvedCommand{Node: node, Argv: argv}, nil
}

func (n *CommandNode) matchingChildren(prefix string, role Role) []*CommandNode {
	matches := make([]*CommandNode, 0)
	for _, child := range n.Children {
		if !role.Allows(child.MinRole) {
			continue
		}
		if strings.HasPrefix(child.Name, prefix) {
			matches = append(matches, child)
		}
	}
	return matches
}

func (n *CommandNode) visibleChildren(role Role) []*CommandNode {
	out := make([]*CommandNode, 0, len(n.Children))
	for _, child := range n.Children {
		if role.Allows(child.MinRole) {
			out = append(out, child)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func formatHelp(nodes []*CommandNode) string {
	if len(nodes) == 0 {
		return ""
	}
	lines := make([]string, 0, len(nodes))
	for _, node := range nodes {
		lines = append(lines, fmt.Sprintf("  %-20s %s", node.Name, node.Help))
	}
	return strings.Join(lines, "\n") + "\n"
}

func validateName(value string) error {
	if value == "" {
		return errors.New("empty value")
	}
	for _, r := range value {
		if r == '-' || r == '_' || r == '.' {
			continue
		}
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		return fmt.Errorf("invalid character %q", r)
	}
	return nil
}

func validateIP(value string) error {
	if net.ParseIP(value) == nil {
		return fmt.Errorf("invalid IP address %q", value)
	}
	return nil
}
