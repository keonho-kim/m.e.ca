// Purpose: Own Scope and Permission values and deterministic execution decisions.
// Pattern: Pure Function.
// Usage: Imported by configuration, tools and agent execution.
// Related: internal/policy/path.go, internal/policy/policy_test.go, SYSTEM.md.

package policy

import "fmt"

type Permission string
type Scope string

const (
	Approve   Permission = "approve"
	AllowAll  Permission = "allow-all"
	Directory Scope      = "directory"
	System    Scope      = "system"
)

type Settings struct {
	Permission Permission `toml:"permission" json:"permission"`
	Scope      Scope      `toml:"-" json:"scope"`
}

func ParsePermission(value string) (Permission, error) {
	switch Permission(value) {
	case Approve, AllowAll:
		return Permission(value), nil
	default:
		return "", fmt.Errorf("invalid Permission: %q", value)
	}
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
	if _, err := ParsePermission(string(settings.Permission)); err != nil {
		return err
	}
	_, err := ParseScope(string(settings.Scope))
	return err
}

type Effect uint8

const (
	Read Effect = iota
	Change
	Command
)

func Decide(settings Settings, effect Effect) (bool, error) {
	if err := settings.Check(); err != nil {
		return false, err
	}
	if effect != Read && effect != Change && effect != Command {
		return false, fmt.Errorf("unknown operation effect")
	}
	return (effect == Change || effect == Command) && settings.Permission == Approve, nil
}
