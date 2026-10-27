// Purpose: Verify publication rejects stale commits, unsafe build metadata, altered artifacts and tag conflicts.
// Pattern: Offline publication adapter tests.
// Usage: go test ./scripts/release -run TestPublish.
// Related: scripts/release/publish.sh, scripts/release/prepare_test.go, .github/workflows/release.yml.
package release

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPublishRelease(t *testing.T) {
	for _, mode := range []string{"existing", "new", "extra build tags", "stale", "changed tag", "bad checksum", "missing notice", "missing build tag", "lookalike build tag", "missing build metadata", "unreadable build metadata", "immutable", "auth failure", "lease failure", "upload failure"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			copyReleaseScript(t, root, "publish.sh")
			writeReleaseFile(t, root, "docs/release-notes.md", "Fixture notes\n")
			var manifest strings.Builder
			for _, platform := range []string{"linux_amd64", "linux_arm64", "macos_amd64", "macos_arm64"} {
				name := "meca_0.0.20261026_" + platform + ".tar.gz"
				metadata := "\tbuild\t-tags=godbus_no_tcp\n"
				if mode == "extra build tags" {
					metadata = "\tbuild\t-tags=first,godbus_no_tcp,last\n"
				}
				if platform == "macos_arm64" {
					switch mode {
					case "missing build tag":
						metadata = "\tbuild\t-trimpath=true\n"
					case "lookalike build tag":
						metadata = "\tbuild\t-tags=prefix_godbus_no_tcp,godbus_no_tcp_suffix\n"
					case "missing build metadata":
						metadata = "fixture without build settings\n"
					case "unreadable build metadata":
						metadata = "invalid executable\n"
					}
				}
				body := releaseArchive(t, mode != "missing notice", metadata)
				writeReleaseFile(t, root, "dist/"+name, string(body))
				fmt.Fprintf(&manifest, "%x  %s\n", sha256.Sum256(body), name)
			}
			writeReleaseFile(t, root, "dist/checksums.txt", manifest.String())
			if mode == "bad checksum" {
				writeReleaseFile(t, root, "dist/meca_0.0.20261026_linux_amd64.tar.gz", "corrupted")
			}
			if _, err := exec.LookPath("sha256sum"); err != nil {
				writeReleaseFile(t, root, "commands/sha256sum", "#!/bin/sh\nexec shasum -a 256 \"$@\"\n")
			}
			writeReleaseFile(t, root, "commands/git", fakeReleaseGit)
			writeReleaseFile(t, root, "commands/gh", fakeReleaseGitHub)
			writeReleaseFile(t, root, "commands/go", fakeReleaseGo)
			commit, previous := strings.Repeat("1", 40), strings.Repeat("2", 40)
			main, remote := commit, previous
			if mode == "new" {
				previous, remote = "", ""
			}
			if mode == "stale" {
				main = strings.Repeat("3", 40)
			}
			if mode == "changed tag" {
				remote = strings.Repeat("4", 40)
			}
			command := exec.Command("bash", "publish.sh")
			command.Dir = root
			command.Env = append(os.Environ(),
				"PATH="+filepath.Join(root, "commands")+string(os.PathListSeparator)+os.Getenv("PATH"),
				"GITHUB_REPOSITORY=example/meca", "GITHUB_SHA="+commit, "GH_TOKEN=fixture-token",
				"GITHUB_OUTPUT="+filepath.Join(root, "output"), "MECA_RELEASE_TAG=v0.0.20261026", "MECA_PREVIOUS_TAG="+previous,
				"RELEASE_TEST_COMMIT="+commit, "RELEASE_TEST_MAIN="+main, "RELEASE_TEST_TAG="+remote,
				"RELEASE_TEST_MODE="+mode, "RELEASE_TEST_CALLS="+filepath.Join(root, "calls"))
			output, err := command.CombinedOutput()
			remaining, globErr := filepath.Glob(filepath.Join(root, ".cache/release-verify.*"))
			if globErr != nil || len(remaining) != 0 {
				t.Fatalf("release verification files remain: %v, %v", remaining, globErr)
			}
			calls, _ := os.ReadFile(filepath.Join(root, "calls"))
			status, _ := os.ReadFile(filepath.Join(root, "output"))
			success := mode == "existing" || mode == "new" || mode == "extra build tags"
			if success {
				if err != nil || string(status) != "published=true\n" || !strings.Contains(string(calls), "push --force-with-lease=refs/tags/v0.0.20261026:"+previous) {
					t.Fatalf("publish: %v\n%s\n%s", err, output, calls)
				}
				if mode == "new" && !(strings.Index(string(calls), "release create") < strings.Index(string(calls), "release upload")) {
					t.Fatal("new release was not staged before uploading")
				}
				if !strings.Contains(string(calls), "release edit") {
					t.Fatal("verified release was not published")
				}
				return
			}
			if mode == "stale" {
				if err != nil || string(status) != "published=false\n" || len(calls) != 0 {
					t.Fatal("stale commit performed publication work")
				}
				return
			}
			if err == nil || strings.Contains(string(status), "published=true") || strings.Contains(string(calls), "release edit") {
				t.Fatalf("failed publication reported success: %v\n%s\n%s", err, output, calls)
			}
			if mode != "lease failure" && mode != "upload failure" && (strings.Contains(string(calls), "push ") || strings.Contains(string(calls), "release upload")) {
				t.Fatal("preflight failure moved the remote tag or uploaded assets")
			}
			if mode == "lease failure" && strings.Contains(string(calls), "release upload") {
				t.Fatal("conflicting tag update still uploaded assets")
			}
		})
	}
}

func releaseArchive(t *testing.T, notice bool, metadata string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gz)
	for _, name := range []string{"meca", "LICENSE", "NOTICE", "THIRD_PARTY_NOTICES.md"} {
		if name == "NOTICE" && !notice {
			continue
		}
		body := []byte("fixture " + name)
		if name == "meca" {
			body = []byte(metadata)
		}
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

const fakeReleaseGit = `#!/bin/sh
if [ "$1" = -c ]; then shift 2; fi
case "$1" in
  rev-parse) echo "$RELEASE_TEST_COMMIT" ;;
  ls-remote)
    printf '%s\trefs/heads/main\n' "$RELEASE_TEST_MAIN"
    if [ -n "$RELEASE_TEST_TAG" ]; then printf '%s\trefs/tags/v0.0.20261026\n' "$RELEASE_TEST_TAG"; fi ;;
  push)
    printf '%s\n' "$*" >> "$RELEASE_TEST_CALLS"
    if [ "$RELEASE_TEST_MODE" = 'lease failure' ]; then exit 1; fi ;;
  *) exit 2 ;;
esac
`

const fakeReleaseGitHub = `#!/bin/sh
if [ "$1" = api ]; then
  case "$RELEASE_TEST_MODE" in
    new) echo 'gh: Not Found (HTTP 404)' >&2; exit 1 ;;
    'auth failure') echo 'gh: Unauthorized (HTTP 401)' >&2; exit 1 ;;
    immutable) echo true ;;
    *) echo false ;;
  esac
else
  printf '%s\n' "$*" >> "$RELEASE_TEST_CALLS"
  if [ "$2" = upload ] && [ "$RELEASE_TEST_MODE" = 'upload failure' ]; then exit 1; fi
fi
`

const fakeReleaseGo = `#!/bin/sh
[ "$1" = version ] && [ "$2" = -m ] && [ "$#" = 3 ] || exit 2
if grep -Fq 'invalid executable' "$3"; then
  echo 'fixture: unreadable Go build metadata' >&2
  exit 1
fi
cat "$3"
`
