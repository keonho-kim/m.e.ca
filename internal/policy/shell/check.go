// Purpose: Route direct shell invocations covered by dedicated tools back to those tools.
// Pattern: Shell AST policy classifier; not a filesystem sandbox.
// Usage: Tool preparation calls Check before requesting shell execution approval.
// Related: internal/tools/local/prepare.go, internal/policy/shell/check_test.go.
package shell

import (
	"fmt"
	"mvdan.cc/sh/v3/syntax"
	"path/filepath"
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
			target = "read_file"
		case "ls":
			target = "list_directory"
		case "grep", "egrep", "fgrep", "rg":
			target = "grep"
		case "find":
			target = "search_files (or delete_file for deletion)"
		case "mkdir":
			target = "create_directory"
		case "mv":
			target = "move_file"
		case "touch", "truncate":
			target = "write_file"
		case "sed":
			for _, arg := range args[index+1:] {
				if arg == "--in-place" || strings.HasPrefix(arg, "-i") {
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
