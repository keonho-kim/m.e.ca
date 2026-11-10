// Purpose: Bound one submitted image collection independently of display and model context budgets.
// Pattern: Immutable input contract and deterministic validation.
// Usage: Composer, API admission and runner validate snapshots before taking task ownership.
// Related: internal/agent/media/image.go, internal/commands/agentrequest/request.go.
package media

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const MaxImages = 8
const MaxInputBytes = 128 << 20

func CheckInput(images []Image) error {
	if len(images) > MaxImages {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaImageCollectionLimit)
	}
	size := 0
	for _, image := range images {
		if err := image.Check(); err != nil {
			return err
		}
		size += len(image.Data)
		if size > MaxInputBytes {
			return faults.New(diagcatalog.ResourceExhausted, diagcatalog.MediaImageCollectionLimit)
		}
	}
	return nil
}
