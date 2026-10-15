// Purpose: Retain bounded execution receipts with explicit freshness for completion review.
// Pattern: Pure state transitions.
// Usage: The run owner observes outcomes and persists State in its checkpoint.
// Related: internal/agent/completion/review.go, internal/agent/run/completion_state.go.
package completion

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

const maxUnmet = 16
const MaxTextBytes = 16 * 1024 * 1024
const MaxStateBytes = 256 * 1024 * 1024

type Receipt struct {
	Kind         string            `json:"kind,omitempty"`
	SourceCallID string            `json:"source_call_id,omitempty"`
	Versions     map[string]string `json:"versions,omitempty"`
	Command      string            `json:"command,omitempty"`
	ID           string            `json:"id"`
	Tool         string            `json:"tool"`
	Targets      []string          `json:"targets,omitempty"`
	Success      bool              `json:"success"`
	Stale        bool              `json:"stale"`
	Output       string            `json:"output"`
}
type State struct {
	Receipts []Receipt `json:"receipts,omitempty"`
	Reviewed string    `json:"reviewed,omitempty"`
	Verdict  *Verdict  `json:"verdict,omitempty"`
}

func (s *State) Observe(receipt Receipt, changed []string, unknown bool) {
	receipts := slices.Clone(s.Receipts)
	for i := range receipts {
		if receipts[i].Kind == "command" {
			continue
		}
		if unknown {
			receipts[i].Stale = true
			continue
		}
		for _, target := range receipts[i].Targets {
			for _, path := range changed {
				if overlaps(target, path) && (receipts[i].Kind != "directory_exists" || contains(path, target)) {
					receipts[i].Stale = true
				}
			}
		}
	}
	receipt.Targets = slices.Clone(receipt.Targets)
	receipt.Versions = maps.Clone(receipt.Versions)
	if len(receipt.Targets) > 1 {
		for i, path := range receipt.Targets {
			fact := receipt
			fact.SourceCallID = receipt.ID
			fact.ID = fmt.Sprintf("%s/%d", receipt.ID, i+1)
			fact.Targets = []string{path}
			fact.Versions = map[string]string{path: receipt.Versions[path]}
			if receipt.Kind == "directory_exists" {
				fact.Output = "Directory exists at the target path."
			}
			receipts = append(receipts, fact)
		}
		s.Receipts = receipts
	} else {
		s.Receipts = append(receipts, receipt)
	}
	// A previous judgement is historical after any new observation. The digest
	// still prevents another review of identical facts despite new call IDs.
}
func contains(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func overlaps(a, b string) bool {
	for _, pair := range [][2]string{{a, b}, {b, a}} {
		rel, err := filepath.Rel(pair[0], pair[1])
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
func (s State) Fingerprint(request string) string {
	// Compare unique semantic receipts, not execution IDs or repeat counts.
	facts := map[string]bool{}
	for _, r := range s.Receipts {
		r.ID = ""
		r.SourceCallID = ""
		body, _ := json.Marshal(r)
		facts[string(body)] = true
	}
	body, _ := json.Marshal(struct {
		Request string
		Facts   map[string]bool
	}{request, facts})
	return fmt.Sprintf("%x", sha256.Sum256(body))
}
func (s State) Check() error {
	body, err := json.Marshal(s)
	if err != nil || len(body) > MaxStateBytes {
		return fmt.Errorf("completion facts exceed the resource budget; execution records are retained")
	}
	if len(s.Reviewed) > 64 {
		return fmt.Errorf("invalid completion memory")
	}
	if v := s.Verdict; v != nil {
		if v.Decision != "complete" && v.Decision != "continue" && v.Decision != "blocked" || tool.CheckReason(v.Reason) != nil || len(v.Evidence) > len(s.Receipts)+1 || checkUnmet(v.Unmet) != nil {
			return fmt.Errorf("invalid persisted completion verdict")
		}
	}
	for _, r := range s.Receipts {
		switch r.Kind {
		case "", "content", "directory_exists", "path_missing", "command":
		default:
			return fmt.Errorf("unknown completion fact kind")
		}
		if len(r.SourceCallID) > 256 || !utf8.ValidString(r.SourceCallID) {
			return fmt.Errorf("invalid completion provenance")
		}
		for path, version := range r.Versions {
			if len(path) > 4096 || len(version) > 128 || !utf8.ValidString(path) || !utf8.ValidString(version) {
				return fmt.Errorf("invalid completion version")
			}
		}
		if r.ID == "" || len(r.ID) > 256 || r.Tool == "" || len(r.Tool) > 128 || len(r.Command) > MaxTextBytes || len(r.Output) > MaxStateBytes || !utf8.ValidString(r.Output) || len(r.Targets) > 64 || len(r.Versions) > 64 {
			return fmt.Errorf("invalid completion receipt")
		}
		for _, p := range r.Targets {
			if len(p) > 4096 || !utf8.ValidString(p) {
				return fmt.Errorf("invalid completion target")
			}
		}
	}
	return nil
}
