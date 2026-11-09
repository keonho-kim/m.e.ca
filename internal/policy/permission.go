// Purpose: Define the Default and All Mighty approval mode identities.
// Pattern: Value Object.
// Usage: Global TOML and each request's policy settings share Permission.
// Related: internal/config/settings.go, internal/agent/run/prepare_execution.go.
package policy

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Permission uint8

const (
	PermissionDefault Permission = iota
	PermissionAllMighty
)

func (p Permission) Check() error {
	if p != PermissionDefault && p != PermissionAllMighty {
		return faults.Errorf(diagcatalog.PolicyInvalidPermissionMode, p)
	}
	return nil
}

func (p Permission) MarshalText() ([]byte, error) {
	if err := p.Check(); err != nil {
		return nil, err
	}
	if p == PermissionAllMighty {
		return []byte("all_mighty"), nil
	}
	return []byte("default"), nil
}

func (p *Permission) UnmarshalText(text []byte) error {
	switch string(text) {
	case "default":
		*p = PermissionDefault
	case "all_mighty":
		*p = PermissionAllMighty
	default:
		return faults.Errorf(diagcatalog.PolicyInvalidPermissionMode2, text)
	}
	return nil
}
