// Purpose: Define file Scope validation and tool effect classifications.
// Pattern: Pure Function.
// Usage: File adapters and agent execution use Scope and Effect contracts.
// Related: internal/policy/path.go, internal/policy/policy_test.go, SYSTEM.md.

package policy

import "fmt"

type Scope string

const (
	Directory Scope = "directory"
	System    Scope = "system"
)

type Settings struct {
	Scope Scope `json:"scope"`
}

func ParseScope(value string) (Scope, error) {
	switch Scope(value) {
	case Directory, System:
		return Scope(value), nil
	default:
		return "", fmt.Errorf("invalid Scope: %q", value)
	}
}

func (settings Settings) Check() error {
	_, err := ParseScope(string(settings.Scope))
	return err
}

type Effect uint8

const (
	Read Effect = iota
	Change
	Command
)
