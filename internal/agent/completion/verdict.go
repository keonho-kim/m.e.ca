// Purpose: Parse a completion decision and bind its references to the original scope and current facts.
// Pattern: Strict model-output boundary.
// Usage: Review accepts only Parse-validated decisions.
// Related: internal/agent/completion/review.go, internal/agent/completion/state.go.
package completion

import (
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/reviewjson"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"io"
	"strings"
)

type Verdict struct {
	Decision string   `json:"decision"`
	Reason   string   `json:"reason"`
	Unmet    []string `json:"unmet,omitempty"`
	Evidence []string `json:"evidence"`
}

func Parse(text string, input Input) (Verdict, error) {
	var v Verdict
	normalized, err := reviewjson.Object(text)
	if err != nil {
		return v, err
	}
	text = normalized
	d := json.NewDecoder(strings.NewReader(text))
	d.DisallowUnknownFields()
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return v, fmt.Errorf("expected completion JSON object")
	}
	fields := map[string]json.RawMessage{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return v, err
		}
		name, ok := key.(string)
		if !ok {
			return v, fmt.Errorf("invalid key")
		}
		if _, exists := fields[name]; exists {
			return v, fmt.Errorf("duplicate completion key")
		}
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return v, err
		}
		fields[name] = raw
	}
	if _, err := d.Token(); err != nil {
		return v, err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return v, fmt.Errorf("trailing completion output")
	}
	body, _ := json.Marshal(fields)
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&v); err != nil {
		return v, err
	}
	if err := tool.CheckReason(v.Reason); err != nil {
		return v, fmt.Errorf("reason: %w", err)
	}
	if len(v.Evidence) == 0 {
		return v, fmt.Errorf("missing evidence: include exact receipt IDs, or request for an unmet requirement; complete also requires evidence")
	}
	if len(v.Evidence) > len(input.State.Receipts)+1 {
		return v, fmt.Errorf("too many evidence references")
	}
	if v.Decision != "complete" && v.Decision != "continue" && v.Decision != "blocked" {
		return v, fmt.Errorf("invalid completion decision")
	}
	if v.Decision == "continue" && len(v.Unmet) == 0 || v.Decision == "complete" && len(v.Unmet) != 0 {
		return v, fmt.Errorf("continue needs unmet requested outcomes; complete cannot have unmet requested outcomes")
	}
	if err := checkUnmet(v.Unmet); err != nil {
		return v, err
	}
	if input.PendingReview && v.Decision == "complete" {
		return v, fmt.Errorf("requested changes remain unapplied")
	}
	usable := false
	for _, id := range v.Evidence {
		if id == "request" {
			continue
		}
		found := false
		for _, r := range input.State.Receipts {
			if r.ID != id {
				continue
			}
			found = true
			if v.Decision == "complete" && (!r.Success || r.Stale) {
				return v, fmt.Errorf("completion cites failed or stale evidence")
			}
			usable = usable || r.Success && !r.Stale
		}
		if !found {
			return v, fmt.Errorf("completion cites an unknown receipt")
		}
	}
	if v.Decision == "complete" && !usable {
		return v, fmt.Errorf("completion lacks a current successful receipt")
	}
	return v, nil
}

// Unmet outcomes remain model claims; the reviewer compares them with the original request.
func checkUnmet(outcomes []string) error {
	if len(outcomes) > maxUnmet {
		return fmt.Errorf("too many unmet outcomes")
	}
	seen := map[string]bool{}
	for _, outcome := range outcomes {
		text := strings.TrimSpace(outcome)
		if tool.CheckReason(outcome) != nil || seen[text] {
			return fmt.Errorf("unmet outcomes must be distinct nonempty descriptions of requested work")
		}
		seen[text] = true
	}
	return nil
}
