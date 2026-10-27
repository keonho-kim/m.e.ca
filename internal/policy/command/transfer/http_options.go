// Purpose: Classify curl options by payload, destination and routing effects.
// Pattern: Fixed protocol option table.
// Usage: The curl parser consumes option values without mistaking them for URLs.
// Related: internal/policy/command/transfer/curl.go.
package transfer

import "strings"

func curlLongOption(name string) string {
	switch name {
	case "data", "data-ascii", "data-binary", "data-raw", "data-urlencode", "json", "form", "form-string", "upload-file", "url-query", "user", "proxy-user", "cookie", "oauth2-bearer", "aws-sigv4", "quote", "mail-from", "mail-rcpt", "request-target":
		return "send"
	case "request":
		return "method"
	case "url":
		return "url"
	case "header", "proxy-header":
		return "header"
	case "config", "variable", "netrc-file":
		return "opaque"
	case "proxy", "preproxy", "socks4", "socks4a", "socks5", "socks5-hostname", "connect-to", "resolve", "unix-socket", "abstract-unix-socket", "haproxy-clientip":
		return "route"
	case "location", "location-trusted", "follow":
		return "redirect"
	case "netrc", "netrc-optional":
		return "unknown"
	case "output", "output-dir", "write-out", "max-time", "connect-timeout", "retry", "retry-delay", "retry-max-time", "max-redirs", "max-filesize", "limit-rate", "speed-time", "speed-limit", "parallel-max", "user-agent", "referer", "range", "cacert", "capath", "cert", "cert-type", "key", "key-type", "pass", "ciphers", "tls13-ciphers", "interface", "local-port", "proto", "proto-redir", "request-rate", "stderr", "time-cond", "etag-save", "etag-compare", "dump-header":
		return "value"
	case "silent", "show-error", "fail", "fail-with-body", "compressed", "insecure", "head", "get", "globoff", "no-progress-meter", "progress-bar", "verbose", "include", "remote-name", "remote-header-name", "remote-time", "create-dirs", "retry-all-errors", "retry-connrefused", "http1.0", "http1.1", "http2", "http2-prior-knowledge", "http3", "ipv4", "ipv6", "parallel", "parallel-immediate", "next", "no-buffer", "disable", "help", "version", "no-location", "no-location-trusted", "no-follow":
		return "flag"
	}
	// Do not guess what abbreviated or newly added options do. In particular,
	// --dat, --conf and --expand-* must not bypass payload/config inspection.
	if strings.HasPrefix(name, "expand-") {
		return "opaque"
	}
	return "unknown"
}

func curlShortOption(flag byte) string {
	switch flag {
	case 'd', 'F', 'T', 'u', 'U', 'b', 'Q':
		return "send"
	case 'X':
		return "method"
	case 'H':
		return "header"
	case 'K':
		return "opaque"
	case 'x':
		return "route"
	case 'L':
		return "redirect"
	case 'o', 'w', 'm', 'A', 'e', 'r', 'C', 'D', 'z', 'c':
		return "value"
	case 's', 'S', 'f', 'k', 'I', 'G', 'g', 'v', 'i', 'O', 'J', 'R', 'N', 'q', '4', '6', '#', 'h', 'V', '0', '1', '2', '3', ':':
		return "flag"
	}
	return "unknown"
}
