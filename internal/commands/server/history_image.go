// Purpose: Lend current attachment metadata and deferred pixel reads under one session lease.
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
	Revision string
	Metadata media.Metadata
	Load     func() (media.Image, error)
	Release  func()
}
type imageHistory interface {
	HistoryImageMetadata(context.Context, int64, int) (media.Metadata, string, error)
	HistoryImage(context.Context, int64, int, history.ReadBounds) (media.Image, error)
}

func (s *Service) OpenImage(ctx context.Context, session string, sequence int64, index int) (ImageRead, error) {
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
	metadata, revision, err := store.HistoryImageMetadata(ctx, sequence, index)
	if err != nil {
		release()
		return ImageRead{}, err
	}
	return ImageRead{Metadata: metadata, Revision: revision, Load: func() (media.Image, error) {
		image, err := store.HistoryImage(ctx, sequence, index, history.ReadBounds{MaxBytes: MaxOriginalBytes, Reserve: memory.Resize})
		if err == nil && (image.Digest != metadata.Digest || image.MIME != metadata.MIME || image.Width != metadata.Width || image.Height != metadata.Height) {
			err = faults.New(diagcatalog.Conflict, diagcatalog.ServerReadRevision)
		}
		return image, err
	}, Release: release}, nil
}
