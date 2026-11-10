// Purpose: Resolve durable native request retries before borrowing temporary uploads.
// Pattern: Application input boundary with stable reference identity.
// Usage: Native Start commands call StartAttachments; direct engine callers use Start.
// Related: internal/commands/server/tasks.go, internal/commands/server/identity.go.
package server

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/agent/media"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func (s *Service) StartAttachments(ctx context.Context, session, key, text string, ids, paths []string) (Task, error) {
	if !sessionstate.ValidID(session) || len(key) < 8 || len(key) > 128 || strings.ContainsAny(key, "\r\n\x00") || len(ids) > media.MaxImages {
		return Task{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerRequestKeyMustContainToUTFBytes)
	}
	for _, id := range ids {
		if !strings.HasPrefix(id, "img_") || len(id) != 30 {
			return Task{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.MediaAttachmentUnavailable)
		}
	}
	if !utf8.ValidString(key) || !utf8.ValidString(text) || len(text) > MaxPromptBytes || len(paths) > 256 {
		return Task{}, faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	h := sha256.New()
	var buffer [4096]byte
	write := func(value string) {
		binary.BigEndian.PutUint64(buffer[:8], uint64(len(value)))
		_, _ = h.Write(buffer[:8])
		for len(value) > 0 {
			n := copy(buffer[:], value)
			_, _ = h.Write(buffer[:n])
			value = value[n:]
		}
	}

	write("native-attachments-v1")
	write(text)
	write(strconv.Itoa(len(ids)))
	for _, id := range ids {
		write(id)
	}
	write(strconv.Itoa(len(paths)))
	for _, path := range paths {
		write(path)
	}
	var digest [32]byte
	h.Sum(digest[:0])
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return Task{}, err
	}
	defer done()
	if task, found, err := s.existingRequest(ctx, session, key, digest); err != nil || found {
		return task, err
	}
	images, err := s.ImageAttachments(ids)
	if err != nil {
		return Task{}, err
	}
	resolved, err := s.resolvePaths(ctx, text, paths)
	if err != nil {
		return Task{}, err
	}
	return s.start(ctx, session, key, resolved, digest, images...)
}
