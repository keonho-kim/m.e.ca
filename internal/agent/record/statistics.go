// Purpose: Define validated final request measurements shared by persistence and presentation.
// Pattern: Immutable execution record contract.
// Usage: Runtime publishes statistics; event validation and terminal history consume them.
// Related: internal/agent/record/events.go, internal/agent/run/statistics.go.
package record

import (
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type Statistics struct {
	ElapsedMillis    int64                `json:"elapsed_millis"`
	GenerationMillis int64                `json:"generation_millis"`
	Usage            *modelapi.TokenUsage `json:"usage,omitempty"`
}

func (s Statistics) check() error {
	if s.ElapsedMillis < 0 || s.GenerationMillis < 0 || s.GenerationMillis > s.ElapsedMillis {
		return fmt.Errorf("invalid request measurement duration")
	}
	if s.Usage == nil {
		if s.GenerationMillis != 0 {
			return fmt.Errorf("generation measurement requires reported token usage")
		}
	} else if err := s.Usage.CheckTotals(); err != nil {
		return err
	}
	return nil
}
