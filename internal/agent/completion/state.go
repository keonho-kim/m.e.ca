// Purpose: Retain bounded execution receipts with explicit freshness for the main agent.
// Pattern: Pure state transitions.
// Usage: The run owner observes outcomes and persists State in its checkpoint.
// Related: internal/agent/completion/freshness.go, internal/agent/run/completion_state.go.
package completion

import (
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"unicode/utf8"
)

const MaxTextBytes = 16 * 1024 * 1024
const MaxStateBytes = 256 * 1024 * 1024

type Receipt struct {
	Failure            *tool.Failure     `json:"failure,omitempty"`
	ObservedRun        string            `json:"observed_run,omitempty"`
	Revision           int               `json:"revision"`
	BeforeLaterChanges bool              `json:"before_later_changes"`
	Changed            bool              `json:"changed,omitempty"`
	ProcessStatus      string            `json:"process_status,omitempty"`
	ExitCode           *int              `json:"exit_code,omitempty"`
	Kind               string            `json:"kind,omitempty"`
	SourceCallID       string            `json:"source_call_id,omitempty"`
	Versions           map[string]string `json:"versions,omitempty"`
	Command            string            `json:"command,omitempty"`
	ID                 string            `json:"id"`
	Tool               string            `json:"tool"`
	Targets            []string          `json:"targets,omitempty"`
	Success            bool              `json:"success"`
	Stale              bool              `json:"stale"`
	Output             string            `json:"output"`
}
type State struct {
	CurrentFiles []tool.FileFact `json:"current_files,omitempty"`
	Receipts     []Receipt       `json:"receipts,omitempty"`
}

func (s *State) Observe(receipt Receipt, changed []string, unknown bool) {
	receipts := slices.Clone(s.Receipts)
	for i := range receipts {
		if receipts[i].Kind == "command" {
			if len(changed) > 0 {
				receipts[i].BeforeLaterChanges = true
			}
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
	if len(receipt.Targets) > 1 && receipt.Kind != "command" {
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
	s.observeCurrent(receipt, changed, unknown)
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
func (s State) Check() error {
	if err := (tool.Facts{Files: s.CurrentFiles}).Check(); err != nil {
		return err
	}
	body, err := json.Marshal(s)
	if err != nil || len(body) > MaxStateBytes {
		return fmt.Errorf("completion facts exceed the resource budget; execution records are retained")
	}
	for _, r := range s.Receipts {
		if r.Revision < 0 || len(r.ObservedRun) > 256 || !utf8.ValidString(r.ObservedRun) {
			return fmt.Errorf("invalid observation ordering")
		}
		switch r.Kind {
		case "", "content", "directory_exists", "command":
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
		if r.ID == "" || len(r.ID) > 256 || r.Tool == "" || len(r.Tool) > 128 || len(r.Command) > MaxTextBytes || len(r.Output) > MaxStateBytes || !utf8.ValidString(r.Output) {
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
