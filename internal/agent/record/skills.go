// Purpose: Describe successful instruction reads without copying instruction bodies into notifications.
// Pattern: Execution metadata contract.
// Usage: Runtime outcomes, public events and CLI JSON exchange SkillRead.
// Related: internal/agent/record/events.go, internal/agent/run/skills.go, internal/serverapi/projection.go.
package record

import (
	"encoding/hex"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type SkillRead struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Source   string `json:"source"`
	Revision string `json:"revision"`
}

func (s SkillRead) Check() error {
	if len(s.ID) == 0 || len(s.ID) > 512 || len(s.Name) == 0 || len(s.Name) > 64 || len(s.Source) == 0 || len(s.Source) > 32 || len(s.Revision) != 64 {
		return faults.Errorf(diagcatalog.RecordInvalidSkillReadMetadata)
	}
	_, err := hex.DecodeString(s.Revision)
	return err
}

// PublicKind names a completed skill read consistently across live and saved projections.
func (e Event) PublicKind() string {
	if e.Kind == "tool_succeeded" && e.Skill != nil {
		return "skill_loaded"
	}
	return e.Kind
}
