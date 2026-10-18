// Purpose: Preserve current per-file observations independently of operation outcomes.
// Pattern: State transition regression tests.
// Usage: go test ./internal/agent/completion -run TestCurrentFiles.
// Related: internal/agent/completion/freshness.go, internal/agent/completion/current_files.go.
package completion

import (
	"testing"
)

func TestCurrentFilesKeepFailedOperationEffectsAndIndividualFreshness(t *testing.T) {
	s := State{Receipts: []Receipt{{ID: "write", Tool: "write_file", Kind: "content", Success: false, Changed: true, Versions: map[string]string{"/p/a": "v1", "/p/b": "v1"}}}}
	if len(s.FileTargets()) != 2 {
		t.Fatal("failed operation discarded applied effects")
	}
	s.ReconcileFiles(map[string]string{"/p/a": "v1", "/p/b": "v2"})
	facts := s.Facts()
	if len(facts.Files) != 2 || facts.Files[0].Stale || facts.Files[1].Stale || facts.Files[1].Version != "v2" {
		t.Fatalf("current files = %+v", facts.Files)
	}
	if !s.Receipts[0].Stale {
		t.Fatal("historical report must remain outdated")
	}
}

func TestCurrentFilesDistinguishMissingUnavailableAndUnobserved(t *testing.T) {
	s := State{Receipts: []Receipt{{ID: "read", Tool: "inspect_fixture", Versions: map[string]string{"/p/a": "v1", "/p/b": "v1"}}}}
	s.ReconcileFiles(map[string]string{"/p/a": "missing"})
	if len(s.CurrentFiles) != 2 || s.CurrentFiles[0].Observation != "missing" || s.CurrentFiles[1].Observation != "unavailable" {
		t.Fatalf("observations = %+v", s.CurrentFiles)
	}
}
