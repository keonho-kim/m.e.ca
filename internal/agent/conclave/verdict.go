// Purpose: Validate a single structured candidate review and its evidence references.
// Pattern: Review trust-boundary parser.
// Usage: Review parses model output before any approval can proceed.
// Related: internal/agent/conclave/review.go, internal/agent/tool/candidate.go.
package conclave

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/reviewjson"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"io"
	"strings"
)

type Finding struct {
	Path   string `json:"path"`
	Side   string `json:"side"`
	Line   int    `json:"line"`
	Reason string `json:"reason"`
}
type ContextRequest struct {
	Path   string `json:"path"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
	Reason string `json:"reason"`
}
type Verdict struct {
	Decision        string           `json:"decision"`
	Findings        []Finding        `json:"findings,omitempty"`
	ContextRequests []ContextRequest `json:"context_requests,omitempty"`
}
type Result struct {
	CallID, CandidateDigest  string
	Lines, Attempts          int
	Complete, Passed, Cached bool
	Verdict                  Verdict
	Error                    string
}

func Parse(text string) (Verdict, error) {
	normalized, err := reviewjson.Object(text)
	if err != nil {
		return Verdict{}, err
	}
	text = normalized
	d := json.NewDecoder(strings.NewReader(text))
	d.DisallowUnknownFields()
	first, err := d.Token()
	if err != nil || first != json.Delim('{') {
		return Verdict{}, fmt.Errorf("review must be one JSON object")
	}
	var v Verdict
	seen := map[string]bool{}
	for d.More() {
		token, err := d.Token()
		if err != nil {
			return v, err
		}
		key, ok := token.(string)
		if !ok || seen[key] {
			return v, fmt.Errorf("invalid or duplicate review field")
		}
		seen[key] = true
		switch key {
		case "decision":
			err = d.Decode(&v.Decision)
		case "findings":
			err = d.Decode(&v.Findings)
		case "context_requests":
			err = d.Decode(&v.ContextRequests)
		default:
			return v, fmt.Errorf("unknown review field %s", key)
		}
		if err != nil {
			return v, err
		}
	}
	if _, err = d.Token(); err != nil {
		return v, err
	}
	if _, err = d.Token(); err != io.EOF {
		return v, fmt.Errorf("trailing review content")
	}
	return v, v.Check()
}
func (v Verdict) Check() error {
	if len(v.Findings) > 3 || len(v.ContextRequests) > 2 {
		return fmt.Errorf("review requests exceed bounds")
	}
	for _, f := range v.Findings {
		if tool.CheckReason(f.Reason) != nil || f.Path == "" || len(f.Path) > 4096 || f.Line < 0 || f.Side != "before" && f.Side != "after" && f.Side != "request" {
			return fmt.Errorf("finding requires a bounded reason and evidence reference")
		}
	}
	for _, r := range v.ContextRequests {
		if r.Path == "" || len(r.Path) > 4096 || r.Offset < 0 || r.Limit < 1 || r.Limit > 8192 || tool.CheckReason(r.Reason) != nil {
			return fmt.Errorf("invalid review context request")
		}
	}
	switch v.Decision {
	case "pass":
		if len(v.Findings) != 0 || len(v.ContextRequests) != 0 {
			return fmt.Errorf("pass must have no unresolved findings")
		}
	case "reject":
		if len(v.Findings) == 0 || len(v.ContextRequests) != 0 {
			return fmt.Errorf("reject requires findings only")
		}
	case "needs_context":
		if len(v.ContextRequests) == 0 || len(v.Findings) != 0 {
			return fmt.Errorf("needs_context requires scoped read requests only")
		}
	default:
		return fmt.Errorf("decision must be pass, reject or needs_context")
	}
	return nil
}
func (r Result) Check() error {
	digest, err := hex.DecodeString(r.CandidateDigest)
	if err != nil || len(digest) != 32 || r.CallID == "" || len(r.CallID) > 128 || r.Lines < 1 || r.Attempts < 0 || r.Attempts > 2 || len(r.Error) > tool.MaxReasonBytes {
		return fmt.Errorf("invalid review binding or attempts")
	}
	if r.Error != "" {
		if r.Passed || r.Complete {
			return fmt.Errorf("failed review cannot pass")
		}
		return nil
	}
	if err := r.Verdict.Check(); err != nil {
		return err
	}
	if !r.Complete || r.Passed != (r.Verdict.Decision == "pass") {
		return fmt.Errorf("inconsistent review result")
	}
	return nil
}
func (r Result) Feedback() string {
	if r.Error != "" {
		return "Review unavailable; nothing executed: " + r.Error
	}
	body, _ := json.Marshal(r.Verdict)
	return "Pre-execution review; nothing executed: " + string(body) + ". Correct the specific defect or provide new observed evidence; do not repeat the unchanged candidate."
}
