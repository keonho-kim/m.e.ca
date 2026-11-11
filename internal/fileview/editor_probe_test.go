// Purpose: Distinguish sibling writes, acknowledged saves and external changes without resending documents.
// Pattern: File-version contract tests and measured metadata fast path.
// Usage: go test ./internal/fileview -run TestEditorProbe; go test ./internal/fileview -run '^$' -bench BenchmarkEditorProbe -benchmem.
// Related: internal/fileview/editor_probe.go, internal/fileview/editor_write.go.
package fileview

import (
	"github.com/keonho-kim/meca/internal/policy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func probeReader(tb testing.TB, body string) (*Reader, string) {
	tb.Helper()
	base := tb.TempDir()
	path := filepath.Join(base, "source.txt")
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		tb.Fatal(err)
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := root.Close(); err != nil {
			tb.Error(err)
		}
	})
	reader := New(base, func(_ policy.Scope, p string) (Target, error) {
		return Target{Absolute: filepath.Join(base, p), Relative: p, Root: root}, nil
	}, func(v string) string { return v })
	return reader, base
}
func TestEditorProbeVersions(t *testing.T) {
	reader, base := probeReader(t, "before")
	ctx := t.Context()
	settings := policy.Settings{Scope: policy.Directory}
	doc, err := reader.Edit(ctx, settings, "source.txt")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "sibling.txt"), []byte("sibling"), 0600); err != nil {
		t.Fatal(err)
	}
	same, err := reader.Probe(ctx, settings, "source.txt", doc.WatchRevision)
	if err != nil || !same.Unchanged || same.Revision != "" {
		t.Fatalf("sibling changed editor: %+v %v", same, err)
	}
	saved, err := reader.Save(ctx, settings, "source.txt", "saved", doc.Revision)
	if err != nil {
		t.Fatal(err)
	}
	same, err = reader.Probe(ctx, settings, "source.txt", saved.WatchRevision)
	if err != nil || !same.Unchanged {
		t.Fatalf("own save reported external change: %+v %v", same, err)
	}
	if err := os.WriteFile(filepath.Join(base, "source.txt"), []byte("external change"), 0600); err != nil {
		t.Fatal(err)
	}
	changed, err := reader.Probe(ctx, settings, "source.txt", saved.WatchRevision)
	if err != nil || changed.Unchanged || changed.Revision == saved.Revision || changed.Revision == "" {
		t.Fatalf("missed external write: %+v %v", changed, err)
	}
	if err := os.Remove(filepath.Join(base, "source.txt")); err != nil {
		t.Fatal(err)
	}
	missing, err := reader.Probe(ctx, settings, "source.txt", changed.WatchRevision)
	if err != nil || !missing.Missing {
		t.Fatalf("missed removal: %+v %v", missing, err)
	}
}
func BenchmarkEditorProbe(b *testing.B) {
	reader, _ := probeReader(b, strings.Repeat("x", EditorBytes))
	settings := policy.Settings{Scope: policy.Directory}
	doc, err := reader.Edit(b.Context(), settings, "source.txt")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		if version, err := reader.Probe(b.Context(), settings, "source.txt", doc.WatchRevision); err != nil || !version.Unchanged {
			b.Fatal(version, err)
		}
	}
}
