// Purpose: Separate model intent metadata from executable tool arguments.
// Pattern: Native call boundary parser.
// Usage: Runtime advertises WithIntent and calls ExtractIntent before preparing tools.
// Related: internal/agent/tool/schema.go, internal/agent/grounding/record.go.
package tool

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func WithIntent(d Definition) Definition {
	d.Parameters = append(append([]Parameter(nil), d.Parameters...), Parameter{Name: "reason", Description: "Brief reason for this operation in English; cite observed sources or the user request. One line, at most 256 characters.", Kind: Text, Required: true, MaxBytes: 1024, MaxChars: 256})
	return d
}
func ExtractIntent(call Call) (Call, string, error) {
	if err := call.Check(); err != nil {
		return Call{}, "", err
	}
	fields, err := argumentObject(call.Arguments)
	if err != nil {
		return Call{}, "", err
	}
	var reason string
	if err := json.Unmarshal(fields["reason"], &reason); err != nil {
		return Call{}, "", fmt.Errorf("reason is required as one short string")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || len(reason) > 1024 || utf8.RuneCountInString(reason) > 256 || strings.ContainsFunc(reason, unicode.IsControl) {
		return Call{}, "", fmt.Errorf("reason must be one line within 256 characters")
	}
	delete(fields, "reason")
	call.Arguments, err = json.Marshal(fields)
	return call, reason, err
}
