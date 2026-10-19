// Purpose: Inspect find actions and nested commands while preserving runtime-selected target uncertainty.
// Pattern: Structured command argument analysis.
// Usage: The command effect classifier dispatches find here.
// Related: internal/policy/command/effects.go, internal/policy/command/wrappers.go.
package command

import "strings"

func (a *analyzer) find(args []word, s *state) {
	var roots []word
	i := 0
	for ; i < len(args); i++ {
		arg := args[i]
		if arg.known && (arg.text == "-L" || arg.text == "-P" || arg.text == "-H") {
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
	filtered := false
	for ; i < len(args); i++ {
		arg := args[i]
		if !arg.known {
			a.result.Ask("Dynamic find action requires inspection")
			continue
		}
		switch arg.text {
		case "-delete":
			a.result.Ask("find deletion requires approval")
			for _, root := range roots {
				if filtered && root.known {
					root.text += "/__selected_entry__"
				}
				a.target(root, s, true)
			}
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
						a.result.Ask("find supplies runtime-selected arguments")
						if root.known && j > 0 {
							nested[j] = word{text: strings.ReplaceAll(value.text, "{}", root.text+"/__selected_entry__"), known: true}
						} else {
							nested[j] = word{}
						}
					}
				}
				work := s.clone()
				if arg.text == "-execdir" || arg.text == "-okdir" {
					work.dir = ""
				}
				a.invoke(nested, nil, work)
			}
		case "-fprint", "-fprint0", "-fprintf":
			if i+1 < len(args) {
				i++
				a.target(args[i], s, false)
			}
		case "-name", "-path", "-iname", "-ipath", "-regex", "-type", "-mindepth", "-maxdepth", "-mtime", "-size", "-user", "-group":
			filtered = true
			if i+1 < len(args) {
				i++
			}
		}
	}
}
