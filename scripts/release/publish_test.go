// Purpose: Verify release publication, draft recovery and preflight failures.
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
	for _, mode := range []string{"existing", "new", "resume draft", "extra build tags", "stale", "changed tag", "bad checksum", "missing notice", "missing UI notice", "tui UI notice", "missing build tag", "lookalike build tag", "missing variant tag", "wrong variant tag", "missing build metadata", "unreadable build metadata", "immutable", "auth failure", "lease failure", "upload failure", "cleanup failure"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			copyReleaseScript(t, root, "publish.sh")
			writeReleaseFile(t, root, "docs/release-notes.md", "Fixture notes\n")
			var manifest strings.Builder
			for _, variant := range []string{"tui", "full"} {
				for _, platform := range []string{"linux_amd64", "linux_arm64", "macos_amd64", "macos_arm64"} {
					name := "meca_0.0.20261026_" + variant + "_" + platform + ".tar.gz"
					metadata := "\tbuild\t-tags=godbus_no_tcp\n"
					if variant == "tui" {
						metadata = "\tbuild\t-tags=godbus_no_tcp,tui\n"
					}
					if mode == "extra build tags" {
						metadata = "\tbuild\t-tags=first,godbus_no_tcp,last\n"
						if variant == "tui" {
							metadata = "\tbuild\t-tags=first,godbus_no_tcp,tui,last\n"
						}
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
						case "missing variant tag":
							if variant == "tui" {
								metadata = "\tbuild\t-tags=godbus_no_tcp\n"
							}
						case "wrong variant tag":
							if variant == "full" {
								metadata = "\tbuild\t-tags=godbus_no_tcp,tui\n"
							}
						}
					}
					uiNotice := variant == "full" && mode != "missing UI notice" || variant == "tui" && mode == "tui UI notice"
					body := releaseArchive(t, mode != "missing notice", uiNotice, metadata)
					writeReleaseFile(t, root, "dist/"+name, string(body))
					fmt.Fprintf(&manifest, "%x  %s\n", sha256.Sum256(body), name)
				}
			}
			writeReleaseFile(t, root, "dist/checksums.txt", manifest.String())
			if mode == "bad checksum" {
				writeReleaseFile(t, root, "dist/meca_0.0.20261026_tui_linux_amd64.tar.gz", "corrupted")
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
			command.Env = append(command.Env, "RELEASE_TEST_METADATA="+filepath.Join(root, "metadata-calls"))
			output, err := command.CombinedOutput()
			remaining, globErr := filepath.Glob(filepath.Join(root, ".cache/release-verify.*"))
			if globErr != nil || len(remaining) != 0 {
				t.Fatalf("release verification files remain: %v, %v", remaining, globErr)
			}
			calls, _ := os.ReadFile(filepath.Join(root, "calls"))
			status, _ := os.ReadFile(filepath.Join(root, "output"))
			success := mode == "existing" || mode == "new" || mode == "resume draft" || mode == "extra build tags"
			if success {
				metadataCalls, readErr := os.ReadFile(filepath.Join(root, "metadata-calls"))
				if readErr != nil || strings.Count(string(metadataCalls), "\n") != 8 {
					t.Fatal("archives were not verified exactly once", string(metadataCalls), readErr)
				}
				if err != nil || string(status) != "published=true\n" || !strings.Contains(string(calls), "push --force-with-lease=refs/tags/v0.0.20261026:"+previous) {
					t.Fatalf("publish: %v\n%s\n%s", err, output, calls)
				}
				create := strings.Index(string(calls), "release create")
				if mode == "new" {
					if create < 0 || create >= strings.Index(string(calls), "release upload") {
						t.Fatal("new release was not staged before uploading")
					}
				} else if create >= 0 {
					t.Fatal("existing release was not reused", string(calls))
				}
				if !strings.Contains(string(calls), "release edit") {
					t.Fatal("verified release was not published")
				}
				deleted := strings.Count(string(calls), "release delete-asset")
				wantDeleted := 8
				if mode == "new" || mode == "resume draft" {
					wantDeleted = 0
				}
				if deleted != wantDeleted {
					t.Fatal("incorrect obsolete archive cleanup", string(calls))
				}
				if deleted > 0 && strings.Index(string(calls), "release delete-asset") < strings.Index(string(calls), "release upload") {
					t.Fatal("old archives were removed before uploading both distributions")
				}
				for _, call := range strings.Split(string(calls), "\n") {
					if strings.Contains(call, "release delete-asset") && (strings.Contains(call, "_full_") || strings.Contains(call, "20261025") || strings.Contains(call, "unrelated-notes")) {
						t.Fatal("cleanup removed a current or unrelated asset", call)
					}
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
			if mode != "lease failure" && mode != "upload failure" && mode != "cleanup failure" && (strings.Contains(string(calls), "push ") || strings.Contains(string(calls), "release upload")) {
				t.Fatal("preflight failure moved the remote tag or uploaded assets")
			}
			if mode == "lease failure" && strings.Contains(string(calls), "release upload") {
				t.Fatal("conflicting tag update still uploaded assets")
			}
			if mode == "upload failure" && strings.Contains(string(calls), "release delete-asset") {
				t.Fatal("failed upload removed old archives")
			}
		})
	}
}

func releaseArchive(t *testing.T, notice, uiNotice bool, metadata string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gz)
	for _, name := range []string{"meca", "LICENSE", "NOTICE", "THIRD_PARTY_NOTICES.md", "WEBUI_THIRD_PARTY_NOTICES.txt"} {
		if name == "NOTICE" && !notice || name == "WEBUI_THIRD_PARTY_NOTICES.txt" && !uiNotice {
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
if [ "$1" = api ] && { [ "$RELEASE_TEST_MODE" = new ] || [ "$RELEASE_TEST_MODE" = 'resume draft' ]; }; then
  echo 'gh: Not Found (HTTP 404)' >&2
  exit 1
fi
if [ "$1" = api ] || [ "$1 $2" = 'release view' ]; then
  selection=$7
  if [ "$1" = api ]; then selection=$4; fi
  if [ "$selection" = '.assets[].name' ]; then
    if [ "$RELEASE_TEST_MODE" != new ] && [ "$RELEASE_TEST_MODE" != 'resume draft' ]; then
      for platform in linux_amd64 linux_arm64 macos_amd64 macos_arm64; do
        printf 'meca_0.0.20261026_%s.tar.gz\n' "$platform"
        printf 'meca_0.0.20261026_minimum_%s.tar.gz\n' "$platform"
        printf 'meca_0.0.20261026_full_%s.tar.gz\n' "$platform"
        printf 'meca_0.0.20261025_minimum_%s.tar.gz\n' "$platform"
      done
      echo unrelated-notes.txt
    fi
    exit 0
  fi
  case "$RELEASE_TEST_MODE" in
    new) echo 'release not found' >&2; exit 1 ;;
    'auth failure') echo 'gh: Unauthorized (HTTP 401)' >&2; exit 1 ;;
    immutable) echo true ;;
    *) echo false ;;
  esac
else
  printf '%s\n' "$*" >> "$RELEASE_TEST_CALLS"
  if [ "$2" = upload ] && [ "$RELEASE_TEST_MODE" = 'upload failure' ]; then exit 1; fi
  if [ "$2" = delete-asset ] && [ "$RELEASE_TEST_MODE" = 'cleanup failure' ]; then exit 1; fi
fi
`

const fakeReleaseGo = `#!/bin/sh
[ "$1" = version ] && [ "$2" = -m ] && [ "$#" = 3 ] || exit 2
printf '%s\n' "$3" >> "$RELEASE_TEST_METADATA"
if grep -Fq 'invalid executable' "$3"; then
  echo 'fixture: unreadable Go build metadata' >&2
  exit 1
fi
cat "$3"
`
