// Purpose: Parse document URLs and match a private read to the current user's input.
// Pattern: Pure URL normalization and request authorization.
// Usage: Document preparation parses the target; the transport checks private destinations.
// Related: internal/tools/browser/webfetch/documents.go, internal/tools/browser/webfetch/client.go.
package webfetch

import (
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"golang.org/x/net/idna"
)

const MaxURLBytes = 2048

type origin struct {
	scheme, host, port string
}

func urlOrigin(target *url.URL) origin {
	port := target.Port()
	if port == "" {
		port = "80"
		if target.Scheme == "https" {
			port = "443"
		}
	}
	return origin{scheme: target.Scheme, host: target.Hostname(), port: port}
}

func parseURL(address string) (*url.URL, error) {
	invalid := func() error { return fault.New(diag.InvalidArgument, diag.WebFetchURL) }
	if len(address) == 0 || len(address) > MaxURLBytes {
		return nil, invalid()
	}
	target, err := url.Parse(address)
	if err != nil || target.User != nil || target.Opaque != "" ||
		target.Scheme != "http" && target.Scheme != "https" || strings.HasSuffix(target.Host, ":") {
		return nil, invalid()
	}
	host := strings.ToLower(strings.TrimSuffix(target.Hostname(), "."))
	parsedIP, ipErr := netip.ParseAddr(host)
	if ipErr == nil {
		if parsedIP.Zone() != "" {
			return nil, invalid()
		}
		host = parsedIP.String()
	} else {
		if strings.HasPrefix(target.Host, "[") {
			return nil, invalid()
		}
		if !validHostname(host) {
			host, err = idna.Lookup.ToASCII(host)
			if err != nil || !validHostname(host) {
				return nil, invalid()
			}
		}
	}
	if port := target.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 || port != strconv.Itoa(number) {
			return nil, invalid()
		}
		target.Host = net.JoinHostPort(host, port)
	} else if parsedIP.Is6() {
		target.Host = "[" + host + "]"
	} else {
		target.Host = host
	}
	if target.Path == "" {
		target.Path = "/"
	}
	// Fragments identify in-document locations and stay off the wire.
	target.Fragment, target.RawFragment = "", ""
	if len(target.String()) > MaxURLBytes {
		return nil, invalid()
	}
	return target, nil
}

func validHostname(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}
	for label := range strings.SplitSeq(host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if c != '-' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
				return false
			}
		}
	}
	return true
}

// Compare the complete on-wire URL. A reference to one private path does not
// also authorize other paths or additional query data at that origin.
func userSuppliedURL(text string, target *url.URL) bool {
	site := urlOrigin(target)
	for token := range strings.FieldsSeq(text) {
		end := strings.Index(token, "://")
		if end < 4 {
			continue
		}
		start := end - 4
		if end >= 5 && strings.EqualFold(token[end-5:end], "https") {
			start = end - 5
		}
		if start > 0 && !strings.ContainsRune("(<[\"'`", rune(token[start-1])) {
			continue
		}
		address := token[start:]
		for _, candidate := range [2]string{address, strings.TrimRight(address, ">)],.;!?\"'`")} {
			url, err := parseURL(candidate)
			if err == nil && urlOrigin(url) == site && url.EscapedPath() == target.EscapedPath() &&
				url.RawQuery == target.RawQuery && url.ForceQuery == target.ForceQuery {
				return true
			}
		}
	}
	return false
}
