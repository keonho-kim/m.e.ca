// Purpose: Bind a model's short intent and runtime source reference to exact execution arguments.
// Pattern: Immutable execution evidence record.
// Usage: Runtime persists Bind before tool preparation; this record is a claim, not authorization.
// Related: internal/agent/tool/intent.go, internal/agent/run/grounding.go.
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
	Reason          string `json:"reason"`
	Source          string `json:"source"`
}

func digest(call tool.Call) string {
	value := sha256.Sum256(call.Arguments)
	return hex.EncodeToString(value[:])
}
func Bind(call tool.Call, reason, source string) (Record, error) {
	r := Record{CallID: call.ID, Tool: call.Name, ArgumentsDigest: digest(call), Reason: reason, Source: source}
	if err := call.Check(); err != nil {
		return r, err
	}
	return r, r.Check()
}
func (r Record) Check() error {
	if err := (tool.Call{ID: r.CallID, Name: r.Tool, Arguments: []byte(`{}`)}).Check(); err != nil {
		return err
	}
	hash, err := hex.DecodeString(r.ArgumentsDigest)
	if err != nil || len(hash) != 32 {
		return fmt.Errorf("invalid grounding binding")
	}
	if strings.TrimSpace(r.Reason) == "" || len(r.Reason) > 1024 || utf8.RuneCountInString(r.Reason) > 256 || !utf8.ValidString(r.Reason) || strings.ContainsFunc(r.Reason, unicode.IsControl) {
		return fmt.Errorf("invalid grounding reason")
	}
	if r.Source == "" || len(r.Source) > 512 || !utf8.ValidString(r.Source) {
		return fmt.Errorf("invalid grounding source")
	}
	return nil
}
func (r Record) Matches(call tool.Call) bool {
	return r.Check() == nil && r.CallID == call.ID && r.Tool == call.Name && r.ArgumentsDigest == digest(call)
}

func (r Record) Markdown() string {
	return fmt.Sprintf("Intent for %s (%s; source %s): %s", r.CallID, r.Tool, r.Source, r.Reason)
}
