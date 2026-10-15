// Purpose: Interpret only documented options for built-in Plan listing and search queries.
// Pattern: Pure option parsing.
// Usage: PlanQuery delegates command-specific arguments here.
// Related: internal/policy/shell/query.go.
package shell

import (
	"fmt"
	"strconv"
	"strings"
)

func badQuery() error { return fmt.Errorf("unsupported Plan query option") }

func (q *Query) listOptions(args []string) error {
	pathSet, options := false, true
	for _, arg := range args {
		if options && arg == "--" {
			options = false
			continue
		}
		if options && strings.HasPrefix(arg, "-") {
			for _, flag := range strings.TrimPrefix(arg, "-") {
				switch flag {
				case 'a', 'A':
					q.Hidden = true
				case 'l':
					q.Long = true
				case '1':
				default:
					return badQuery()
				}
			}
		} else {
			if pathSet {
				return badQuery()
			}
			q.Path = arg
			pathSet = true
		}
	}
	return nil
}

func (q *Query) findOptions(args []string) error {
	q.Hidden = true
	q.Recursive = true
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		q.Path = args[0]
		args = args[1:]
	}
	for i := 0; i < len(args); i++ {
		if args[i] == "-print" {
			continue
		}
		if i+1 >= len(args) {
			return badQuery()
		}
		flag, value := args[i], args[i+1]
		i++
		switch flag {
		case "-name":
			q.Pattern = value
		case "-type":
			if value != "f" && value != "d" {
				return badQuery()
			}
			q.Files = value == "f"
			q.Directories = value == "d"
		case "-maxdepth":
			n, err := strconv.Atoi(value)
			if err != nil || n < 0 || n > 1000 {
				return badQuery()
			}
			q.MaxDepth = n
		default:
			return badQuery()
		}
	}
	return nil
}

func (q *Query) searchOptions(args []string) error {
	q.Content = true
	q.Recursive = q.Name == "rg"
	q.Hidden = q.Name == "grep"
	var positional []string
	options := true
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if options && arg == "--" {
			options = false
			continue
		}
		if options && strings.HasPrefix(arg, "-") {
			switch arg {
			case "--files":
				if q.Name != "rg" {
					return badQuery()
				}
				q.Content = false
				q.Files = true
			case "--hidden":
				q.Hidden = true
			case "-g", "--glob":
				if q.Name != "rg" || i+1 >= len(args) {
					return badQuery()
				}
				i++
				q.Glob = args[i]
			default:
				if strings.HasPrefix(arg, "--") {
					return badQuery()
				}
				for _, flag := range strings.TrimPrefix(arg, "-") {
					switch flag {
					case 'n':
					case 'r':
						q.Recursive = true
					case 'i':
						q.Insensitive = true
					case 'F':
						q.Fixed = true
					default:
						return badQuery()
					}
				}
			}
		} else {
			positional = append(positional, arg)
		}
	}
	if q.Content {
		if len(positional) < 1 {
			return badQuery()
		}
		q.Pattern = positional[0]
		positional = positional[1:]
	}
	if len(positional) > 1 {
		return badQuery()
	}
	if len(positional) == 1 {
		q.Path = positional[0]
	}
	return nil
}
