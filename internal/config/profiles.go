// Purpose: Validate exact model bindings and offline syntax for native reasoning preferences.
// Pattern: Configuration boundary.
// Usage: Model connection workers read Config.ModelPreference before setup.
// Related: internal/config/settings.go, internal/provider/profile/profile.go, internal/provider/connection/routes.go.
package config

import (
	"fmt"
	"strings"

	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/profile"
)

type ModelProfile struct {
	ReasoningEffort string                 `toml:"reasoning_effort,omitempty"`
	Provider        modelconn.ProviderKind `toml:"provider"`
	Endpoint        string                 `toml:"endpoint"`
	Model           string                 `toml:"model"`
	Profile         string                 `toml:"profile"`
}

func checkProfiles(bindings []ModelProfile) error {
	if len(bindings) > 128 {
		return fmt.Errorf("model profile mappings exceed 128 entries")
	}
	seen := map[string]bool{}
	for _, binding := range bindings {
		if strings.TrimSpace(binding.Endpoint) == "" || strings.TrimSpace(binding.Model) == "" || len(binding.Model) > 512 || strings.ContainsAny(binding.Model, "\r\n\x00") || binding.Profile == "" || len(binding.Profile) > 128 {
			return fmt.Errorf("model profile mapping requires endpoint, literal model ID and profile")
		}
		connection, err := modelconn.New(binding.Provider, binding.Endpoint)
		if err != nil {
			return fmt.Errorf("model profile endpoint: %w", err)
		}
		selected, resolveErr := profile.Resolve(binding.Model, binding.Profile)
		if resolveErr != nil {
			return resolveErr
		}
		if binding.Provider == modelconn.Ollama || binding.Provider == modelconn.LMStudio {
			err = modelconn.CheckEffortName(binding.ReasoningEffort)
		} else {
			err = selected.CheckEffort(binding.Provider, binding.ReasoningEffort)
		}
		if err != nil {
			return err
		}
		key := string(binding.Provider) + "\n" + connection.APIBase() + "\n" + binding.Model
		if seen[key] {
			return fmt.Errorf("duplicate model profile mapping for one provider endpoint and model")
		}
		seen[key] = true
	}
	return nil
}

// ModelPreference reads validated configuration. It does not substitute the wire model ID.
func (c Config) ModelPreference(connection modelconn.Connection, model string) ModelProfile {
	for _, binding := range c.ModelProfiles {
		if binding.Provider != connection.Kind || binding.Model != model {
			continue
		}
		declared, err := modelconn.New(binding.Provider, binding.Endpoint)
		if err == nil && declared.APIBase() == connection.APIBase() {
			return binding
		}
	}
	return ModelProfile{}
}
