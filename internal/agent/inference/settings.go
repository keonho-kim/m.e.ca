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
	Temperature     float64 `toml:"temperature"`
	MaxOutputTokens int     `toml:"max_output_tokens"`
}
type Settings struct {
	Agent      Parameters `toml:"agent"`
	Plan       Parameters `toml:"plan"`
	Goal       Parameters `toml:"goal"`
	PFCValue   Parameters `toml:"pfc_value"`
	PFCContent Parameters `toml:"pfc_content"`
	Grounding  Parameters `toml:"grounding"`
	Conclave   Parameters `toml:"conclave"`
	Compaction Parameters `toml:"compaction"`
	Report     Parameters `toml:"report"`
	Diagnostic Parameters `toml:"diagnostic"`
}

func Defaults() Settings {
	return Settings{
		Agent:      Parameters{Temperature: 0.7, MaxOutputTokens: 2048},
		Plan:       Parameters{Temperature: 0.7, MaxOutputTokens: 2048},
		Goal:       Parameters{Temperature: 0.2, MaxOutputTokens: 2048},
		PFCValue:   Parameters{Temperature: 0.2, MaxOutputTokens: 2048},
		PFCContent: Parameters{Temperature: 0.7, MaxOutputTokens: 8192},
		Grounding:  Parameters{Temperature: 0.2, MaxOutputTokens: 2048},
		Conclave:   Parameters{Temperature: 0.4, MaxOutputTokens: 2048},
		Compaction: Parameters{Temperature: 0.2, MaxOutputTokens: 4096},
		Report:     Parameters{Temperature: 0.2, MaxOutputTokens: 2048},
		Diagnostic: Parameters{Temperature: 0.2, MaxOutputTokens: 512},
	}
}
func (p Parameters) Check() error {
	if err := modelapi.CheckTemperature(&p.Temperature); err != nil {
		return err
	}
	if p.MaxOutputTokens < 1 || p.MaxOutputTokens > 65536 {
		return fmt.Errorf("max_output_tokens must be between 1 and 65536")
	}
	return nil
}
func (s Settings) Check() error {
	for _, role := range []Role{Agent, Plan, Goal, PFCValue, PFCContent, Grounding, Conclave, Compaction, Report, Diagnostic} {
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
	case Goal:
		p = s.Goal
	case PFCValue:
		p = s.PFCValue
	case PFCContent:
		p = s.PFCContent
	case Grounding:
		p = s.Grounding
	case Conclave:
		p = s.Conclave
	case Compaction:
		p = s.Compaction
	case Report:
		p = s.Report
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
