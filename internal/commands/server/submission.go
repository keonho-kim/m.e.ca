// Purpose: Atomically bind a compatible request, select its conversation and admit its task.
// Pattern: Reserved application state transition.
// Usage: OpenAI handlers call Submit instead of racing Select and Start commands.
// Related: internal/commands/server/submission_session.go, internal/commands/server/execution.go.
package server

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/agent/work"
	"github.com/keonho-kim/meca/internal/diagnostics"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Service) Submit(ctx context.Context, input Submission) (submitted Submitted, resultErr error) {
	if err := checkSubmission(input); err != nil {
		return Submitted{}, err
	}
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return Submitted{}, err
	}
	defer done()
	if s.options.Responses == nil || s.options.ReadOutput == nil {
		return Submitted{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerCompatibleSubmissionStorageIsUnavailable)
	}
	key := input.Key
	if key == "" {
		key = rand.Text()
	}
	scope := s.scope
	value := Response{ID: ResponseID(scope, input.Kind, key), Scope: scope, Kind: input.Kind, Previous: input.Previous, Stored: input.Stored, Background: input.Background, Status: "queued", Created: time.Now().Unix()}
	value.Digest = submissionDigest(input)
	if old, found, err := s.findSubmission(ctx, value); err != nil || found {
		return Submitted{Response: old, Duplicate: true}, err
	}
	change, err := s.beginChange(ctx)
	if err != nil {
		if old, found, cause := s.findSubmission(ctx, value); cause != nil || found {
			return Submitted{Response: old, Duplicate: true}, cause
		}
		return Submitted{}, err
	}
	defer change.finish()
	s.mu.Lock()
	s.submitting = value.ID
	s.mu.Unlock()
	value, duplicate, err := s.options.Responses.ReserveResponse(change.ctx, value)
	if err != nil || duplicate {
		return Submitted{Response: value, Duplicate: duplicate}, err
	}
	// Failed or uncertain admissions remain queryable and are never replayed.
	defer func() {
		if resultErr == nil {
			return
		}
		value.Status, value.Error, value.Completed = "failed", diagcatalog.Text(diagcatalog.AdmissionUncertain), max(value.Created, time.Now().Unix())
		value.Code = diagcatalog.AdmissionFailed
		value.TaskStatus = diagcatalog.Interrupted
		value.Through = value.Begin
		value.End = work.End{Status: work.Failed, Reason: value.Error}
		flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		saveErr := s.options.Responses.SaveResponse(flush, value)
		cancel()
		saveErr = s.confirmCompletion(ctx, Task{}, &value, saveErr)
		if saveErr != nil {
			s.mu.Lock()
			s.completionErr = saveErr
			s.closeErr = errors.Join(s.closeErr, saveErr)
			s.mu.Unlock()
			value.Code, value.Error = diagcatalog.CompletionNotSaved, diagcatalog.Text(diagcatalog.CompletionUncertain)
			value.End.Reason = value.Error
			resultErr = errors.Join(resultErr, faults.New(diagcatalog.CompletionNotSaved, diagcatalog.CompletionUncertain), saveErr)
			diagnostics.Log(ctx, diagcatalog.CompletionSaveFailed, "error_code", faults.Code(saveErr))
		}
		submitted.Response = value
	}()
	c, err := s.submissionSession(change, input.Previous, value.ID)
	if err != nil {
		return Submitted{Response: value}, err
	}
	value.Session, value.Team = c.Runtime.Session.ID, c.TeamEnabled
	text := input.Messages[len(input.Messages)-1].Text
	// The reserved response is the durable admission record for this task.
	// Its reversible identity also serves native task lookup without a second file.
	task := Task{ID: "task_" + value.ID[5:], Session: value.Session, Status: "accepted"}
	value.Task = task.ID
	s.mu.Lock()
	s.admitting = task.ID
	s.mu.Unlock()
	clientContext, err := seedInput(change.ctx, c, input.Messages[:len(input.Messages)-1], input.Kind == "response")
	if err != nil {
		return Submitted{Response: value}, err
	}
	value.Begin, err = c.Output.OutputPosition(change.ctx, value.Session)
	if err != nil {
		return Submitted{Response: value}, err
	}
	value.Status = "in_progress"
	if err := s.options.Responses.SaveResponse(change.ctx, value); err != nil {
		return Submitted{Response: value}, err
	}
	s.mu.Lock()
	cursor := s.feed.cursor()
	s.mu.Unlock()
	if _, err := s.launch(c, task, text, &value, clientContext); err != nil {
		return Submitted{Response: value}, err
	}
	return Submitted{Response: value, Cursor: cursor}, nil
}

func (s *Service) findSubmission(ctx context.Context, value Response) (Response, bool, error) {
	old, err := s.options.Responses.ReadResponse(ctx, value.ID)
	if err != nil {
		var public *faults.Error
		if errors.As(err, &public) && public.ErrorCode() == diagcatalog.NotFound {
			return Response{}, false, nil
		}
		return Response{}, false, err
	}
	if old.Digest != value.Digest || old.Scope != value.Scope || old.Kind != value.Kind {
		return Response{}, true, faults.New(diagcatalog.Conflict, diagcatalog.ServerIdempotencyKeyWasAlreadyUsedWithA)
	}
	return old, true, nil
}

func checkSubmission(input Submission) error {
	if input.Kind != "chat" && input.Kind != "response" {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerUnsupportedCompatibleRequestKind)
	}
	if len(input.Key) > 128 || !utf8.ValidString(input.Key) || strings.ContainsAny(input.Key, "\r\n\x00") {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerIdempotencyKeyMustContainToUTFBytes)
	}
	if len(input.Messages) == 0 || len(input.Messages) > 4096 || input.Messages[len(input.Messages)-1].Role != "user" {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerInputMustEndWithAUserMessage)
	}
	bytes := 0
	for _, message := range input.Messages {
		switch message.Role {
		case "system", "developer", "user", "assistant":
		default:
			return faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerUnsupportedInputRole)
		}
		if !utf8.ValidString(message.Text) || strings.TrimSpace(message.Text) == "" {
			return faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerInputMessagesMustContainUTFText)
		}
		bytes += len(message.Text)
	}
	if bytes > 16<<20 {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerInputTextExceedsMiB)
	}
	return nil
}
