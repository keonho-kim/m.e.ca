// Purpose: Verify user previews stay bounded, redacted and restricted to accessible text files.
// Pattern: Filesystem boundary tests.
// Usage: go test ./internal/tools/local -run TestFilePreviewRead.
// Related: internal/tools/local/preview.go.
package local

import (
	"context"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFilePreviewRead(t *testing.T) {
	e, root := testExecutor(t)
	settings := policy.Settings{Permission: policy.AllowAll, Scope: policy.System}
	e.redact = func(s string) string { return strings.ReplaceAll(s, "secret", "[redacted]") }
	file := filepath.Join(root, "a")
	if err := os.WriteFile(file, []byte("secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := e.Preview(context.Background(), settings, file)
	if err != nil || got.Content != "[redacted]\n" || got.Truncated {
		t.Fatalf("%+v %v", got, err)
	}
	if _, err = e.Preview(context.Background(), settings, e.boundary.Private()); err == nil {
		t.Fatal("private state accessible")
	}
	if _, err = e.Preview(context.Background(), settings, root); err == nil {
		t.Fatal("directory read as file")
	}
	if err = os.WriteFile(file, []byte{0, 1, 2}, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Preview(context.Background(), settings, file); err == nil {
		t.Fatal("binary shown as text")
	}
	if err = os.WriteFile(file, []byte(strings.Repeat("가", (16*1024*1024)/3+100)), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = e.Preview(context.Background(), settings, file)
	if err != nil || !got.Truncated || len(got.Content) > 16*1024*1024 || !utf8.ValidString(got.Content) {
		t.Fatalf("bounded preview: %d %v", len(got.Content), err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = e.Preview(ctx, settings, file); err != context.Canceled {
		t.Fatal(err)
	}
}
