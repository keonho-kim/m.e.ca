// Purpose: Verify user previews stay bounded, redacted and restricted to accessible text files.
// Pattern: Filesystem boundary tests.
// Usage: go test ./internal/tools/local -run TestFilePreviewRead.
// Related: internal/fileview/preview.go.
package local

import (
	"context"
	"github.com/keonho-kim/meca/internal/policy"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestFilePreviewRead(t *testing.T) {
	e, root := testExecutor(t)
	settings := policy.Settings{Scope: policy.System}
	e.redact = func(s string) string { return strings.ReplaceAll(s, "secret", "[redacted]") }
	file := filepath.Join(root, "a")
	if err := os.WriteFile(file, []byte("secret\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := e.Files.Preview(context.Background(), settings, file)
	if err != nil || got.Content != "[redacted]\n" || got.Truncated {
		t.Fatalf("%+v %v", got, err)
	}
	private := filepath.Join(e.home, ".meca")
	if err := os.Mkdir(private, 0700); err != nil {
		t.Fatal(err)
	}
	privateFile := filepath.Join(private, "config.toml")
	if err := os.WriteFile(privateFile, []byte("private configuration"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Files.Preview(context.Background(), settings, privateFile); err == nil {
		t.Fatal("private state accessible")
	}
	if _, err = e.Files.Preview(context.Background(), settings, root); err == nil {
		t.Fatal("directory read as file")
	}
	if err = os.WriteFile(file, []byte{0, 1, 2}, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.Files.Preview(context.Background(), settings, file); err == nil {
		t.Fatal("binary shown as text")
	}
	if err = os.WriteFile(file, []byte(strings.Repeat("가", (16*1024*1024)/3+100)), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = e.Files.Preview(context.Background(), settings, file)
	if err != nil || !got.Truncated || len(got.Content) > 16*1024*1024 || !utf8.ValidString(got.Content) {
		t.Fatalf("bounded preview: %d %v", len(got.Content), err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = e.Files.Preview(ctx, settings, file); err != context.Canceled {
		t.Fatal(err)
	}
}
