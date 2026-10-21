// Purpose: Inspect find actions and root participation while preserving runtime-selected target uncertainty.
// Pattern: Structured command argument analysis.
// Usage: The command effect classifier dispatches find here.
// Related: internal/policy/command/effects.go, internal/policy/command/wrappers.go.
package command

import (
	"path/filepath"
	"strconv"
	"strings"
)

func (a *analyzer) find(args []word, s *state) {
	var roots []word
	follows := false
	i := 0
	for ; i < len(args); i++ {
		arg := args[i]
		if arg.known && (arg.text == "-L" || arg.text == "-P" || arg.text == "-H") {
			follows = arg.text != "-P"
			continue
		}
		if arg.known && (strings.HasPrefix(arg.text, "-") || arg.text == "!" || arg.text == "(") {
			break
		}
		roots = append(roots, arg)
	}
	if len(roots) == 0 {
		roots = []word{{text: ".", known: true}}
	}
	filtered, deletion, descendants, conjunctive := false, false, false, true
	excludesRoot := make([]bool, len(roots))
	deletesRoot := make([]bool, len(roots))
	for ; i < len(args); i++ {
		arg := args[i]
		if !arg.known {
			a.ask("Dynamic find action requires inspection")
			continue
		}
		switch arg.text {
		case "-delete":
			deletion = true
			for j := range roots {
				deletesRoot[j] = deletesRoot[j] || !excludesRoot[j]
			}
		case "!", "-not", "-o", "-or", ",":
			conjunctive = false
		case "-follow":
			follows = true
		case "-exec", "-execdir", "-ok", "-okdir":
			start := i + 1
			for i++; i < len(args); i++ {
				if args[i].known && (args[i].text == ";" || args[i].text == "+") {
					break
				}
			}
			for _, root := range roots {
				nested := append([]word(nil), args[start:i]...)
				for j, value := range nested {
					if strings.Contains(value.text, "{}") {
						if root.known && j > 0 {
							nested[j] = word{text: strings.ReplaceAll(value.text, "{}", root.text+"/*"), glob: true}
						} else {
							nested[j] = word{}
						}
					}
				}
				work := s.clone()
				if arg.text == "-execdir" || arg.text == "-okdir" {
					work.dir = ""
				}
				a.invoke(nested, nil, work, true)
			}
		case "-fprint", "-fprint0", "-fprintf":
			if i+1 < len(args) {
				i++
				a.target(args[i], s, false)
			}
			if arg.text == "-fprintf" && i+1 < len(args) {
				i++
			}
		case "-printf":
			if i+1 < len(args) {
				i++
			}
		case "-name", "-path", "-iname", "-ipath", "-regex", "-type", "-mindepth", "-maxdepth", "-mtime", "-size", "-user", "-group":
			filtered = true
			if i+1 < len(args) {
				i++
				value := args[i]
				if arg.text == "-mindepth" {
					descendants = false
				}
				if !value.known {
					continue
				}
				if arg.text == "-mindepth" {
					depth, err := strconv.Atoi(value.text)
					descendants = err == nil && depth > 0
				}
				for j, root := range roots {
					if arg.text == "-type" && len(value.text) == 1 && strings.Contains("flbcps", value.text) {
						excludesRoot[j] = true // A session project root is a directory.
					}
					if (arg.text == "-name" || arg.text == "-iname") && root.known && !strings.ContainsAny(value.text, "[]\\") {
						name, pattern := filepath.Base(strings.TrimRight(root.text, "/")), value.text
						if arg.text == "-iname" {
							name, pattern = strings.ToLower(name), strings.ToLower(pattern)
						}
						matches, err := filepath.Match(pattern, name)
						excludesRoot[j] = excludesRoot[j] || err == nil && !matches
					}
				}
			}
		}
	}
	if deletion {
		for i, root := range roots {
			if descendants || conjunctive && !deletesRoot[i] {
				a.directoryTarget(root, s, true)
			} else {
				a.target(root, s, true)
				if follows {
					a.directoryTarget(root, s, true)
				}
			}
		}
		if filtered || follows {
			a.ask("Mutation target cannot be resolved statically")
		}
	}
}
