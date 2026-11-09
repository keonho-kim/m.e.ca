// Purpose: Measure metadata discovery separately from full skill-body reads.
// Pattern: Filesystem allocation benchmark.
// Usage: go test ./internal/skills -run '^$' -bench BenchmarkSkillListMetadata -benchmem.
// Related: internal/skills/catalog.go, internal/skills/files.go.
package skills

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func BenchmarkSkillListMetadata(b *testing.B) {
	project, home := b.TempDir(), b.TempDir()
	for i := 0; i < 20; i++ {
		name := fmt.Sprintf("review-%02d", i)
		dir := filepath.Join(project, ".agents/skills", name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			b.Fatal(err)
		}
		body := "---\nname: " + name + "\ndescription: Review source\n---\n" + strings.Repeat("Instruction text\n", 8000)
		if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0600); err != nil {
			b.Fatal(err)
		}
	}
	r := New(project, home, Options{Redact: func(s string) string { return s }, Check: func(string) error { return nil }})
	b.ReportAllocs()
	for b.Loop() {
		if _, err := r.List(context.Background(), "", 32); err != nil {
			b.Fatal(err)
		}
	}
}
