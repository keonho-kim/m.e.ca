//go:build linux || darwin

// Purpose: Exercise installed language tools, child execution and HTTP through the host shell.
// Pattern: Native host-process contract tests.
// Usage: go test ./internal/tools/local/process -run TestHostTools.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/process/host_unix_test.go.
package process

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHostTools(t *testing.T) {
	project := t.TempDir()
	cache := t.TempDir()
	t.Setenv("GOCACHE", cache)
	t.Setenv("HOST_BOUNDARY_TEST", "preserved")
	if err := os.WriteFile(filepath.Join(project, "go.mod"), []byte("module local.test/example\n\ngo 1.27\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "main_test.go"), []byte("package example\nimport \"testing\"\nfunc TestValue(t *testing.T){if 1+1!=2{t.Fatal(\"value\")}}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "network-ok") }))
	defer server.Close()
	for _, tc := range []struct{ binary, command, expected string }{
		{"go", "go test ./...", "ok"},
		{"node", `node -e 'const c=require("child_process"); const p=c.spawnSync(process.execPath,["-e","process.stdout.write(process.env.HOST_BOUNDARY_TEST)"]); process.stdout.write(p.stdout); process.exit(p.status)'`, "preserved"},
		{"python3", `python3 -c 'import os,subprocess,sys,asyncio; asyncio.run(asyncio.sleep(0)); subprocess.run([sys.executable,"-c","import os; print(os.environ[\"HOST_BOUNDARY_TEST\"])"] ,check=True)'`, "preserved"},
		{"curl", "curl -fsS " + server.URL, "network-ok"},
	} {
		t.Run(tc.binary, func(t *testing.T) {
			if _, err := exec.LookPath(tc.binary); err != nil {
				t.Skip("installed tool unavailable")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			output, err := Run(ctx, project, tc.command, os.Environ())
			if err != nil || output.ExitCode != 0 || !strings.Contains(output.Stdout, tc.expected) {
				t.Fatalf("%+v %v", output, err)
			}
		})
	}
}
