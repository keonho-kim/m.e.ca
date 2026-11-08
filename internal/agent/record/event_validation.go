// Purpose: Validate durable runtime event payloads before persistence.
// Pattern: Simple Module.
// Usage: Journal adapters use Event.Check at their write boundary.
// Related: internal/agent/record/events.go.
package record

import (
	"encoding/json"

	"time"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/work"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (event Event) Check() error {
	if event.OccurredAt != "" {
		if len(event.OccurredAt) > 35 {
			return faults.Errorf(diagcatalog.RecordInvalidEventOccurrenceTime)
		}
		if _, err := time.Parse(time.RFC3339Nano, event.OccurredAt); err != nil {
			return faults.Errorf(diagcatalog.RecordInvalidEventOccurrenceTime)
		}
	}
	if !diagcatalog.ValidCode(event.ErrorCode) {
		return faults.Errorf(diagcatalog.RecordInvalidEventErrorCode)
	}
	if event.Skill != nil {
		if event.Kind != "skill_loaded" && event.Kind != "tool_succeeded" {
			return faults.Errorf(diagcatalog.RecordSkillMetadataRequiresASuccessfulRead)
		}
		if err := event.Skill.Check(); err != nil {
			return err
		}
	}
	if event.Source != nil {
		if event.Kind != "tool_succeeded" || event.Source.Kind != "team_discussion" && event.Source.Kind != "team_work" {
			return faults.Errorf(diagcatalog.RecordPublicResultSourceRequiresASuccessfulOutcome)
		}
		if err := event.Source.Check(); err != nil {
			return err
		}
	}
	if err := event.Team.Check(event.Kind); err != nil {
		return err
	}
	if event.ApprovalSource != "" {
		if event.Approval == nil || event.Kind != "approved" && event.Kind != "denied" ||
			event.ApprovalSource != "user" && event.ApprovalSource != "all_mighty" || event.Kind == "denied" && event.ApprovalSource != "user" {
			return faults.Errorf(diagcatalog.RecordInvalidApprovalDecisionSource)
		}
	}
	if event.Statistics != nil {
		if event.Kind != "run_checkpoint" || event.Checkpoint == nil || event.Checkpoint.Phase != "finished" || event.Checkpoint.Status == Running || event.RunID != event.Checkpoint.ID {
			return faults.Errorf(diagcatalog.RecordRequestStatisticsRequireTheirFinishedCheckpoint)
		}
		if err := event.Statistics.check(); err != nil {
			return err
		}
	}
	if g := event.Generation; g != nil {
		if event.Kind != "usage" && event.Kind != "generation_finished" && event.Kind != "generation_started" || !work.ValidID(g.ID) || g.Attempt < 1 || g.ElapsedMillis < 0 || g.MeasuredMillis < 0 || g.MeasuredMillis > g.ElapsedMillis {
			return faults.Errorf(diagcatalog.RecordInvalidGenerationProvenance)
		}
		switch g.Role {
		case inference.Agent, inference.Compaction, inference.Diagnostic:
		default:
			return faults.Errorf(diagcatalog.RecordInvalidGenerationRole)
		}
	}
	if event.RunID != "" && !work.ValidID(event.RunID) {
		return faults.Errorf(diagcatalog.RecordInvalidRunEventIdentity)
	}
	if event.ChangePreview != "" {
		if event.Kind != "tool_succeeded" || event.Call == nil || len(event.ChangePreview) > MaxChangePreviewBytes || !utf8.ValidString(event.ChangePreview) {
			return faults.Errorf(diagcatalog.RecordInvalidAppliedChangePreview)
		}
		switch event.Call.Name {
		case "write_file", "edit_file":
		default:
			return faults.Errorf(diagcatalog.RecordUnexpectedChangePreviewTool)
		}
	}

	if event.MeasuredDuration != 0 {
		return faults.Errorf(diagcatalog.RecordTransientMeasurementDurationCannotBePersisted)
	}
	if event.Kind == "usage" {
		if event.Usage == nil {
			return faults.Errorf(diagcatalog.RecordMissingModelTokenUsage)
		}
		if err := event.Usage.Check(); err != nil {
			return err
		}
	} else if event.Usage != nil {
		return faults.Errorf(diagcatalog.RecordOnlyFinalModelUsageMayBePersisted)
	}
	if event.Call != nil {
		if err := event.Call.Check(); err != nil {
			return err
		}
		if event.Call.ID != event.CallID {
			return faults.Errorf(diagcatalog.RecordToolEventCallIDMismatch)
		}
	}
	if len(event.Capture) > modelapi.MaxResponseBytes || len(event.Capture) > 0 && !json.Valid(event.Capture) {
		return faults.Errorf(diagcatalog.RecordInvalidCapturedOutput)
	}
	if len(event.Output) > modelapi.MaxResponseBytes || len(event.Output) > 0 && !json.Valid(event.Output) {
		return faults.Errorf(diagcatalog.RecordInvalidToolEventOutput)
	}
	if event.Checkpoint != nil {
		if err := event.Checkpoint.Check(); err != nil {
			return err
		}
	}
	return nil
}
