// Purpose: Describe explicit shell transfer effects independently of AST traversal and authorization.
// Pattern: Pure command argument adapter.
// Usage: The parent AST analyzer passes expanded words and applies outbound policy to each result.
// Related: internal/policy/command/transfer/curl.go, internal/policy/command/transfer/wget.go, internal/policy/command/transfer/remote.go.
package transfer

import (
	"net/url"
	"strings"
)

type Argument struct {
	Value string
	Known bool
}

type Inspection struct {
	Destinations []string
	Indirect     bool
}

func Inspect(name string, args []Argument) Inspection {
	switch name {
	case "curl":
		return inspectCurl(args)
	case "wget":
		return inspectWget(args)
	case "scp", "rsync", "sftp", "ssh":
		return inspectRemote(name, args)
	}
	return Inspection{}
}

type httpCall struct {
	urls                         []string
	sends, indirect, routeChange bool
}

func (h httpCall) result() Inspection {
	result := Inspection{Indirect: h.indirect}
	for _, value := range h.urls {
		u, err := url.Parse(value)
		query := err == nil && (u.RawQuery != "" || u.User != nil)
		if h.sends || query {
			result.Destinations = append(result.Destinations, value)
		}
	}
	if h.sends && len(h.urls) == 0 || len(result.Destinations) > 0 && h.routeChange {
		result.Indirect = true
	}
	return result
}

func (h *httpCall) operand(arg Argument) {
	if !arg.Known {
		h.indirect = true
		return
	}
	value := arg.Value
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	h.urls = append(h.urls, value)
}

func (h *httpCall) option(kind string, value Argument) {
	switch kind {
	case "send", "header":
		h.sends = true // Arbitrary header names can carry file contents or credentials.
	case "method":
		h.sends = h.sends || methodSends(value)
	case "url":
		h.operand(value)
	case "redirect", "route":
		h.routeChange = true
	case "opaque", "unknown":
		h.indirect = true
	}
}

func argumentValue(args []Argument, index *int, attached string, hasAttached bool) Argument {
	if hasAttached {
		return Argument{Value: attached, Known: true}
	}
	*index = *index + 1
	if *index >= len(args) {
		return Argument{}
	}
	return args[*index]
}

func methodSends(value Argument) bool {
	if !value.Known {
		return true
	}
	switch strings.ToUpper(value.Value) {
	case "GET", "HEAD", "OPTIONS":
		return false
	default:
		return true
	}
}
