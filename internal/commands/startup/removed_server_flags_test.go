// Purpose: Reject removed compatible-server flags while preserving the native server option surface.
// Pattern: CLI boundary regression tests.
// Usage: go test ./internal/commands/startup -run TestRemovedServerFlags.
// Related: internal/commands/startup/server.go.
package startup

import "testing"

func TestRemovedServerFlags(t *testing.T) {
	for _,args:=range [][]string{{"--server","--openai"},{"--server","--openai=false"}} {
		if _,err:=Parse(args,t.TempDir());err==nil {t.Fatalf("removed option accepted: %v",args)}
	}
}
