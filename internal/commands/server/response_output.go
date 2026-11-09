// Purpose: Restore public result text under an owned conversation read lease and explicit read bounds.
// Pattern: Application original-output read boundary.
// Usage: Compatible JSON and terminal SSE responses use ResponseResult.
// Related: internal/commands/server/detail.go, internal/commands/server/response.go, internal/storage/sqlite/response_output.go.
package server

import (
	"context"

	"github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

// MaxReadBytes bounds metadata/history pages. Original reads can restore one
// accepted record including its JSON representation, under API memory admission.
const MaxReadBytes = 32 << 20
const MaxOriginalBytes = 2 * model.MaxResponseBytes

func (s *Service) ResponseResult(ctx context.Context, value Response, reserve func(int) error) (string, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return "", err
	}
	defer done()
	if value.Session == "" || value.Through <= value.Begin {
		return "", nil
	}
	// A reader lease keeps the selected database alive across replacement.
	if c, readCtx, release, err := s.beginResultRead(ctx, value.Session); err == nil {
		if c.Runtime.Session.ID == value.Session && c.Output != nil {
			defer release()
			return c.Output.ResponseOutput(readCtx, value.Session, value.Begin, value.Through, MaxOriginalBytes, reserve)
		}
		release()
	}
	if s.options.ReadOutput == nil {
		return "", faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerOriginalOutputIsUnavailable)
	}
	return s.options.ReadOutput(ctx, value.Session, value.Begin, value.Through, MaxOriginalBytes, reserve)
}
