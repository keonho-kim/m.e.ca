// Purpose: Apply exact model metadata to profile selection and reject conflicting specialization.
// Pattern: Pure policy module.
// Usage: Prepare resolves identity evidence before constructing the generation client.
// Related: internal/provider/identity/report.go, internal/provider/profile/profile.go.
package setup

import (
	"fmt"
	"github.com/keonho-kim/v-code/internal/provider/identity"
	"github.com/keonho-kim/v-code/internal/provider/profile"
)

func resolveIdentity(model, override string, report *identity.Report) (profile.Profile, error) {
	report.ProfileSource = "generic"
	if override == profile.Generic {
		return profile.Resolve(model, override)
	}
	if report.State() == "ambiguous" {
		if override != "" {
			return profile.Profile{}, fmt.Errorf("model identity is ambiguous or routed; use generic until a single target is identified")
		}
		return profile.Resolve("", profile.Generic)
	}
	if report.Incomplete && override == "" {
		return profile.Resolve("", profile.Generic)
	}
	selected, err := profile.Resolve(model, override)
	if err != nil {
		return selected, err
	}
	originals := report.Originals()
	if len(originals) == 1 {
		original := identity.Repository(originals[0])
		fromServer, err := profile.Resolve(original, override)
		if err != nil {
			return fromServer, err
		}
		if selected.ID != profile.Generic && fromServer.ID != profile.Generic && selected.ID != fromServer.ID {
			return profile.Profile{}, fmt.Errorf("served model ID conflicts with server model identity")
		}
		selected = fromServer
		if selected.ID != profile.Generic {
			report.ProfileSource = "server"
		}
	} else if selected.ID != profile.Generic {
		report.ProfileSource = "catalogue"
	}
	if override != "" {
		report.ProfileSource = "manual"
	}
	return selected, nil
}
