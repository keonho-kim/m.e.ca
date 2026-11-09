// Purpose: Lend explicit image downloads under the same session and memory ownership as originals.
// Pattern: Application read lease.
// Usage: The HTTP image adapter closes ImageRead after the response body is written.
// Related: internal/storage/sqlite/history_image.go, internal/serverapi/history_image.go.
package server

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
	"github.com/keonho-kim/meca/internal/storage/history"
	"sync"
)

type ImageRead struct {
	Image   media.Image
	Release func()
}
type imageHistory interface {
	HistoryImage(context.Context, int64, int, history.ReadBounds) (media.Image, error)
}

func (s *Service) ReadImage(ctx context.Context, session string, sequence int64, index int) (ImageRead, error) {
	if sequence < 1 || index < 0 || index >= media.MaxImages {
		return ImageRead{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	c, ctx, done, err := s.beginSessionRead(ctx, session)
	if err != nil {
		return ImageRead{}, err
	}
	memory := processmemory.NewPayloadLease()
	release := sync.OnceFunc(func() { memory.Close(); done() })
	store, ok := c.History.(imageHistory)
	if !ok {
		release()
		return ImageRead{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerImageMissing)
	}
	image, err := store.HistoryImage(ctx, sequence, index, history.ReadBounds{MaxBytes: MaxOriginalBytes, Reserve: memory.Resize})
	if err != nil {
		release()
		return ImageRead{}, err
	}
	return ImageRead{Image: image, Release: release}, nil
}
