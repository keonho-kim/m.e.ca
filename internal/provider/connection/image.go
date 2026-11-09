// Purpose: Represent image-input acceptance and its evidence without guessing model families.
// Pattern: Explicit capability value.
// Usage: Discovery and connection probes populate ModelInfo; request composition gates image tools.
// Related: internal/provider/setup/image.go, internal/provider/discovery/model_settings.go.
package connection

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	// ImageInput records supported, unsupported or unknown acceptance and its evidence.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type ImageInput struct {
	State  string `json:"state"`
	Source string `json:"source,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// Supported reports confirmed image-request acceptance.
func (i ImageInput) Supported() bool { return i.State == "supported" }

// Known distinguishes a definitive declaration from missing metadata or a failed probe.
func (i ImageInput) Known() bool { return i.State == "supported" || i.State == "unsupported" }

// Check rejects unsupported capability states before persistence or client binding.
func (i ImageInput) Check() error {
	if i.State != "" && i.State != "unknown" && !i.Known() {
		return faults.Errorf(diagcatalog.ConnectionInvalidImageInputState)
	}
	if len(i.Source) > 256 || len(i.Detail) > 4096 {
		return faults.Errorf(diagcatalog.ConnectionImageInputEvidenceExceedsItsBounds)
	}
	return nil
}

// DeclaredImageInput binds a server-provided boolean to its metadata source.
func DeclaredImageInput(supported bool, source string) ImageInput {
	state := "unsupported"
	if supported {
		state = "supported"
	}
	return ImageInput{State: state, Source: source}
}
