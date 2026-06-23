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
	"strings"
	"unicode"
)

type Token struct {
	Text  string
	Start int
}

func Tokenize(input string) ([]Token, error) {
	tokens := make([]Token, 0)
	var builder strings.Builder
	start := -1
	inQuote := false
	escaped := false

	flush := func() {
		if start < 0 {
			return
		}
		tokens = append(tokens, Token{Text: builder.String(), Start: start})
		builder.Reset()
		start = -1
	}

	for index, r := range input {
		if escaped {
			builder.WriteRune(r)
			escaped = false
			continue
		}
		if inQuote {
			switch r {
			case '\\':
				escaped = true
			case '"':
				inQuote = false
			default:
				builder.WriteRune(r)
			}
			continue
		}
		if unicode.IsSpace(r) {
			flush()
			continue
		}
		if start < 0 {
			start = index
		}
		if r == '"' {
			inQuote = true
			continue
		}
		builder.WriteRune(r)
	}
	if escaped || inQuote {
		return nil, NewIncompleteCommand()
	}
	flush()
	return tokens, nil
}

func tokensToInput(tokens []Token) string {
	if len(tokens) == 0 {
		return ""
	}
	end := tokens[len(tokens)-1].Start + len(tokens[len(tokens)-1].Text)
	buffer := make([]byte, end)
	for index := range buffer {
		buffer[index] = ' '
	}
	for _, token := range tokens {
		copy(buffer[token.Start:], token.Text)
	}
	return string(buffer)
}
