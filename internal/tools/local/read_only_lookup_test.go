//go:build linux || darwin

// Purpose: Preserve command aliases when trusted query executables use multicall dispatch.
// Pattern: Native executable lookup security regression test.
// Usage: go test ./internal/tools/local -run TestReadOnlyLookup.
// Related: internal/tools/local/read_only_command.go, internal/policy/readonly/parse.go.
package local

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/policy/readonly"
)

func TestReadOnlyLookupPreservesMulticallAlias(t *testing.T) {
	directory := t.TempDir()
	trusted, err := filepath.EvalSymlinks(directory)
	if err != nil {
		t.Fatal(err)
	}
	multicall, alias := filepath.Join(directory, "multicall"), filepath.Join(directory, "cat")
	// Resolving the alias and invoking the target basename changes the selected
	// applet. The alternate branch models a mutating applet selected by operands.
	script := "#!/bin/sh\nif [ \"${0##*/}\" = cat ]; then printf 'read:%s\\n' \"$*\"; else printf mutated > \"$2\"; fi\n"
	if err := os.WriteFile(multicall, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(multicall, alias); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(directory, "sentinel")
	if err := os.WriteFile(sentinel, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	query, err := readonly.Parse("cat rm '" + sentinel + "'")
	if err != nil {
		t.Fatal(err)
	}
	command, err := query.Render(func(name string) (string, error) {
		return lookupReadOnlyExecutable(name, []string{directory}, []string{trusted})
	})
	if err != nil {
		t.Fatal(err)
	}
	run := exec.Command("/bin/sh", "-c", command)
	run.Env = []string{"PATH=/usr/bin:/bin"}
	output, err := run.CombinedOutput()
	if err != nil || !strings.HasPrefix(string(output), "read:rm ") {
		t.Fatalf("command alias changed behavior: %q %v", output, err)
	}
	content, err := os.ReadFile(sentinel)
	if err != nil || string(content) != "original" {
		t.Fatalf("multicall target selected a mutating applet: %q %v", content, err)
	}
}

func TestReadOnlyLookupRejectsUntrustedAliasTarget(t *testing.T) {
	directory, outside := t.TempDir(), t.TempDir()
	trusted, err := filepath.EvalSymlinks(directory)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(outside, "cat")
	if err := os.WriteFile(target, []byte("#!/bin/sh\nprintf untrusted"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(directory, "cat")); err != nil {
		t.Fatal(err)
	}
	if _, err := lookupReadOnlyExecutable("cat", []string{directory}, []string{trusted}); err == nil {
		t.Fatal("accepted an alias escaping the trusted executable tree")
	}
}
