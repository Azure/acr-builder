// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

// These cases are the executable specification for the cmd parsing rules in docs/task.md.
func TestSplitCommandLine(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
	}{
		{
			name:     "empty command",
			command:  "",
			expected: nil,
		},
		{
			name:     "whitespace only",
			command:  " \t\r\n ",
			expected: nil,
		},
		{
			name:     "simple command",
			command:  "docker images",
			expected: []string{"docker", "images"},
		},
		{
			name:     "leading trailing and repeated whitespace",
			command:  " \t image  first\r\nsecond \n ",
			expected: []string{"image", "first", "second"},
		},
		{
			name:     "unicode whitespace is literal",
			command:  "image\u00a0first\u2003second",
			expected: []string{"image\u00a0first\u2003second"},
		},
		{
			name:     "other ASCII whitespace is literal",
			command:  "image first\vsecond\fthird",
			expected: []string{"image", "first\vsecond\fthird"},
		},
		{
			name:     "unicode arguments",
			command:  `image "こんにちは 世界" café λ`,
			expected: []string{"image", "こんにちは 世界", "café", "λ"},
		},
		{
			name:     "quoted and empty arguments",
			command:  `bash echo "hello world" 'single quoted' ""`,
			expected: []string{"bash", "echo", "hello world", "single quoted", ""},
		},
		{
			name:     "empty single and double quoted arguments",
			command:  `image '' "" ''`,
			expected: []string{"image", "", "", ""},
		},
		{
			name:     "concatenated quoted argument",
			command:  `image pre"mid dle"post`,
			expected: []string{"image", "premid dlepost"},
		},
		{
			name:     "multiple quoted segments concatenate",
			command:  `image pre"mid dle"'post fix'`,
			expected: []string{"image", "premid dlepost fix"},
		},
		{
			name:     "empty quotes concatenate with text",
			command:  `image a""b ''suffix prefix''`,
			expected: []string{"image", "ab", "suffix", "prefix"},
		},
		{
			name:     "opposite quote characters are literal",
			command:  `image "it's literal" 'say "hello"'`,
			expected: []string{"image", "it's literal", `say "hello"`},
		},
		{
			name:     "quoted whitespace is preserved",
			command:  "image \"line one\nline two\tend\"",
			expected: []string{"image", "line one\nline two\tend"},
		},
		{
			name:     "escaped whitespace",
			command:  `image hello\ world`,
			expected: []string{"image", "hello world"},
		},
		{
			name:     "escaped tab and newline",
			command:  "image tab\\" + "\t" + "joined newline\\" + "\n" + "joined",
			expected: []string{"image", "tab\tjoined", "newline\njoined"},
		},
		{
			name:     "backslash before unicode whitespace is literal",
			command:  "image first\\" + "\u00a0" + "second",
			expected: []string{"image", "first\\\u00a0second"},
		},
		{
			name:     "escaped quotes outside quotes",
			command:  `image say\"hello\" it\'s`,
			expected: []string{"image", `say"hello"`, "it's"},
		},
		{
			name:     "doubled backslash outside quotes is preserved",
			command:  `image one\\two`,
			expected: []string{"image", `one\\two`},
		},
		{
			name:     "doubled backslash before whitespace is preserved",
			command:  `image first\\ second`,
			expected: []string{"image", `first\\`, "second"},
		},
		{
			name:     "backslash before ordinary characters is literal",
			command:  `image \q C:\Windows path\name`,
			expected: []string{"image", `\q`, `C:\Windows`, `path\name`},
		},
		{
			name:     "trailing backslash is literal",
			command:  `image trailing\`,
			expected: []string{"image", `trailing\`},
		},
		{
			name:     "escaped double quotes inside double quotes",
			command:  `image "say \"hello\""`,
			expected: []string{"image", `say "hello"`},
		},
		{
			name:     "doubled backslashes inside double quotes are preserved",
			command:  `image "C:\\temp\\file"`,
			expected: []string{"image", `C:\\temp\\file`},
		},
		{
			name:     "other backslashes inside double quotes are literal",
			command:  `image "line\n dollar\$ space\ value"`,
			expected: []string{"image", `line\n dollar\$ space\ value`},
		},
		{
			name:     "backslashes inside single quotes are literal",
			command:  `image 'C:\\temp\file' 'say\"hello'`,
			expected: []string{"image", `C:\\temp\file`, `say\"hello`},
		},
		{
			name:     "windows paths",
			command:  `image "C:\Program Files\tool" C:\Windows`,
			expected: []string{"image", `C:\Program Files\tool`, `C:\Windows`},
		},
		{
			name:     "doubled backslash before closing quote is preserved",
			command:  `image "C:\Program Files\\"`,
			expected: []string{"image", `C:\Program Files\\`},
		},
		{
			name:     "single quoted windows path ending in backslash",
			command:  `image 'C:\Program Files\'`,
			expected: []string{"image", `C:\Program Files\`},
		},
		{
			name:     "quoted UNC path",
			command:  `image "\\server\share name"`,
			expected: []string{"image", `\\server\share name`},
		},
		{
			name:     "mixed path separators",
			command:  `image C:\workspace/source ./relative/path /absolute/path`,
			expected: []string{"image", `C:\workspace/source`, "./relative/path", "/absolute/path"},
		},
		{
			name:     "literal shell syntax",
			command:  `image echo "$(id)" ';' '|' '&&' '>'`,
			expected: []string{"image", "echo", "$(id)", ";", "|", "&&", ">"},
		},
		{
			name:    "unquoted shell metacharacters are literal",
			command: "image $HOME ${HOME} $(id) `whoami` ; & | || && > >> < << 2>&1 * ? [abc] ~ # ( )",
			expected: []string{
				"image", "$HOME", "${HOME}", "$(id)", "`whoami`", ";", "&", "|", "||", "&&",
				">", ">>", "<", "<<", "2>&1", "*", "?", "[abc]", "~", "#", "(", ")",
			},
		},
		{
			name:     "attached shell metacharacters are literal",
			command:  `image echo;id foo|bar a&&b out>file in<file *.txt`,
			expected: []string{"image", "echo;id", "foo|bar", "a&&b", "out>file", "in<file", "*.txt"},
		},
		{
			name:     "shell comment marker does not start a comment",
			command:  `image echo # this is still data`,
			expected: []string{"image", "echo", "#", "this", "is", "still", "data"},
		},
		{
			name:     "quoted command substitution with spaces is literal",
			command:  `image echo "$(touch /tmp/should-not-exist)"`,
			expected: []string{"image", "echo", "$(touch /tmp/should-not-exist)"},
		},
		{
			name:     "platform variable syntaxes are literal",
			command:  `image FOO=bar %PATH% $env:PATH !PATH! ^caret @echo`,
			expected: []string{"image", "FOO=bar", "%PATH%", "$env:PATH", "!PATH!", "^caret", "@echo"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := splitCommandLine(test.command)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(actual, test.expected) {
				t.Fatalf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func TestSplitCommandLineRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		errorContains string
	}{
		{name: "unterminated empty double quote", command: `image "`, errorContains: "unterminated"},
		{name: "unterminated empty single quote", command: "image '", errorContains: "unterminated"},
		{name: "unterminated double quote after text", command: `image "unterminated`, errorContains: "unterminated"},
		{name: "unterminated single quote after text", command: "image 'unterminated", errorContains: "unterminated"},
		{name: "unterminated quote concatenated with text", command: `image prefix"unterminated`, errorContains: "unterminated"},
		{name: "escaped closing double quote", command: `image "unterminated\"`, errorContains: "unterminated"},
		{name: "trailing backslash in double quotes", command: `image "unterminated\`, errorContains: "unterminated"},
		{name: "mismatched single quote", command: `image "wrong quote'`, errorContains: "unterminated"},
		{name: "mismatched double quote", command: `image 'wrong quote"`, errorContains: "unterminated"},
		{name: "NUL only", command: "\x00", errorContains: "NUL character"},
		{name: "NUL at start", command: "\x00image", errorContains: "NUL character"},
		{name: "NUL between arguments", command: "image first\x00second", errorContains: "NUL character"},
		{name: "NUL at end", command: "image\x00", errorContains: "NUL character"},
		{name: "NUL inside double quotes", command: "image \"foo\x00bar\"", errorContains: "NUL character"},
		{name: "NUL inside single quotes", command: "image 'foo\x00bar'", errorContains: "NUL character"},
		{name: "invalid UTF-8 only", command: "\xff", errorContains: "invalid UTF-8"},
		{name: "invalid UTF-8 in argument", command: "image foo\xffbar", errorContains: "invalid UTF-8"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual, err := splitCommandLine(test.command)
			if err == nil {
				t.Fatalf("expected %q to be rejected, got %v", test.command, actual)
			}
			if actual != nil {
				t.Fatalf("expected no partial arguments on error, got %v", actual)
			}
			if !strings.Contains(err.Error(), test.errorContains) {
				t.Fatalf("expected error %q to contain %q", err, test.errorContains)
			}
		})
	}
}

func FuzzSplitCommandLine(f *testing.F) {
	for _, command := range []string{
		"",
		"docker images",
		" \t\r\n ",
		"image\u00a0first\u2003second",
		`image "hello world" 'single quoted' ""`,
		`image pre"mid dle"post`,
		`image hello\ world say\"hello`,
		`image 'C:\Program Files\' "\\server\share name"`,
		`image $(id) ; | && > $HOME %PATH%`,
		`image "unterminated`,
		"image foo\x00bar",
		string([]byte{'i', 'm', 'a', 'g', 'e', ' ', 0xff}),
	} {
		f.Add(command)
	}

	f.Fuzz(func(t *testing.T, command string) {
		actual, err := splitCommandLine(command)
		repeated, repeatedErr := splitCommandLine(command)

		if (err == nil) != (repeatedErr == nil) {
			t.Fatalf("parser returned inconsistent errors for %q: %v and %v", command, err, repeatedErr)
		}
		if err != nil && err.Error() != repeatedErr.Error() {
			t.Fatalf("parser returned different errors for %q: %v and %v", command, err, repeatedErr)
		}
		if !reflect.DeepEqual(actual, repeated) {
			t.Fatalf("parser returned different arguments for %q: %v and %v", command, actual, repeated)
		}

		if !utf8.ValidString(command) {
			if err == nil {
				t.Fatalf("expected invalid UTF-8 command to be rejected: %q", command)
			}
			if actual != nil {
				t.Fatalf("expected no partial arguments for invalid UTF-8 command, got %v", actual)
			}
			return
		}

		if strings.ContainsRune(command, '\x00') {
			if err == nil {
				t.Fatalf("expected command containing NUL to be rejected: %q", command)
			}
			if actual != nil {
				t.Fatalf("expected no partial arguments for command containing NUL, got %v", actual)
			}
			return
		}

		if err != nil {
			if actual != nil {
				t.Fatalf("expected no partial arguments on error, got %v", actual)
			}
			return
		}

		for _, argument := range actual {
			if strings.ContainsRune(argument, '\x00') {
				t.Fatalf("successful parse returned an argument containing NUL: %q", argument)
			}
		}

		hasGroupingSyntax := strings.ContainsRune(command, '\'') ||
			strings.ContainsRune(command, '"') ||
			strings.ContainsRune(command, '\\')
		if !hasGroupingSyntax {
			expected := strings.FieldsFunc(command, func(character rune) bool {
				switch character {
				case ' ', '\t', '\r', '\n':
					return true
				default:
					return false
				}
			})
			if len(actual) != len(expected) {
				t.Fatalf("plain input %q: expected %v, got %v", command, expected, actual)
			}
			for index := range expected {
				if actual[index] != expected[index] {
					t.Fatalf("plain input %q: expected %v, got %v", command, expected, actual)
				}
			}
		}
	})
}
