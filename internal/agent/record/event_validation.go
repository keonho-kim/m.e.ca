// Purpose: Validate durable runtime event payloads before persistence.
// Pattern: Simple Module.
// Usage: Journal adapters use Event.Check at their write boundary.
// Related: internal/agent/record/events.go.
package record

import (
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/work"
	"strings"
	"unicode/utf8"
)

func (event Event) Check() error {
	if side := event.LAT; side != nil {
		if !event.Internal || event.Kind == "run_checkpoint" {
			return fmt.Errorf("invalid isolated LAT provenance")
		}
		if err := side.Check(); err != nil {
			return err
		}
	}
	if event.Kind == "lat_message" || event.Message != nil {
		if event.Kind != "lat_message" || event.Message == nil || event.LAT == nil {
			return fmt.Errorf("native LAT message requires isolated provenance")
		}
		if err := event.Message.Check(); err != nil {
			return err
		}
	}
	if event.Kind == "lat_checkpoint" && (event.LAT == nil || event.Checkpoint == nil) {
		return fmt.Errorf("native LAT checkpoint requires isolated provenance")
	}
	if event.Kind == "lat_plan" && (len(event.Text) > 16*1024*1024 || strings.TrimSpace(event.Text) == "" || !utf8.ValidString(event.Text) || strings.ContainsRune(event.Text, 0)) {
		return fmt.Errorf("refined implementation plan requires nonempty UTF-8 text within 16 MiB")
	}
	if event.ApprovalSource != "" {
		if event.Approval == nil || event.Kind != "approved" && event.Kind != "denied" ||
			event.ApprovalSource != "user" && event.ApprovalSource != "all_mighty" || event.Kind == "denied" && event.ApprovalSource != "user" {
			return fmt.Errorf("invalid approval decision source")
		}
	}
	if event.Statistics != nil {
		if event.Kind != "run_checkpoint" && event.Kind != "lat_checkpoint" || event.Checkpoint == nil || event.Checkpoint.Phase != "finished" || event.Checkpoint.Status == Running || event.RunID != event.Checkpoint.ID {
			return fmt.Errorf("request statistics require their finished checkpoint")
		}
		if err := event.Statistics.check(); err != nil {
			return err
		}
	}
	if g := event.Generation; g != nil {
		if event.Kind != "usage" && event.Kind != "generation_finished" || !work.ValidID(g.ID) || g.Attempt < 1 || g.ElapsedMillis < 0 || g.MeasuredMillis < 0 || g.MeasuredMillis > g.ElapsedMillis {
			return fmt.Errorf("invalid generation provenance")
		}
		switch g.Role {
		case inference.Agent, inference.Compaction, inference.Diagnostic, inference.LAT, inference.LATCompaction:
		default:
			return fmt.Errorf("invalid generation role")
		}
	}
	if event.RunID != "" && !work.ValidID(event.RunID) {
		return fmt.Errorf("invalid run event identity")
	}
	if event.ChangePreview != "" {
		if event.Kind != "tool_succeeded" || event.Call == nil || len(event.ChangePreview) > MaxChangePreviewBytes || !utf8.ValidString(event.ChangePreview) {
			return fmt.Errorf("invalid applied change preview")
		}
		switch event.Call.Name {
		case "write_file", "edit_file":
		default:
			return fmt.Errorf("unexpected change preview tool")
		}
	}

	if event.TokensPerSecond != nil {
		return fmt.Errorf("transient throughput cannot be persisted")
	}
	if event.Kind == "usage" {
		if event.Usage == nil {
			return fmt.Errorf("missing model token usage")
		}
		if err := event.Usage.Check(); err != nil {
			return err
		}
	} else if event.Usage != nil {
		return fmt.Errorf("only final model usage may be persisted")
	}
	if event.Call != nil {
		if err := event.Call.Check(); err != nil {
			return err
		}
		if event.Call.ID != event.CallID {
			return fmt.Errorf("tool event call ID mismatch")
		}
	}
	if len(event.Capture) > modelapi.MaxResponseBytes || len(event.Capture) > 0 && !json.Valid(event.Capture) {
		return fmt.Errorf("invalid captured output")
	}
	if len(event.Output) > modelapi.MaxResponseBytes || len(event.Output) > 0 && !json.Valid(event.Output) {
		return fmt.Errorf("invalid tool event output")
	}
	if event.Checkpoint != nil {
		if err := event.Checkpoint.Check(); err != nil {
			return err
		}
	}
	return nil
}

func (side LATRun) Check() error {
	if !work.ValidID(side.ID) || !work.ValidID(side.ParentRunID) {
		return fmt.Errorf("invalid isolated LAT provenance")
	}
	switch side.Phase {
	case "plan", "critique", "refine", "review":
		return nil
	default:
		return fmt.Errorf("invalid isolated LAT phase")
	}
}
