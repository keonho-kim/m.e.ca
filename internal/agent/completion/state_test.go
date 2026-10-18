// Purpose: Verify completion evidence expires by target and unknown command effects.
// Pattern: Pure state contract tests.
// Usage: go test ./internal/agent/completion.
// Related: internal/agent/completion/state.go.
package completion

import "testing"

func TestEvidenceValidity(t *testing.T) {
	var s State
	s.Observe(Receipt{ID: "read-a", Tool: "inspect_fixture", Targets: []string{"/p/a"}, Success: true, Output: "a"}, nil, false)
	s.Observe(Receipt{ID: "write-b", Tool: "write_file", Targets: []string{"/p/b"}, Success: true}, []string{"/p/b"}, false)
	if s.Receipts[0].Stale {
		t.Fatal("unrelated edit invalidated read")
	}
	s.Observe(Receipt{ID: "write-a", Tool: "write_file", Targets: []string{"/p/a"}, Success: true}, []string{"/p/a"}, false)
	if !s.Receipts[0].Stale {
		t.Fatal("changed source retained valid read")
	}
	s.Observe(Receipt{ID: "command", Tool: "execute_command", Success: true}, nil, true)
	if !s.Receipts[1].Stale || !s.Receipts[2].Stale {
		t.Fatal("unknown effects retained facts")
	}
}

func TestDirectoryExistenceAndContentHaveDifferentValidity(t *testing.T) {
	var s State
	s.Observe(Receipt{ID: "dirs", Tool: "create_directory", Kind: "directory_exists", Targets: []string{"/p/src", "/p/docs"}, Success: true}, nil, false)
	s.Observe(Receipt{ID: "list", Tool: "inspect_fixture", Kind: "content", Targets: []string{"/p/src"}, Success: true}, nil, false)
	s.Observe(Receipt{ID: "file", Tool: "write_file", Targets: []string{"/p/src/main.py"}, Success: true}, []string{"/p/src/main.py"}, false)
	if len(s.Receipts) != 4 || s.Receipts[0].Stale || s.Receipts[1].Stale || !s.Receipts[2].Stale {
		t.Fatalf("wrong validity: %+v", s.Receipts)
	}
	s.Observe(Receipt{ID: "delete", Tool: "execute_command", Targets: []string{"/p/src"}, Success: true}, []string{"/p/src"}, false)
	if !s.Receipts[0].Stale || s.Receipts[1].Stale {
		t.Fatal("directory facts were invalidated as one batch")
	}
}
