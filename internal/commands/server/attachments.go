// Purpose: Own bounded expiring uploaded image snapshots until requests borrow immutable bytes.
// Pattern: Process-owned attachment registry with joined expiry callbacks.
// Usage: HTTP upload stages images; native commands resolve IDs before admission.
// Related: internal/commands/server/lifecycle.go, internal/serverapi/attachments.go.
package server

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const attachmentBytes = 256 << 20
const AttachmentLifetime = 5 * time.Minute

type attachment struct {
	expires time.Time
	image   media.Image
	timer   *time.Timer
	done    func()
}

// StageImage takes ownership of immutable, already decoded image bytes.
func (s *Service) StageImage(image media.Image) (string, error) {
	if err := image.Check(); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.ctx.Err() != nil {
		return "", faults.New(diagcatalog.Unavailable, diagcatalog.ServerServerIsShuttingDown)
	}
	size := len(image.Data)
	for _, a := range s.attachments {
		size += len(a.image.Data)
	}
	if len(s.attachments) >= 16 || size > attachmentBytes {
		return "", faults.New(diagcatalog.ResourceExhausted, diagcatalog.MediaAttachmentCapacity)
	}
	release, available := s.options.Attachments.reserve(len(image.Data))
	if !available {
		return "", faults.New(diagcatalog.ResourceExhausted, diagcatalog.MediaAttachmentCapacity)
	}
	if s.attachments == nil {
		s.attachments = make(map[string]*attachment)
	}
	id := "img_" + rand.Text()
	s.workers.Add(1)
	a := &attachment{image: image, expires: time.Now().Add(AttachmentLifetime), done: sync.OnceFunc(func() { release(); s.workers.Done() })}
	a.timer = time.AfterFunc(AttachmentLifetime, func() {
		defer a.done()
		s.mu.Lock()
		delete(s.attachments, id)
		s.mu.Unlock()
	})
	s.attachments[id] = a
	return id, nil
}

func (s *Service) ImageExpiry(id string) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a := s.attachments[id]; a != nil {
		return a.expires
	}
	return time.Time{}
}

func (s *Service) ImageAttachments(ids []string) ([]media.Image, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(ids) > media.MaxImages {
		return nil, faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaImageCollectionLimit)
	}
	images := make([]media.Image, 0, len(ids))
	for _, id := range ids {
		a := s.attachments[id]
		if a == nil {
			return nil, faults.New(diagcatalog.NotFound, diagcatalog.MediaAttachmentUnavailable)
		}
		images = append(images, a.image)
	}
	return images, media.CheckInput(images)
}

func (s *Service) ReleaseImage(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	a := s.attachments[id]
	if a == nil {
		return faults.New(diagcatalog.NotFound, diagcatalog.MediaAttachmentUnavailable)
	}
	delete(s.attachments, id)
	if a.timer.Stop() {
		a.done()
	}
	return nil
}

// closeAttachments requires s.mu and runs before joining workers.
func (s *Service) closeAttachments() {
	for id, a := range s.attachments {
		delete(s.attachments, id)
		if a.timer.Stop() {
			a.done()
		}
	}
}
