// Purpose: Verify the built MECA command installs, recovers its receipt and cleanly removes its own state.
// Pattern: Temporary-home executable integration checks.
// Usage: TestHeadlessExecutable supplies the built binary and isolated home to checkInstallation.
// Related: cmd/meca/headless_test.go, cmd/meca/install.go, internal/commands/install/binary.go.
package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func checkInstallation(t *testing.T, ctx context.Context, binary, home string) {
	t.Helper()
	profile := filepath.Join(home, ".profile")
	original := "export EDITOR=vi\n"
	if err := os.WriteFile(profile, []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	run := func(program string, args ...string) string {
		t.Helper()
		command := exec.CommandContext(ctx, program, args...)
		command.Env = append(os.Environ(), "HOME="+home, "SHELL=/bin/sh", "PATH=/usr/bin:/bin")
		body, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("%v: %s %v", args, body, err)
		}
		return string(body)
	}
	if output := run(binary, "install"); !strings.Contains(output, "MECA 설치 완료:") {
		t.Fatal("wrong installation identity:", output)
	}
	installed := filepath.Join(home, ".local/bin/meca")
	if output := run(installed, "--help"); !strings.Contains(output, "Usage: meca [path]") {
		t.Fatal("installed command did not run:", output)
	}
	body, err := os.ReadFile(profile)
	if err != nil || strings.Count(string(body), "# >>> meca PATH >>>") != 1 {
		t.Fatalf("MECA PATH registration: %s %v", body, err)
	}
	// An interrupted replacement may leave a stale receipt for a genuine MECA binary.
	receiptPath := filepath.Join(home, ".meca/installation.json")
	body, err = os.ReadFile(receiptPath)
	if err != nil {
		t.Fatal(err)
	}
	var receipt map[string]json.RawMessage
	if err := json.Unmarshal(body, &receipt); err != nil {
		t.Fatal(err)
	}
	receipt["binary_sha256"] = json.RawMessage(`"` + strings.Repeat("0", 64) + `"`)
	body, err = json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, body, 0600); err != nil {
		t.Fatal(err)
	}
	run(binary, "install")
	if output := run(installed, "uninstall"); !strings.Contains(output, "MECA 삭제 완료:") {
		t.Fatal("wrong uninstallation identity:", output)
	}
	for _, path := range []string{installed, filepath.Join(home, ".meca")} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("uninstall left %s: %v", path, err)
		}
	}
	if body, err := os.ReadFile(profile); err != nil || string(body) != original {
		t.Fatalf("uninstall changed the user's shell settings: %q %v", body, err)
	}
}
