// Purpose: Parse HTTP gzip and identity preferences for compressed assets and JSON replies.
// Pattern: Pure HTTP content negotiation.
// Usage: Embedded assets and native API replies share this negotiation rule.
// Related: internal/webui/assets.go, internal/serverapi/http_compression.go.
package httpencoding

import (
	"strconv"
	"strings"
)

func Accept(header string) (gzip, identity bool) {
	identity = true
	var wildcard float64 = -1
	explicit, identityExplicit := false, false
	for _, part := range strings.Split(header, ",") {
		words := strings.Split(part, ";")
		name := strings.ToLower(strings.TrimSpace(words[0]))
		q := 1.0
		for _, word := range words[1:] {
			if value, ok := strings.CutPrefix(strings.TrimSpace(word), "q="); ok {
				valueQ, err := strconv.ParseFloat(value, 64)
				if err != nil || valueQ < 0 || valueQ > 1 {
					q = 0
				} else {
					q = valueQ
				}
			}
		}
		switch name {
		case "gzip":
			gzip = q > 0
			explicit = true
		case "identity":
			identity = q > 0
			identityExplicit = true
		case "*":
			wildcard = q
		}
	}
	if !explicit && wildcard >= 0 {
		gzip = wildcard > 0
	}
	if !identityExplicit && wildcard == 0 {
		identity = false
	}
	return
}
