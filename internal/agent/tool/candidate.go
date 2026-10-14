// Purpose: Describe an immutable prepared file-change candidate bound to one PFC call.
// Pattern: Tool trust-boundary value.
// Usage: Preparation supplies Candidate through Approval; Conclave reviews it before approval.
// Related: internal/tools/local/review.go, internal/agent/conclave/review.go.
package tool

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type FileDelta struct {
	Path         string `json:"path"`
	BeforeDigest string `json:"before_digest"`
	AfterDigest  string `json:"after_digest"`
	Added        int    `json:"added"`
	Deleted      int    `json:"deleted"`
	Diff         string `json:"diff"`
}
type Candidate struct {
	CallID  string      `json:"call_id"`
	Binding string      `json:"binding"`
	Files   []FileDelta `json:"files"`
}

func BindCandidate(call Call, files []FileDelta) Candidate {
	return Candidate{CallID: call.ID, Binding: fmt.Sprintf("%x", sha256.Sum256(append([]byte(call.Name+"\n"), call.Arguments...))), Files: files}
}
func (c Candidate) Check(call Call) error {
	if err := call.Check(); err != nil {
		return err
	}
	expected := BindCandidate(call, nil)
	if c.CallID != expected.CallID || c.Binding != expected.Binding || len(c.Files) > 64 {
		return fmt.Errorf("candidate does not match the frozen PFC call")
	}
	for _, file := range c.Files {
		if file.Path == "" || file.Added < 0 || file.Deleted < 0 || file.Added > 10000000 || file.Deleted > 10000000 || len(file.Diff) > 8*1024*1024 {
			return fmt.Errorf("invalid prepared file delta")
		}
	}
	return nil
}
func (c Candidate) Lines() int {
	total := 0
	for _, file := range c.Files {
		total += file.Added + file.Deleted
	}
	return total
}
func (c Candidate) Digest() (string, error) {
	body, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(body)), nil
}
