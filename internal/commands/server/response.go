// Purpose: Define durable compatible submissions and references to original engine output.
// Pattern: Consumer-owned persistence and result contracts.
// Usage: Submit and ResponseResult serve the OpenAI adapter without transport DTOs.
// Related: internal/commands/server/submission.go, internal/storage/serverrequests/responses.go.
package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/work"
)

// MaxReadBytes bounds original restoration per external read, independently of execution limits.
const MaxReadBytes = 32 << 20

type InputMessage struct{ Role, Text string }
type Submission struct {
	Kind, Key, Previous string
	Messages            []InputMessage
	Stored, Background  bool
}

// Response stores only output boundaries. Text remains in the conversation journal.
type Response struct {
	ID, Scope, Digest, Kind       string
	Session, Task, Previous, Next string
	Status, Error, Code           string
	TaskStatus                    string
	Created, Completed            int64
	Begin, Through                int64
	Stored, Background, Team      bool
	End                           work.End
	Usage                         *model.TokenUsage
}

type ResponseStore interface {
	ReserveResponse(context.Context, Response) (Response, bool, error)
	ReadResponse(context.Context, string) (Response, error)
	SaveResponse(context.Context, Response) error
}

type OutputHistory interface {
	OutputPosition(context.Context, string) (int64, error)
	ResponseOutput(context.Context, string, int64, int64, int) (string, error)
}

type Submitted struct {
	Response  Response
	Cursor    string
	Duplicate bool
}

func ProjectScope(project string) string {
	sum := sha256.Sum256([]byte(project))
	return hex.EncodeToString(sum[:])
}
func ResponseID(scope, kind, key string) string {
	sum := sha256.Sum256([]byte(scope + "\x00" + kind + "\x00" + key))
	return "resp_" + hex.EncodeToString(sum[:])
}

func (r Response) task() Task {
	status := r.TaskStatus
	if status == "" {
		status = "interrupted"
	}
	return Task{ID: r.Task, Session: r.Session, Status: status, Error: r.Error}
}

// ResponseState hides nonretained results only from public lookup; POST callers
// can still await their accepted task and an idempotent retry can recover it.
func (s *Service) ResponseState(ctx context.Context, id string, lookup bool) (Response, error) {
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}
	s.mu.Lock()
	if s.closed || s.ctx.Err() != nil {
		s.mu.Unlock()
		return Response{}, failure("unavailable", "server is shutting down")
	}
	if r := s.active; r != nil && r.response != nil && r.response.ID == id {
		value := *r.response
		s.mu.Unlock()
		if lookup && (!value.Stored || value.Kind != "response") {
			return Response{}, failure("not_found", "response was not found")
		}
		return value, nil
	}
	s.mu.Unlock()
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return Response{}, err
	}
	defer done()
	if s.options.Responses == nil {
		return Response{}, failure("failed_precondition", "compatible results are unavailable")
	}
	value, err := s.options.Responses.ReadResponse(ctx, id)
	if err != nil {
		return Response{}, err
	}
	if value.Scope != s.scope || lookup && (!value.Stored || value.Kind != "response") {
		return Response{}, failure("not_found", "response was not found")
	}
	s.mu.Lock()
	live := s.active != nil && s.active.response != nil && s.active.response.ID == id || s.submitting == id
	s.mu.Unlock()
	if !live && (value.Status == "queued" || value.Status == "in_progress") {
		// Completion is persisted before active is cleared. Re-read after
		// observing that transition rather than classifying an older file view.
		value, err = s.options.Responses.ReadResponse(ctx, id)
		if err != nil {
			return Response{}, err
		}
	}
	if !live && (value.Status == "queued" || value.Status == "in_progress") {
		value.Status, value.Error = "failed", "request was interrupted; its effects must not be replayed"
		value.Code = "interrupted"
		value.End = work.End{Status: work.Interrupted, Reason: value.Error}
	}
	return value, nil
}

func (s *Service) ResponseResult(ctx context.Context, value Response) (string, error) {
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
			return c.Output.ResponseOutput(readCtx, value.Session, value.Begin, value.Through, MaxReadBytes)
		}
		release()
	}
	if s.options.ReadOutput == nil {
		return "", failure("failed_precondition", "original output is unavailable")
	}
	return s.options.ReadOutput(ctx, value.Session, value.Begin, value.Through, MaxReadBytes)
}
