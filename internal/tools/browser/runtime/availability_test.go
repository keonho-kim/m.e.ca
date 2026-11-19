// Purpose: Verify dependency discovery uses published files and rejects missing runtimes without downloads.
// Pattern: Isolated filesystem and deterministic managed-install transport.
// Usage: go test ./internal/tools/browser/runtime -run 'TestBrowserAvailability|TestManagedBrowserAvailability'.
// Related: internal/tools/browser/runtime/availability.go, internal/tools/browser/runtime/managed.go.
package runtime

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func TestBrowserAvailabilityRequiresHelperAndBrowser(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")
	chrome := filepath.Join(t.TempDir(), "chrome")
	if err := os.WriteFile(chrome, []byte("executable fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MECA_BROWSER_EXECUTABLE", chrome)
	if Available(state) {
		t.Fatal("browser without its helper was advertised")
	}
	if _, err := OpenInstalled(t.Context(), state); fault.Code(err) != string(diag.FailedPrecondition) {
		t.Fatalf("opening without the helper should report missing dependencies: %v", err)
	}
	if _, err := os.Lstat(state); !os.IsNotExist(err) {
		t.Fatalf("the availability check created installation state: %v", err)
	}
	a, err := hostAgent()
	if err != nil {
		t.Skipf("unsupported helper platform: %v", err)
	}
	installed := filepath.Join(runtimeDirectory(state), a.name)
	if err := os.MkdirAll(installed, 0700); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(installed, a.executable)
	if err := os.WriteFile(binary, []byte("executable fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	if Available(state) {
		t.Fatal("an unpublished helper was advertised")
	}
	if err := os.WriteFile(filepath.Join(installed, "verified.sha256"), []byte(a.digest), 0600); err != nil {
		t.Fatal(err)
	}
	if !Available(state) {
		t.Fatal("installed dependencies were not advertised")
	}
	if _, err := OpenInstalled(t.Context(), state); err == nil {
		t.Fatal("opening accepted a helper with mismatched executable bytes")
	}
	if err := os.Remove(chrome); err != nil {
		t.Fatal(err)
	}
	if Available(state) {
		t.Fatal("the tool catalog retained a removed browser")
	}
}

func TestManagedBrowserAvailabilityUsesPublishedSelection(t *testing.T) {
	directory := t.TempDir()
	provider := &stableTransport{version: "157.0.8103.78"}
	client := &http.Client{Transport: provider}
	now := time.Now()
	if availableManagedChrome(directory, "linux", "arm64") {
		t.Fatal("a missing managed browser was advertised")
	}
	if _, err := managedChrome(t.Context(), client, directory, "linux", "arm64", now, false); fault.Code(err) != string(diag.FailedPrecondition) {
		t.Fatalf("opening without a managed browser should report missing dependencies: %v", err)
	}
	if provider.metadata != 0 || provider.archives != 0 {
		t.Fatal("opening attempted to provision a missing managed browser")
	}
	use, err := managedChrome(t.Context(), client, directory, "linux", "arm64", now, true)
	if err != nil {
		t.Fatal(err)
	}
	defer use.close()
	if !availableManagedChrome(directory, "linux", "arm64") {
		t.Fatal("the installed managed browser was not advertised")
	}
	if err := os.Remove(use.path); err != nil {
		t.Fatal(err)
	}
	if availableManagedChrome(directory, "linux", "arm64") {
		t.Fatal("the tool catalog retained a removed managed browser")
	}
}
