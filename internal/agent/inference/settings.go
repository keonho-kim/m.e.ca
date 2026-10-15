// Purpose: Own role-specific sampling and output budgets independently of providers.
// Pattern: Simple Module.
// Usage: Configuration stores Settings; each model call selects its role parameters.
// Related: internal/agent/inference/role.go, internal/agent/model/temperature.go.
package inference

import (
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

type Parameters struct {
	Thinking        string  `toml:"thinking"`
	Temperature     float64 `toml:"temperature"`
	MaxOutputTokens int     `toml:"max_output_tokens"`
}
type Settings struct {
	Agent      Parameters `toml:"agent"`
	Plan       Parameters `toml:"plan"`
	Conclave   Parameters `toml:"conclave"`
	Compaction Parameters `toml:"compaction"`
	Diagnostic Parameters `toml:"diagnostic"`
}

func Defaults() Settings {
	return Settings{
		Agent:      Parameters{Thinking: "inherit", Temperature: 0.7, MaxOutputTokens: 0},
		Plan:       Parameters{Thinking: "inherit", Temperature: 0.7, MaxOutputTokens: 12800},
		Conclave:   Parameters{Thinking: "off_if_supported", Temperature: 0.4, MaxOutputTokens: 12800},
		Compaction: Parameters{Thinking: "off_if_supported", Temperature: 0.2, MaxOutputTokens: 8192},
		Diagnostic: Parameters{Thinking: "inherit", Temperature: 0.2, MaxOutputTokens: 12800},
	}
}
func (p Parameters) Check() error {
	if p.Thinking != "" && p.Thinking != "inherit" && p.Thinking != "off_if_supported" {
		return fmt.Errorf("thinking must be inherit or off_if_supported")
	}
	if err := modelapi.CheckTemperature(&p.Temperature); err != nil {
		return err
	}
	if err := modelapi.CheckOutputBudget(p.MaxOutputTokens); err != nil {
		return err
	}
	return nil
}
func (s Settings) Check() error {
	for _, role := range []Role{Agent, Plan, Conclave, Compaction, Diagnostic} {
		if _, err := Resolve(&s, role); err != nil {
			return err
		}
	}
	return nil
}
func Resolve(settings *Settings, role Role) (Parameters, error) {
	s := Defaults()
	if settings != nil {
		s = *settings
	}
	var p Parameters
	switch role {
	case Agent:
		p = s.Agent
	case Plan:
		p = s.Plan
	case Conclave:
		p = s.Conclave
	case Compaction:
		p = s.Compaction
	case Diagnostic:
		p = s.Diagnostic
	default:
		return p, fmt.Errorf("unknown inference role %q", role)
	}
	if err := p.Check(); err != nil {
		return p, fmt.Errorf("inference.%s: %w", role, err)
	}
	return p, nil
}

// Apply returns a request-local snapshot. Parallel calls never mutate configured defaults.
func (p Parameters) Apply(request modelapi.Request) modelapi.Request {
	request.Temperature = &p.Temperature
	request.MaxOutput = p.MaxOutputTokens
	return request
}

// ApplyThinking preserves the selected model's supported controls.
func (p Parameters) ApplyThinking(request modelapi.Request, supportsOff bool) modelapi.Request {
	if p.Thinking == "off_if_supported" && supportsOff {
		off := false
		request.Thinking = &off
		request.ReasoningEffort = ""
	}
	return request
}
