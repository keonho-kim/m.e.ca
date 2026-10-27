// Purpose: Identify wget request bodies, query data and opaque configuration inputs.
// Pattern: Conservative command option parser.
// Usage: Inspect handles wget while ordinary downloads remain unclassified as uploads.
// Related: internal/policy/command/transfer/inspect.go, internal/policy/command/transfer/curl.go.
package transfer

import "strings"

func inspectWget(args []Argument) Inspection {
	var h httpCall
	redirects := true // Wget follows redirects unless explicitly disabled.
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
			kind := wgetLongOption(name)
			if kind == "flag" {
				continue
			}
			if kind == "unknown" {
				h.indirect = true
				continue
			}
			value := argumentValue(args, &i, attached, hasAttached)
			if name == "max-redirect" {
				redirects = !value.Known || value.Value != "0"
			} else {
				h.option(kind, value)
			}
			continue
		}
		if arg.Value == "-nv" {
			continue
		}
		for offset := 1; offset < len(arg.Value); offset++ {
			flag := arg.Value[offset]
			if strings.ContainsRune("qcvNnrpkESHhV", rune(flag)) {
				continue
			}
			kind := "unknown"
			switch flag {
			case 'O', 'o', 'a', 'P', 't', 'T', 'U', 'w', 'l', 'A', 'R':
				kind = "value"
			case 'e', 'i':
				kind = "opaque"
			}
			if kind == "unknown" {
				h.indirect = true
				continue
			}
			h.option(kind, argumentValue(args, &i, arg.Value[offset+1:], offset+1 < len(arg.Value)))
			break
		}
	}
	h.routeChange = h.routeChange || redirects
	return h.result()
}

func wgetLongOption(name string) string {
	switch name {
	case "post-data", "post-file", "body-data", "body-file", "user", "password", "http-user", "http-password", "proxy-user", "proxy-password", "load-cookies":
		return "send"
	case "method":
		return "method"
	case "header":
		return "header"
	case "config", "execute", "input-file":
		return "opaque"
	case "output-document", "output-file", "append-output", "directory-prefix", "timeout", "connect-timeout", "read-timeout", "dns-timeout", "tries", "wait", "waitretry", "user-agent", "referer", "limit-rate", "max-redirect", "ca-certificate", "certificate", "private-key", "save-cookies", "accept", "reject", "level":
		return "value"
	case "quiet", "verbose", "no-verbose", "continue", "no-check-certificate", "spider", "server-response", "content-disposition", "trust-server-names", "timestamping", "no-clobber", "recursive", "page-requisites", "convert-links", "adjust-extension", "no-parent", "no-proxy", "help", "version":
		return "flag"
	}
	return "unknown"
}
