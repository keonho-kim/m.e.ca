// Purpose: Reject submitted images before task execution when the selected model lacks vision support.
// Pattern: Deterministic admission policy.
// Usage: Native admission calls checkImageModel after model preparation.
// Related: internal/commands/server/tasks.go, internal/commands/server/submission.go.
package server

import (
	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func checkImageModel(c *Conversation, images []media.Image) error {
	if len(images) == 0 {
		return nil
	}
	if c.Runtime.Selection.Details == nil || !c.Runtime.Selection.Details.ImageInput.Supported() {
		return faults.New(diagcatalog.FailedPrecondition, diagcatalog.MediaImageInputUnavailable)
	}
	return nil
}
