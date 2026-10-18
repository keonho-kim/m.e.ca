// Purpose: Separate model intent metadata from executable tool arguments.
// Pattern: Native call boundary parser.
// Usage: Runtime advertises WithIntent and calls ExtractIntent before preparing tools.
// Related: internal/agent/tool/schema.go, internal/agent/grounding/record.go.
package tool

import (
	"encoding/json"
	"fmt"
)

func WithIntent(d Definition) Definition {
	d.Parameters = append(append([]Parameter(nil), d.Parameters...), Parameter{Name: "reason", Description: "Briefly explain why this action is needed, using the request or an observed result.", Kind: Text, Required: true, MaxBytes: MaxReasonBytes})
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
		return Call{}, "", fmt.Errorf("reason is required as a string")
	}
	if err := CheckReason(reason); err != nil {
		return Call{}, "", err
	}
	delete(fields, "reason")
	call.Arguments, err = json.Marshal(fields)
	return call, reason, err
}
