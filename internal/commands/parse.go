// Purpose: Parse local commands without involving a model or shell.
// Pattern: Pure Function.
// Usage: Imported by terminal input handling; tested with go test ./internal/commands.
// Related: internal/commands/parse_test.go, internal/commands/catalog.go, internal/tui, SYSTEM.md.

package commands

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Kind uint8

const (
	Empty Kind = iota
	Message
	Model
	Directory
	Quit
	Exit
	NewSession
	Language
	Permission
)

const MaxInputBytes = 16 * 1024 * 1024

type Command struct {
	Kind     Kind
	Argument string
}

func Parse(input string) (Command, error) {
	if len(input) > MaxInputBytes || !utf8.ValidString(input) || strings.ContainsRune(input, 0) {
		return Command{}, fmt.Errorf("invalid or oversized input")
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return Command{Kind: Empty}, nil
	}
	if input == ":q" {
		return Command{Kind: Quit}, nil
	}
	if !strings.HasPrefix(input, "!") {
		return Command{Kind: Message, Argument: input}, nil
	}
	words, err := quotedWords(input)
	if err != nil {
		return Command{}, err
	}
	var kind Kind
	for _, command := range catalog {
		if words[0] == command.name {
			kind = command.kind
			break
		}
	}
	if kind == Empty {
		return Command{}, fmt.Errorf("unknown command: %s", words[0])
	}
	if len(words) > 1 && kind != Directory || len(words) > 2 {
		return Command{}, fmt.Errorf("invalid arguments for %s", words[0])
	}
	command := Command{Kind: kind}
	if len(words) == 2 {
		command.Argument = words[1]
	}
	return command, nil
}

// Quoting only groups literal paths; it never expands variables or executes text.
func quotedWords(input string) ([]string, error) {
	var words []string
	var word strings.Builder
	var quote rune
	escaped, started := false, false
	for _, char := range input {
		if escaped {
			word.WriteRune(char)
			escaped, started = false, true
			continue
		}
		if char == '\\' && quote != '\'' {
			escaped, started = true, true
			continue
		}
		if quote != 0 {
			if char == quote {
				quote = 0
			} else {
				word.WriteRune(char)
			}
			continue
		}
		if char == '\'' || char == '"' {
			quote, started = char, true
			continue
		}
		if unicode.IsSpace(char) {
			if started {
				words = append(words, word.String())
				word.Reset()
				started = false
			}
			continue
		}
		word.WriteRune(char)
		started = true
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("unterminated quote or escape")
	}
	if started {
		words = append(words, word.String())
	}
	return words, nil
}
