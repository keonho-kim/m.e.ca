// Purpose: Accept verification shell structures whose top-level failures reach the exit status.
// Pattern: Conservative AST policy, independent of filesystem containment.
// Usage: Local tool preparation applies CheckVerification only to purpose=check.
// Related: internal/policy/shell/check.go, internal/tools/local/prepare.go.
package shell

import (
	"fmt"
	"mvdan.cc/sh/v3/syntax"
	"path/filepath"
	"strings"
)

func CheckVerification(text string) error {
	tree, err := syntax.NewParser(syntax.Variant(syntax.LangPOSIX)).Parse(strings.NewReader(text), "")
	if err != nil {
		return err
	}
	invalid := func() error {
		return fmt.Errorf("verification command must be one foreground command or an && chain, with literal arguments; split wrappers, pipelines and status-masking commands into separate checks")
	}
	if len(tree.Stmts) != 1 {
		return invalid()
	}
	var statement func(*syntax.Stmt) bool
	statement = func(stmt *syntax.Stmt) bool {
		if stmt.Background || stmt.Negated || stmt.Coprocess {
			return false
		}
		for _, redirect := range stmt.Redirs {
			if redirect.Hdoc != nil || redirect.Word == nil {
				return false
			}
			if _, ok := literal(redirect.Word.Parts); !ok {
				return false
			}
		}
		switch cmd := stmt.Cmd.(type) {
		case *syntax.BinaryCmd:
			return cmd.Op == syntax.AndStmt && statement(cmd.X) && statement(cmd.Y)
		case *syntax.CallExpr:
			if len(cmd.Args) == 0 {
				return false
			}
			for _, arg := range cmd.Args {
				if _, ok := literal(arg.Parts); !ok {
					return false
				}
			}
			for _, assignment := range cmd.Assigns {
				if assignment.Array != nil || assignment.Value == nil {
					return false
				}
				if _, ok := literal(assignment.Value.Parts); !ok {
					return false
				}
			}
			name, _ := literal(cmd.Args[0].Parts)
			switch filepath.Base(name) {
			case "sh", "bash", "zsh", "dash", "fish", "eval", "exec", "exit", "return", "true", ":", "echo", "printf", "env", "command", "source", ".", "set", "trap":
				return false
			}
			return true
		default:
			return false
		}
	}
	if !statement(tree.Stmts[0]) {
		return invalid()
	}
	return nil
}
