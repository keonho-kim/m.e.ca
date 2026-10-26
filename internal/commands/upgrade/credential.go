// Purpose: Resolve GitHub download authentication without persisting or printing credentials.
// Pattern: Environment and process boundary adapter.
// Usage: The upgrade CLI resolves one token before constructing its release client.
// Related: cmd/meca/upgrade.go, internal/commands/upgrade/request.go.
package upgrade

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

func Credential(ctx context.Context) string {
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if token := strings.TrimSpace(os.Getenv(name)); token != "" {
			return token
		}
	}
	path, err := exec.LookPath("gh")
	if err != nil {
		return ""
	}
	lookup, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	command := exec.CommandContext(lookup, path, "auth", "token", "--hostname", "github.com")
	command.WaitDelay = time.Second
	output, err := command.Output()
	if err != nil || len(output) > maxInfoBytes {
		return ""
	}
	return strings.TrimSpace(string(output))
}
