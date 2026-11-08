// Purpose: Define validated final request measurements shared by persistence and presentation.
// Pattern: Immutable execution record contract.
// Usage: Runtime publishes statistics; event validation and terminal history consume them.
// Related: internal/agent/record/events.go, internal/agent/run/statistics.go.
package record

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Statistics struct {
	Concurrent       bool                 `json:"concurrent,omitempty"`
	ElapsedMillis    int64                `json:"elapsed_millis"`
	GenerationMillis int64                `json:"generation_millis"`
	Usage            *modelapi.TokenUsage `json:"usage,omitempty"`
}

func (s Statistics) check() error {
	if s.ElapsedMillis < 0 || s.GenerationMillis < 0 || !s.Concurrent && s.GenerationMillis > s.ElapsedMillis {
		return faults.Errorf(diagcatalog.RecordInvalidRequestMeasurementDuration)
	}
	if s.Usage == nil {
		if s.GenerationMillis != 0 {
			return faults.Errorf(diagcatalog.RecordGenerationMeasurementRequiresReportedTokenUsage)
		}
	} else if err := s.Usage.CheckTotals(); err != nil {
		return err
	}
	return nil
}
