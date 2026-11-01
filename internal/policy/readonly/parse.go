// Purpose: Parse the LAT query language into literal, explicitly allowed command invocations.
// Pattern: Fail-closed syntax boundary.
// Usage: The local read-only executor parses again immediately before starting a command.
// Related: internal/policy/readonly/queries.go, internal/tools/local/read_only_command.go.
package readonly

import (
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/syntax"
)

// Query contains only literal arguments and the supported composition operators.
// Render resolves every executable itself; caller input never becomes shell code.
type Query struct {
	args        []string
	left, right *Query
	operator    string
}

func Parse(text string) (*Query, error) {
	if len(text) > 128*1024 {
		return nil, denied("command is too large")
	}
	file, err := syntax.NewParser(syntax.Variant(syntax.LangPOSIX)).Parse(strings.NewReader(text), "LAT query")
	if err != nil {
		return nil, denied("invalid POSIX syntax: %v", err)
	}
	if len(file.Stmts) == 0 {
		return nil, denied("supply a query command")
	}
	if len(file.Stmts) > 128 {
		return nil, denied("too many sequential queries; split the request")
	}
	var query *Query
	for _, statement := range file.Stmts {
		next, err := parseStatement(statement, 0)
		if err != nil {
			return nil, err
		}
		if query == nil {
			query = next
		} else {
			query = &Query{left: query, right: next, operator: ";"}
		}
	}
	return query, nil
}

func parseStatement(statement *syntax.Stmt, depth int) (*Query, error) {
	if depth > 32 || statement.Background || statement.Negated || statement.Coprocess || len(statement.Redirs) != 0 {
		return nil, denied("background jobs, negation and redirections are unsupported")
	}
	switch command := statement.Cmd.(type) {
	case *syntax.BinaryCmd:
		operator := ""
		switch command.Op {
		case syntax.Pipe:
			operator = "|"
		case syntax.AndStmt:
			operator = "&&"
		default:
			return nil, denied("only pipelines and && may compose queries")
		}
		left, err := parseStatement(command.X, depth+1)
		if err != nil {
			return nil, err
		}
		right, err := parseStatement(command.Y, depth+1)
		if err != nil {
			return nil, err
		}
		return &Query{left: left, right: right, operator: operator}, nil
	case *syntax.CallExpr:
		if len(command.Assigns) != 0 || len(command.Args) == 0 {
			return nil, denied("environment assignments and empty commands are unsupported")
		}
		args := make([]string, 0, len(command.Args))
		for _, word := range command.Args {
			if err := literalParts(word.Parts, false); err != nil {
				return nil, err
			}
			value, err := expand.Literal(nil, word)
			if err != nil || strings.ContainsRune(value, 0) {
				return nil, denied("use literal arguments")
			}
			args = append(args, value)
		}
		if err := validate(args); err != nil {
			return nil, err
		}
		return &Query{args: args}, nil
	default:
		return nil, denied("functions, loops, groups and shell code are unsupported")
	}
}

func literalParts(parts []syntax.WordPart, quoted bool) error {
	for _, part := range parts {
		switch value := part.(type) {
		case *syntax.Lit:
			if !quoted && strings.ContainsAny(value.Value, "*?[~{}") {
				return denied("unquoted globs and tilde expansion are unsupported; use find or rg with quoted patterns")
			}
		case *syntax.SglQuoted:
			if value.Dollar {
				return denied("use ordinary quoted literal arguments")
			}
		case *syntax.DblQuoted:
			if err := literalParts(value.Parts, true); err != nil {
				return err
			}
		default:
			return denied("variables, substitutions and arithmetic are unsupported; use literal arguments")
		}
	}
	return nil
}

// Render binds executable lookup to the caller's trusted resolver.
func (query *Query) Render(resolve func(string) (string, error)) (string, error) {
	if query.operator != "" {
		left, err := query.left.Render(resolve)
		if err != nil {
			return "", err
		}
		right, err := query.right.Render(resolve)
		if err != nil {
			return "", err
		}
		return left + " " + query.operator + " " + right, nil
	}
	name, err := resolve(query.args[0])
	if err != nil {
		return "", err
	}
	args := append([]string{name}, query.args[1:]...)
	if query.args[0] == "rg" {
		args = append(args[:1], append([]string{"--no-config"}, args[1:]...)...)
	}
	if query.args[0] == "git" {
		args = gitInvocation(args)
	}
	for i, arg := range args {
		args[i] = "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
	}
	return strings.Join(args, " "), nil
}

func denied(format string, args ...any) error {
	return fmt.Errorf("LAT read-only query rejected: "+format+". Use literal read/search commands (ls, cat, head, tail, grep, rg, find, sed or git queries)", args...)
}
