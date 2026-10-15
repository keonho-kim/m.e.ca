// Purpose: Route direct shell invocations covered by dedicated tools back to those tools.
// Pattern: Shell AST policy classifier; not a filesystem sandbox.
// Usage: Tool preparation calls Check before requesting shell execution approval.
// Related: internal/tools/local/prepare.go, internal/policy/shell/check_test.go.
package shell

import (
	"fmt"
	"mvdan.cc/sh/v3/syntax"
	"path/filepath"
	"strconv"
	"strings"
)

type Dedicated struct{ Command, Tool string }

func (e *Dedicated) Error() string {
	return fmt.Sprintf("use %s instead of the direct %s shell command", e.Tool, e.Command)
}

func Check(text string) error { return check(text, 0) }
func check(text string, depth int) error {
	if depth > 8 {
		return fmt.Errorf("nested shell command exceeds inspection limit")
	}
	tree, err := syntax.NewParser(syntax.Variant(syntax.LangPOSIX)).Parse(strings.NewReader(text), "")
	if err != nil {
		return fmt.Errorf("invalid shell command: %w", err)
	}
	syntax.Walk(tree, func(node syntax.Node) bool {
		if err != nil {
			return false
		}
		call, ok := node.(*syntax.CallExpr)
		if !ok || len(call.Args) == 0 {
			return true
		}
		args := make([]string, len(call.Args))
		for i, word := range call.Args {
			args[i], _ = literal(word.Parts)
		}
		index := 0
		for index < len(args) {
			name := filepath.Base(args[index])
			if name != "env" && name != "command" && name != "exec" {
				break
			}
			index++
			for index < len(args) && (strings.HasPrefix(args[index], "-") || strings.Contains(args[index], "=")) {
				index++
			}
		}
		if index >= len(args) {
			return true
		}
		name := filepath.Base(args[index])
		target := ""
		switch name {
		case "rm", "rmdir", "unlink":
			target = "delete_file"
		case "cat", "head", "tail":
			if !stdinOnly(name, args[index+1:]) {
				target = "read_file"
			}
		case "find":
			for _, word := range call.Args[index+1:] {
				if _, ok := literal(word.Parts); !ok {
					err = fmt.Errorf("find requires literal arguments for action inspection")
					return false
				}
			}
			for _, arg := range args[index+1:] {
				if arg == "-delete" {
					target = "delete_file"
				}
				if strings.HasPrefix(arg, "-fprint") || arg == "-fls" {
					target = "write_file"
				}
				if strings.HasPrefix(arg, "-exec") || strings.HasPrefix(arg, "-ok") {
					err = fmt.Errorf("find executable actions are unsupported; run a separately reviewed command or dedicated file tool")
					return false
				}
			}
		case "rg":
			for _, arg := range args[index+1:] {
				if arg == "--pre" || strings.HasPrefix(arg, "--pre=") {
					err = fmt.Errorf("rg preprocessing executes external commands; run a separate reviewed command")
					return false
				}
			}
		case "mkdir":
			target = "create_directory"
		case "mv":
			target = "move_file"
		case "touch", "truncate":
			target = "write_file"
		case "sed":
			for _, arg := range args[index+1:] {
				if arg == "--in-place" || strings.HasPrefix(arg, "--in-place=") || strings.HasPrefix(arg, "-i") {
					target = "edit_file"
				}
			}
		case "git":
			if index+1 < len(args) && args[index+1] == "rm" {
				target = "delete_file, followed by git add -u when requested"
			}
		case "sh", "bash", "zsh", "dash":
			for i := index + 1; i+1 < len(args); i++ {
				if args[i] == "-c" && args[i+1] != "" {
					err = check(args[i+1], depth+1)
					break
				}
			}
		}
		if target != "" {
			err = &Dedicated{Command: name, Tool: target}
			return false
		}
		return true
	})
	return err
}
func literal(parts []syntax.WordPart) (string, bool) {
	var out strings.Builder
	for _, part := range parts {
		switch p := part.(type) {
		case *syntax.Lit:
			out.WriteString(p.Value)
		case *syntax.SglQuoted:
			if p.Dollar {
				return "", false
			}
			out.WriteString(p.Value)
		case *syntax.DblQuoted:
			if p.Dollar {
				return "", false
			}
			text, ok := literal(p.Parts)
			if !ok {
				return "", false
			}
			out.WriteString(text)
		default:
			return "", false
		}
	}
	return out.String(), true
}

// Permit stdin filters in search pipelines without routing them to file reads.
func stdinOnly(name string, args []string) bool {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-" {
			continue
		}
		if name == "cat" {
			if arg == "-n" || arg == "-b" || arg == "-s" {
				continue
			}
			return false
		}
		if arg == "-n" || arg == "-c" || arg == "--lines" || arg == "--bytes" {
			i++
			if i >= len(args) {
				return false
			}
			if _, err := strconv.Atoi(args[i]); err != nil {
				return false
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			value := strings.TrimPrefix(arg, "-")
			if _, err := strconv.Atoi(value); err == nil {
				continue
			}
		}
		return false
	}
	return true
}
