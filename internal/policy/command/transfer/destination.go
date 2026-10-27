// Purpose: Recognize literal local HTTP targets and sanitize outbound approval destinations.
// Pattern: Pure URL projection.
// Usage: The parent AST analyzer preserves local API tests and records external targets.
// Related: internal/policy/command/transfer.go, internal/policy/command/transfer/inspect.go.
package transfer

import (
	"net/netip"
	"net/url"
	"strings"
)

// LoopbackHTTP avoids DNS and accepts only explicit loopback HTTP addresses.
// Indirect routing is checked separately by the command parser.
func LoopbackHTTP(value string) bool {
	u, err := url.Parse(value)
	if err != nil || u.Opaque != "" || u.User != nil || u.Scheme != "http" && u.Scheme != "https" || strings.ContainsAny(value, "*\r\n\t\\") {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	address, err := netip.ParseAddr(host)
	return host == "localhost" || err == nil && address.IsLoopback()
}

// Display omits credentials, query values and fragments from audit targets.
func Display(value string) string {
	u, err := url.Parse(value)
	if err != nil || u.Host == "" {
		return ""
	}
	u.User, u.RawQuery, u.Fragment = nil, "", ""
	u.ForceQuery = false
	return u.String()
}
