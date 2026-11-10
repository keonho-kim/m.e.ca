//go:build minimum

// Purpose: Keep minimum startup usable for terminal work and reject every server entry.
// Pattern: Distribution-specific startup boundary test.
// Usage: GOFLAGS=-tags=minimum go test ./internal/commands/startup -run TestMinimum.
// Related: internal/commands/startup/server.go, cmd/meca/server_minimum.go.
package startup

import (
	"strings"
	"testing"
)

func TestMinimumRejectsServer(t *testing.T) {
	for _, args := range [][]string{{"--server"}, {"--server=false"}, {"--server", "--ui"}, {"--listen", "127.0.0.1:0"}} {
		if _, err := Parse(args, t.TempDir()); err == nil || !strings.Contains(err.Error(), "full") {
			t.Fatalf("minimum accepted server input or omitted the full distribution: %v: %v", args, err)
		}
	}
}

func TestMinimumRetainsTerminalAndCLI(t *testing.T) {
	project := t.TempDir()
	for _, args := range [][]string{nil, {"--print", "hello"}, {"--mode", "json", "hello"}} {
		mode, err := Parse(args, project)
		if err != nil || mode.Server || mode.Project == "" {
			t.Fatal(args, mode, err)
		}
	}
}
