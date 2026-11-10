// Purpose: Verify platform selection, complete-download checks and retries before any installation runs.
// Pattern: HTTPS integration and format contract tests.
// Usage: go test ./internal/commands/upgrade -run 'Test(Fetch|ArchiveName|ArchiveChecksum)'.
// Related: internal/commands/upgrade/release.go, internal/commands/upgrade/request.go, internal/commands/upgrade/fixtures_test.go.
package upgrade

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/distribution"
)

func TestFetchVerifiesAndRetriesPartialDownloads(t *testing.T) {
	archive := tarball(t, archiveEntry{name: "meca", content: "complete executable"})
	var requests atomic.Int32
	client := releaseFixture(t, distribution.Name, archive, nil, func(writer http.ResponseWriter, request *http.Request) bool {
		if request.URL.Path == "/releases/assets/2" && requests.Add(1) == 1 {
			writer.Header().Set("Content-Length", fmt.Sprint(len(archive)))
			_, _ = writer.Write(archive[:len(archive)/2])
			return true
		}
		return false
	})
	root := temporaryRoot(t)
	version, err := client.fetch(context.Background(), root, distribution.Name)
	if err != nil || version != "v0.0.20261026" || requests.Load() != 2 {
		t.Fatalf("fetch = %q, %v; download attempts %d", version, err, requests.Load())
	}
	file, err := root.Open("meca")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	body, err := io.ReadAll(file)
	if err != nil || string(body) != "complete executable" {
		t.Fatalf("extracted executable = %q, %v", body, err)
	}
	file, err = root.Open("release.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	body, err = io.ReadAll(file)
	if err != nil || !bytes.Equal(body, archive) {
		t.Fatal("retried archive was appended to the partial response")
	}
}

func TestArchiveName(t *testing.T) {
	for _, platform := range []struct{ system, architecture, suffix string }{
		{"darwin", "arm64", "macos_arm64"}, {"darwin", "amd64", "macos_amd64"},
		{"linux", "arm64", "linux_arm64"}, {"linux", "amd64", "linux_amd64"},
	} {
		for _, kind := range []distribution.Kind{distribution.TUI, distribution.Full} {
			got, err := archiveName("v0.0.20261026", kind, platform.system, platform.architecture)
			if err != nil || got != "meca_0.0.20261026_"+string(kind)+"_"+platform.suffix+".tar.gz" {
				t.Fatalf("platform archive = %q, %v", got, err)
			}
		}
	}
	for _, version := range []string{"v0.0.0-beta", "../../meca", "v0.0.20260230", "v00.0.20261026"} {
		if _, err := archiveName(version, distribution.Name, "linux", "amd64"); err == nil {
			t.Fatalf("accepted unsupported release %q", version)
		}
	}
	if _, err := archiveName("v0.0.20261026", distribution.Name, "windows", "amd64"); err == nil {
		t.Fatal("accepted an unsupported operating system")
	}
}

func TestArchiveChecksum(t *testing.T) {
	digest := strings.Repeat("a", 64)
	for _, manifest := range []string{"", digest + "  another.tar.gz", "xyz  meca.tar.gz", digest + "  meca.tar.gz\n" + digest + "  meca.tar.gz"} {
		if _, err := archiveChecksum(manifest, "meca.tar.gz"); err == nil {
			t.Fatalf("accepted invalid manifest %q", manifest)
		}
	}
	if got, err := archiveChecksum(strings.ToUpper(digest)+"  meca.tar.gz\n", "meca.tar.gz"); err != nil || got != digest {
		t.Fatalf("valid checksum = %q, %v", got, err)
	}
}

func TestFetchSelectsRequestedDistribution(t *testing.T) {
	for _, kind := range []distribution.Kind{distribution.TUI, distribution.Full} {
		t.Run(string(kind), func(t *testing.T) {
			archive := tarball(t, archiveEntry{name: "meca", content: string(kind)})
			other := distribution.TUI
			if kind == other {
				other = distribution.Full
			}
			client := releaseFixture(t, kind, archive, func(latest *release, manifest *string) {
				wrong := strings.Replace(latest.Assets[1].Name, "_"+string(kind)+"_", "_"+string(other)+"_", 1)
				latest.Assets = append(latest.Assets, asset{ID: 3, Name: wrong, Size: int64(len(archive))})
				*manifest += fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), wrong)
				latest.Assets[0].Size = int64(len(*manifest))
			}, func(w http.ResponseWriter, r *http.Request) bool {
				if r.URL.Path != "/releases/assets/3" {
					return false
				}
				t.Error("updater selected the other distribution")
				w.WriteHeader(http.StatusConflict)
				return true
			})
			root := temporaryRoot(t)
			if _, err := client.fetch(t.Context(), root, kind); err != nil {
				t.Fatal(err)
			}
			body, err := os.ReadFile(filepath.Join(root.Name(), "meca"))
			if err != nil || string(body) != string(kind) {
				t.Fatal(string(body), err)
			}
		})
	}
}

func TestFetchRejectsUnknownDistributionBeforeNetwork(t *testing.T) {
	client := releaseFixture(t, distribution.Name, nil, nil, func(http.ResponseWriter, *http.Request) bool {
		t.Error("invalid distribution reached the network")
		return false
	})
	for _, kind := range []distribution.Kind{"", "minimum", "../full"} {
		if _, err := client.fetch(t.Context(), temporaryRoot(t), kind); err == nil {
			t.Fatalf("accepted unsupported distribution %q", kind)
		}
		if _, err := archiveName("v0.0.20261026", kind, "linux", "amd64"); err == nil {
			t.Fatalf("named unsupported distribution %q", kind)
		}
	}
}
