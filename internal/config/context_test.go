// Purpose: Verify global context length defaults, persistence and input bounds.
// Pattern: Configuration contract tests.
// Usage: go test ./internal/config -run TestContextLength.
// Related: internal/config/settings.go, internal/provider/connection/context.go.
package config

import "testing"

func TestContextLengthConfiguration(t *testing.T) {
	c := Defaults()
	if c.ContextLength != 262144 {
		t.Fatal("wrong context default")
	}
	c.ContextLength = 131072
	dir := t.TempDir()
	if err := Save(dir, c); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir)
	if err != nil || loaded.ContextLength != 131072 {
		t.Fatalf("context=%d err=%v", loaded.ContextLength, err)
	}
	for _, invalid := range []int{0, -1, 1<<30 + 1} {
		c.ContextLength = invalid
		if err := Save(dir, c); err == nil {
			t.Fatalf("accepted %d", invalid)
		}
	}
}
