// Purpose: Bind concise evidence, intended action and verification to one exact tool call.
// Pattern: Simple Module at the model trust boundary.
// Usage: The runtime parses a self-grounding reply before preparing a PFC call.
// Related: internal/agent/run/grounding.go, internal/agent/tool/call.go, internal/agent/compact/ledger.go.
package grounding

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxBytes = 2048

type Record struct {
	CallID          string `json:"call_id"`
	Tool            string `json:"tool"`
	ArgumentsDigest string `json:"arguments_digest"`
	Evidence        string `json:"evidence"`
	Action          string `json:"action"`
	Verify          string `json:"verify"`
}

func digest(call tool.Call) string {
	value := sha256.Sum256(call.Arguments)
	return hex.EncodeToString(value[:])
}

func Parse(text string, call tool.Call) (Record, error) {
	record := Record{CallID: call.ID, Tool: call.Name, ArgumentsDigest: digest(call)}
	if err := call.Check(); err != nil {
		return record, err
	}
	if len(text) > MaxBytes || !utf8.ValidString(text) {
		return record, fmt.Errorf("self-grounding exceeds %d bytes or has invalid encoding", MaxBytes)
	}
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) != 3 {
		return record, fmt.Errorf("self-grounding requires exactly three bullets: Evidence, Action, Verify")
	}
	for i, field := range []struct {
		name  string
		value *string
	}{{"Evidence", &record.Evidence}, {"Action", &record.Action}, {"Verify", &record.Verify}} {
		prefix := "- " + field.name + ":"
		if !strings.HasPrefix(strings.TrimSpace(lines[i]), prefix) {
			return record, fmt.Errorf("missing self-grounding %s bullet", field.name)
		}
		*field.value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i]), prefix))
	}
	return record, record.Check()
}

func (r Record) Check() error {
	if err := (tool.Call{ID: r.CallID, Name: r.Tool, Arguments: []byte(`{}`)}).Check(); err != nil {
		return err
	}
	decoded, err := hex.DecodeString(r.ArgumentsDigest)
	if err != nil || len(decoded) != sha256.Size {
		return fmt.Errorf("invalid self-grounding binding")
	}
	if len(r.Evidence)+len(r.Action)+len(r.Verify) > MaxBytes {
		return fmt.Errorf("self-grounding exceeds limit")
	}
	for _, value := range []string{r.Evidence, r.Action, r.Verify} {
		if strings.TrimSpace(value) == "" || !utf8.ValidString(value) {
			return fmt.Errorf("empty or invalid self-grounding field")
		}
		for _, char := range value {
			if unicode.IsControl(char) {
				return fmt.Errorf("self-grounding fields must be single lines")
			}
		}
	}
	return nil
}

func (r Record) Matches(call tool.Call) bool {
	return r.Check() == nil && r.CallID == call.ID && r.Tool == call.Name && r.ArgumentsDigest == digest(call)
}
func (r Record) Markdown() string {
	return fmt.Sprintf("Self-grounding for %s (%s; arguments SHA-256 %s)\n- Evidence: %s\n- Action: %s\n- Verify: %s", r.CallID, r.Tool, r.ArgumentsDigest, r.Evidence, r.Action, r.Verify)
}
