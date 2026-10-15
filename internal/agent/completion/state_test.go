// Purpose: Verify completion evidence expires by target and unknown command effects.
// Pattern: Pure state contract tests.
// Usage: go test ./internal/agent/completion.
// Related: internal/agent/completion/state.go.
package completion

import "testing"

func TestEvidenceValidity(t *testing.T) {
	var s State
	s.Observe(Receipt{ID: "read-a", Tool: "read_file", Targets: []string{"/p/a"}, Success: true, Output: "a"}, nil, false)
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

func TestVerdictRequiresConcreteOriginalScope(t *testing.T) {
	input := Input{Request: "Create a", State: State{Receipts: []Receipt{{ID: "a", Success: true}}}}
	for _, body := range []string{
		`{"decision":"continue","reason":"More polish","unmet":[2],"evidence":["request"]}`,
		`{"decision":"complete","reason":"Done","evidence":["invented"]}`,
		`{"decision":"continue","reason":"Check again","evidence":["a"]}`,
		`{"decision":"complete","decision":"blocked","reason":"x","evidence":["a"]}`,
	} {
		if _, err := Parse(body, input); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	if _, err := Parse(`{"decision":"continue","reason":"Requested file absent","unmet":["Create a"],"evidence":["request"]}`, input); err != nil {
		t.Fatal(err)
	}
	input.State.Receipts[0].Stale = true
	if _, err := Parse(`{"decision":"complete","reason":"Done","evidence":["a"]}`, input); err == nil {
		t.Fatal("stale evidence accepted")
	}
}

func TestFingerprintUsesFileVersionsNotCallIDs(t *testing.T) {
	s := State{Receipts: []Receipt{{ID: "first", Tool: "write_file", Targets: []string{"a"}, Versions: map[string]string{"a": "version1"}, Success: true, Output: "4 bytes"}}}
	first := s.Fingerprint("repair a")
	s.Receipts[0].ID = "other"
	if first != s.Fingerprint("repair a") {
		t.Fatal("call ID counted as progress")
	}
	s.Receipts[0].Versions = map[string]string{"a": "version2"}
	if first == s.Fingerprint("repair a") {
		t.Fatal("equal-length edit was not new evidence")
	}
}

func TestDirectoryExistenceAndContentHaveDifferentValidity(t *testing.T) {
	var s State
	s.Observe(Receipt{ID: "dirs", Tool: "create_directory", Kind: "directory_exists", Targets: []string{"/p/src", "/p/docs"}, Success: true}, nil, false)
	s.Observe(Receipt{ID: "list", Tool: "read_file", Kind: "content", Targets: []string{"/p/src"}, Success: true}, nil, false)
	s.Observe(Receipt{ID: "file", Tool: "write_file", Targets: []string{"/p/src/main.py"}, Success: true}, []string{"/p/src/main.py"}, false)
	if len(s.Receipts) != 4 || s.Receipts[0].Stale || s.Receipts[1].Stale || !s.Receipts[2].Stale {
		t.Fatalf("wrong validity: %+v", s.Receipts)
	}
	s.Observe(Receipt{ID: "delete", Tool: "delete_file", Targets: []string{"/p/src"}, Success: true}, []string{"/p/src"}, false)
	if !s.Receipts[0].Stale || s.Receipts[1].Stale {
		t.Fatal("directory facts were invalidated as one batch")
	}
}
