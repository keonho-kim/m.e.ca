// Purpose: Verify source installation bootstraps Go only when needed and cleans temporary downloads.
// Pattern: Shell boundary integration tests with offline toolchain fixtures.
// Usage: go test ./internal/commands/install -run TestSourceInstaller.
// Related: install.sh, scripts/install/go.sh, scripts/install/go.sha256.
package install

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

func TestSourceInstaller(t *testing.T) {
	for _, tc := range []struct {
		name, os, arch string
		existing       bool
		corrupt        bool
		failDownload   bool
		buildExit      string
		wantError      string
	}{
		{name: "existing Go", existing: true},
		{name: "mac arm64", os: "Darwin", arch: "arm64"},
		{name: "mac amd64", os: "Darwin", arch: "x86_64"},
		{name: "linux arm64", os: "Linux", arch: "aarch64"},
		{name: "linux amd64", os: "Linux", arch: "x86_64"},
		{name: "checksum mismatch", os: "Linux", arch: "x86_64", corrupt: true, wantError: "SHA-256"},
		{name: "download failure", os: "Linux", arch: "x86_64", failDownload: true, wantError: "download unavailable"},
		{name: "build failure", os: "Darwin", arch: "arm64", buildExit: "42", wantError: "fixture build failed"},
		{name: "unsupported OS", os: "Other", arch: "arm64", wantError: "macOS와 Linux"},
		{name: "unsupported CPU", os: "Linux", arch: "other", wantError: "arm64와 amd64"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repository, err := filepath.Abs("../../..")
			if err != nil {
				t.Fatal(err)
			}
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			source, commands := filepath.Join(root, "source with spaces"), filepath.Join(root, "commands")
			for _, directory := range []string{source, commands, filepath.Join(source, "scripts/install")} {
				if err := os.MkdirAll(directory, 0700); err != nil {
					t.Fatal(err)
				}
			}
			for _, name := range []string{"install.sh", "scripts/install/go.sh", "go.mod"} {
				body, err := os.ReadFile(filepath.Join(repository, name))
				if err != nil {
					t.Fatal(err)
				}
				writeScriptFixture(t, filepath.Join(source, name), body)
			}
			for _, command := range []string{"dirname", "mkdir", "mktemp", "rm", "awk", "tar", "gzip", "cp", "shasum"} {
				path, err := exec.LookPath(command)
				if err != nil {
					if command == "shasum" {
						command, path = "sha256sum", ""
						path, err = exec.LookPath(command)
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				if err := os.Symlink(path, filepath.Join(commands, command)); err != nil {
					t.Fatal(err)
				}
			}
			goScript := []byte("#!/bin/sh\nprintf '%s\\n' \"$PWD|$*|${GOROOT-unset}\" > \"$MECA_TEST_GO_CALL\"\nif [ -n \"${MECA_TEST_GO_EXIT:-}\" ]; then\n echo 'fixture build failed' >&2\n exit \"$MECA_TEST_GO_EXIT\"\nfi\ncp \"$MECA_TEST_INSTALLER\" bin/meca\n")
			installer := filepath.Join(root, "fixture-installer")
			writeScriptFixture(t, installer, []byte(`#!/bin/sh
set -eu
printf '%s\n' "$PWD|$*|${GOROOT-unset}" > "$MECA_TEST_INSTALL_CALL"
for sdk in "$MECA_TEST_SOURCE"/.cache/go-install.*; do
    if [ -e "$sdk" ]; then
        echo 'temporary Go remains at installation' >&2
        exit 1
    fi
done
`))
			if tc.existing {
				writeScriptFixture(t, filepath.Join(commands, "go"), goScript)
			}
			archive := goArchiveFixture(t, goScript)
			archivePath := filepath.Join(root, "go.tar.gz")
			writeScriptFixture(t, archivePath, archive)
			mod, _ := os.ReadFile(filepath.Join(source, "go.mod"))
			version := ""
			for _, line := range strings.Split(string(mod), "\n") {
				if strings.HasPrefix(line, "go ") {
					version = strings.TrimSpace(strings.TrimPrefix(line, "go "))
				}
			}
			arch := "arm64"
			if tc.arch == "x86_64" {
				arch = "amd64"
			}
			digest := fmt.Sprintf("%x", sha256.Sum256(archive))
			if tc.corrupt {
				digest = strings.Repeat("0", 64)
			}
			checksum := fmt.Sprintf("%s  go%s.%s-%s.tar.gz\n", digest, version, strings.ToLower(tc.os), arch)
			writeScriptFixture(t, filepath.Join(source, "scripts/install/go.sha256"), []byte(checksum))
			writeScriptFixture(t, filepath.Join(commands, "uname"), []byte(fmt.Sprintf("#!/bin/sh\ncase \"$1\" in -s) echo %s;; -m) echo %s;; esac\n", tc.os, tc.arch)))
			curl := "#!/bin/sh\nwhile [ $# -gt 0 ]; do\n if [ \"$1\" = -o ]; then cp \"$MECA_TEST_ARCHIVE\" \"$2\"; exit; fi\n shift\ndone\nexit 2\n"
			if tc.failDownload {
				curl = "#!/bin/sh\necho 'download unavailable' >&2\nexit 22\n"
			}
			writeScriptFixture(t, filepath.Join(commands, "curl"), []byte(curl))
			call := filepath.Join(root, "go-call")
			installCall := filepath.Join(root, "install-call")
			command := exec.Command("/bin/sh", filepath.Join(source, "install.sh"))
			command.Dir = root
			command.Env = []string{
				"HOME=" + root, "PATH=" + commands, "GOROOT=unrelated-go",
				"MECA_TEST_GO_CALL=" + call, "MECA_TEST_ARCHIVE=" + archivePath, "MECA_TEST_GO_EXIT=" + tc.buildExit,
				"MECA_TEST_INSTALLER=" + installer, "MECA_TEST_INSTALL_CALL=" + installCall, "MECA_TEST_SOURCE=" + source,
			}
			output, err := command.CombinedOutput()
			if tc.wantError == "" && err != nil || tc.wantError != "" && (err == nil || !strings.Contains(string(output), tc.wantError)) {
				t.Fatalf("installer: %v\n%s", err, output)
			}
			invocation, readErr := os.ReadFile(call)
			if tc.wantError == "" || tc.buildExit != "" {
				goroot := "unset"
				if tc.existing {
					goroot = "unrelated-go"
				}
				if readErr != nil || string(invocation) != source+"|build -trimpath -o bin/meca ./cmd/meca|"+goroot+"\n" {
					t.Fatalf("Go invocation = %q, %v", invocation, readErr)
				}
			} else if !os.IsNotExist(readErr) {
				t.Fatalf("unverified Go was run: %q, %v", invocation, readErr)
			}
			installed, readErr := os.ReadFile(installCall)
			if tc.wantError == "" {
				if readErr != nil || string(installed) != root+"|install|unrelated-go\n" {
					t.Fatalf("installer invocation = %q, %v", installed, readErr)
				}
			} else if !os.IsNotExist(readErr) {
				t.Fatalf("installer ran after failed build: %q, %v", installed, readErr)
			}
			remaining, err := filepath.Glob(filepath.Join(source, ".cache/go-install.*"))
			if err != nil || len(remaining) != 0 {
				t.Fatalf("temporary SDK remains: %v, %v", remaining, err)
			}
		})
	}
}

func writeScriptFixture(t *testing.T, path string, body []byte) {
	t.Helper()
	if err := os.WriteFile(path, body, 0700); err != nil {
		t.Fatal(err)
	}
}

func goArchiveFixture(t *testing.T, script []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	writer := tar.NewWriter(gz)
	if err := writer.WriteHeader(&tar.Header{Name: "go/bin/go", Mode: 0755, Size: int64(len(script))}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(script); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}
