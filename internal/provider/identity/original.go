// Purpose: Decode explicit repository IDs in Hugging Face cache paths without guessing filenames.
// Pattern: Pure Function.
// Usage: Model menus display a normalized server-declared original name.
// Related: internal/tui/settings/model/identity.go, internal/provider/identity/report.go.
package identity

import (
	"net/url"
	"strings"
)

func Repository(original string) string {
	if u, err := url.Parse(original); err == nil && u.Scheme == "https" && u.Host == "huggingface.co" && u.User == nil && u.RawQuery == "" && u.Fragment == "" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return parts[0] + "/" + parts[1]
		}
	}
	parts := strings.Split(original, "/")
	for i, part := range parts {
		if !strings.HasPrefix(part, "models--") || i+2 >= len(parts) || parts[i+1] != "snapshots" || parts[i+2] == "" {
			continue
		}
		repository := strings.Split(strings.TrimPrefix(part, "models--"), "--")
		if len(repository) == 2 && repository[0] != "" && repository[1] != "" {
			return repository[0] + "/" + repository[1]
		}
	}
	return original
}
