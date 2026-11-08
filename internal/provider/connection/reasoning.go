// Purpose: Validate server-advertised native reasoning choices without guessing model families.
// Pattern: Simple Module.
// Usage: Discovery, persistence and menus share these model-bound capabilities.
// Related: internal/provider/connection/models.go, internal/provider/discovery/model_settings.go.
package connection

import (
	"slices"
	"unicode"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Reasoning struct {
	DefaultOn *bool    `json:"default_on,omitempty"`
	On        bool     `json:"on"`
	Off       bool     `json:"off"`
	Levels    []string `json:"levels,omitempty"`
	Default   string   `json:"default,omitempty"`
}

// KnownDefault returns only a server declaration or an unambiguous mandatory state.
func (r Reasoning) KnownDefault() *bool {
	if r.DefaultOn != nil {
		value := *r.DefaultOn
		return &value
	}
	if r.Default != "" || r.Enabled() && !r.Off {
		value := true
		return &value
	}
	if r.Off && !r.Enabled() {
		value := false
		return &value
	}
	return nil
}

func CheckEffortName(value string) error {
	if len(value) > 64 {
		return faults.Errorf(diagcatalog.ConnectionReasoningEffortExceedsBytes)
	}
	for _, r := range value {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return faults.Errorf(diagcatalog.ConnectionInvalidReasoningEffort)
		}
	}
	return nil
}

func (r Reasoning) Check() error {
	if r.DefaultOn != nil && (*r.DefaultOn && !r.Enabled() || !*r.DefaultOn && !r.Off) {
		return faults.Errorf(diagcatalog.ConnectionReasoningDefaultIsNotAnAdvertisedState)
	}
	if len(r.Levels) > 32 {
		return faults.Errorf(diagcatalog.ConnectionTooManyReasoningLevels)
	}
	seen := map[string]bool{}
	for _, value := range r.Levels {
		if value == "" || seen[value] {
			return faults.Errorf(diagcatalog.ConnectionInvalidOrDuplicateReasoningLevel)
		}
		if err := CheckEffortName(value); err != nil {
			return err
		}
		seen[value] = true
	}
	if r.Default != "" && !slices.Contains(r.Levels, r.Default) {
		return faults.Errorf(diagcatalog.ConnectionReasoningDefaultIsNotAnAdvertisedLevel)
	}
	return nil
}

func (r Reasoning) Enabled() bool { return r.On || len(r.Levels) > 0 }
func (r Reasoning) CheckEffort(value string) error {
	if err := r.Check(); err != nil {
		return err
	}
	if value != "" && !slices.Contains(r.Levels, value) {
		return faults.Errorf(diagcatalog.ConnectionReasoningEffortIsNotAdvertisedByThis, value)
	}
	return nil
}

func (m ModelInfo) Check() error {
	if err := m.ImageInput.Check(); err != nil {
		return err
	}
	if m.ID == "" || len(m.ID) > 512 || m.ContextWindow < 0 || m.MaxInput < 0 || m.MaxInput > 1<<40 || m.MaxOutput < 0 || m.ContextWindow > 1<<40 || m.MaxOutput > 1<<40 {
		return faults.Errorf(diagcatalog.ConnectionInvalidModelLimits)
	}
	if m.Reasoning != nil {
		return m.Reasoning.Check()
	}
	return nil
}
