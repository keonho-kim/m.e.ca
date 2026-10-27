// Purpose: Identify curl request payloads, query data and indirect transfer routes.
// Pattern: Conservative command option parser.
// Usage: Inspect handles curl without running it or opening configuration files.
// Related: internal/policy/command/transfer/inspect.go, internal/policy/command/transfer/http_options.go.
package transfer

import "strings"

func inspectCurl(args []Argument) Inspection {
	var h httpCall
	operands := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !arg.Known {
			h.indirect = true
			continue
		}
		if operands || !strings.HasPrefix(arg.Value, "-") || arg.Value == "-" {
			h.operand(arg)
			continue
		}
		if arg.Value == "--" {
			operands = true
			continue
		}
		if strings.HasPrefix(arg.Value, "--") {
			name, attached, hasAttached := strings.Cut(arg.Value[2:], "=")
			kind := curlLongOption(name)
			value := Argument{}
			if kind != "flag" && kind != "redirect" && kind != "unknown" {
				value = argumentValue(args, &i, attached, hasAttached)
			}
			h.option(kind, value)
			continue
		}
		for offset := 1; offset < len(arg.Value); offset++ {
			flag := arg.Value[offset]
			kind := curlShortOption(flag)
			if kind == "flag" || kind == "redirect" {
				h.option(kind, Argument{})
				continue
			}
			if kind == "unknown" {
				h.indirect = true
				continue
			}
			h.option(kind, argumentValue(args, &i, arg.Value[offset+1:], offset+1 < len(arg.Value)))
			break
		}
	}
	return h.result()
}
