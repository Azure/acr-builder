// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// splitCommandLine converts a task cmd string into arguments using the same
// rules on every operating system:
//   - ASCII space, tab, carriage return, and line feed separate arguments
//     outside quotes. Other Unicode whitespace is literal data.
//   - Single and double quotes group text and are removed. Empty quotes create
//     an empty argument, and adjacent quoted and unquoted text is concatenated.
//   - Outside quotes, a single backslash escapes an ASCII separator or either
//     quote character. Doubled backslashes and unknown escapes are preserved.
//   - Inside double quotes, a single backslash escapes only a double quote;
//     doubled backslashes are preserved. Inside single quotes, backslashes are
//     always literal.
//   - Shell metacharacters have no special meaning and are preserved literally.
//
// Invalid UTF-8, NUL characters, and unterminated quotes are rejected.
func splitCommandLine(command string) ([]string, error) {
	if !utf8.ValidString(command) {
		return nil, errors.New("command contains invalid UTF-8")
	}

	var args []string
	var current strings.Builder
	var quote rune
	argumentStarted := false
	runes := []rune(command)

	appendArgument := func() {
		args = append(args, current.String())
		current.Reset()
		argumentStarted = false
	}

	for index := 0; index < len(runes); index++ {
		currentRune := runes[index]
		if currentRune == 0 {
			return nil, errors.New("command contains a NUL character")
		}

		if quote != 0 {
			if currentRune == quote {
				quote = 0
				continue
			}
			if quote == '"' && currentRune == '\\' && index+1 < len(runes) {
				nextRune := runes[index+1]
				if nextRune == '\\' {
					current.WriteRune(currentRune)
					current.WriteRune(nextRune)
					index++
					continue
				}
				if nextRune == '"' {
					current.WriteRune(nextRune)
					index++
					continue
				}
			}
			current.WriteRune(currentRune)
			continue
		}

		switch {
		case isCommandSeparator(currentRune):
			if argumentStarted {
				appendArgument()
			}
		case currentRune == '\'' || currentRune == '"':
			quote = currentRune
			argumentStarted = true
		case currentRune == '\\' && index+1 < len(runes):
			nextRune := runes[index+1]
			if nextRune == '\\' {
				current.WriteRune(currentRune)
				current.WriteRune(nextRune)
				argumentStarted = true
				index++
				continue
			}
			if isCommandSeparator(nextRune) || nextRune == '\'' || nextRune == '"' {
				current.WriteRune(nextRune)
				argumentStarted = true
				index++
				continue
			}
			current.WriteRune(currentRune)
			argumentStarted = true
		default:
			current.WriteRune(currentRune)
			argumentStarted = true
		}
	}

	if quote != 0 {
		return nil, fmt.Errorf("command contains an unterminated %q quote", quote)
	}
	if argumentStarted {
		appendArgument()
	}

	return args, nil
}

func isCommandSeparator(character rune) bool {
	switch character {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}
