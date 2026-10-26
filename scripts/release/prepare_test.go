// Purpose: Verify explicit major/minor parsing, Korean release dates and local-only tag preparation.
// Pattern: Offline shell boundary tests.
// Usage: go test ./scripts/release -run TestPrepare.
// Related: scripts/release/prepare.sh, .release-version, scripts/release/publish_test.go.
package release

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareDailyRelease(t *testing.T) {
	for _, tc := range []struct {
		name, series, tool string
		valid              bool
	}{
		{"current", "0.0", "v2.19.0", true},
		{"explicit major and minor", "2.5", "v2.19.0", true},
		{"empty", "", "v2.19.0", false},
		{"missing minor", "1", "v2.19.0", false},
		{"leading zero", "0.01", "v2.19.0", false},
		{"full version", "0.0.20261026", "v2.19.0", false},
		{"multiline", "0.0\ninjected=true", "v2.19.0", false},
		{"missing tool pin", "0.0", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			copyReleaseScript(t, root, "prepare.sh")
			writeReleaseFile(t, root, ".release-version", tc.series+"\n")
			writeReleaseFile(t, root, "go.mod", "module fixture/meca\n\ngo 1.27.2\n")
			writeReleaseFile(t, root, "devtools/go.mod", "module fixture/tools\n\ngo 1.27.3\n\nrequire github.com/goreleaser/goreleaser/v2 "+tc.tool+"\n")
			writeReleaseFile(t, root, "commands/date", "#!/bin/sh\n[ \"$TZ\" = Asia/Seoul ] || exit 1\necho 20261026\n")
			git := func(args ...string) string {
				t.Helper()
				command := exec.Command("git", args...)
				command.Dir = root
				output, err := command.CombinedOutput()
				if err != nil {
					t.Fatalf("git: %v\n%s", err, output)
				}
				return strings.TrimSpace(string(output))
			}
			git("init", "-q")
			git("config", "user.name", "Release fixture")
			git("config", "user.email", "release@example.invalid")
			git("add", ".")
			git("commit", "-qm", "fixture")
			previous := git("rev-parse", "HEAD")
			if tc.valid {
				git("tag", "v"+tc.series+".20261026")
				writeReleaseFile(t, root, "change", "next\n")
				git("add", "change")
				git("commit", "-qm", "next")
			}
			command := exec.Command("bash", "prepare.sh")
			command.Dir = root
			command.Env = append(os.Environ(), "PATH="+filepath.Join(root, "commands")+string(os.PathListSeparator)+os.Getenv("PATH"))
			output, err := command.CombinedOutput()
			if !tc.valid {
				if err == nil || git("tag", "--list") != "" {
					t.Fatalf("invalid version changed tags: %v\n%s", err, output)
				}
				return
			}
			want := "tag=v" + tc.series + ".20261026\nprevious=" + previous + "\ngoreleaser=" + tc.tool + "\n"
			if err != nil || string(output) != want {
				t.Fatalf("prepare: %v\n%s", err, output)
			}
			if git("rev-parse", "v"+tc.series+".20261026") != git("rev-parse", "HEAD") {
				t.Fatal("local tag did not advance to the prepared commit")
			}
		})
	}
}

func copyReleaseScript(t *testing.T, root, name string) {
	t.Helper()
	body, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	writeReleaseFile(t, root, name, string(body))
}

func writeReleaseFile(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0700); err != nil {
		t.Fatal(err)
	}
}
