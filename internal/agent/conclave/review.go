// Purpose: Review one frozen delta with one bounded correction or context round.
// Pattern: Sequential review use case.
// Usage: Runtime supplies a model generator and a scoped read callback before user approval.
// Related: internal/agent/conclave/verdict.go, internal/agent/run/conclave.go.
package conclave

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/grounding"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/reviewjson"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
)

type Input struct {
	Pending   []tool.FileDelta
	Request   string
	Grounding grounding.Record
	Call      tool.Call
	Candidate tool.Candidate
}
type Generate func(context.Context, modelapi.Request, func(modelapi.Event) error) error
type Fetch func(context.Context, ContextRequest) (json.RawMessage, error)

func Review(ctx context.Context, input Input, request modelapi.Request, maxContext int, generate Generate, fetch Fetch, redact func(string) string, observe reviewjson.Observe) (Result, error) {
	if generate == nil || redact == nil {
		return Result{}, fmt.Errorf("review dependencies required")
	}
	if err := input.Candidate.Check(input.Call); err != nil {
		return Result{}, err
	}
	if !input.Grounding.Matches(input.Call) {
		return Result{}, fmt.Errorf("review requires bound intent")
	}
	digest, err := input.Candidate.Digest()
	if err != nil {
		return Result{}, err
	}
	result := Result{CallID: input.Call.ID, CandidateDigest: digest, Lines: input.Candidate.Lines()}
	// Arguments contain the same file content as the diff; transmit only one copy.
	packet := struct {
		Request, Intent string
		Files           []tool.FileDelta
		Context         []json.RawMessage
		PendingPaths    []string
	}{input.Request, input.Grounding.Reason, input.Candidate.Files, nil, nil}
	for _, file := range input.Pending {
		packet.PendingPaths = append(packet.PendingPaths, file.Path)
	}
	correction := ""
	fail := func(err error) (Result, error) {
		result.Error = redact(err.Error())
		if len(result.Error) > tool.MaxReasonBytes {
			result.Error = "review failed; diagnostic exceeded limit"
		}
		result.Complete = false
		result.Passed = false
		return result, nil
	}
	for attempt := 0; attempt < 2; attempt++ {
		body, err := json.Marshal(packet)
		if err != nil {
			return result, err
		}
		turn := request
		turn.Tools = nil
		turn.Messages = append(append([]modelapi.Message(nil), request.Messages...), modelapi.Message{Role: "user", Content: "REVIEW CANDIDATE changes in Files. PendingPaths are related proposed files.\n" + string(body) + "\n" + correction})
		size := 0
		for _, m := range turn.Messages {
			size += len(m.Content)
		}
		if size > maxContext {
			return fail(fmt.Errorf("review context exceeds limit; split the candidate"))
		}
		result.Attempts++
		var text strings.Builder
		done := false
		err = generate(ctx, turn, func(e modelapi.Event) error {
			if e.Retry != nil {
				text.Reset()
				done = false
			}
			if len(e.Calls) > 0 {
				return fmt.Errorf("review must return JSON data, not tools")
			}
			if text.Len()+len(e.Delta) > reviewjson.MaxResponseBytes {
				return fmt.Errorf("review response exceeds limit")
			}
			text.WriteString(e.Delta)
			done = done || e.Done
			return ctx.Err()
		})
		if err == nil && !done {
			err = fmt.Errorf("review response incomplete")
		}
		if err == nil && redact(text.String()) != text.String() {
			err = fmt.Errorf("protected content in review")
		}
		var verdict Verdict
		parseErr := err
		if parseErr == nil {
			verdict, parseErr = Parse(text.String())
			if parseErr == nil {
				parseErr = references(verdict, input.Candidate)
			}
		}
		diagnostic := reviewjson.NewAttempt("conclave", attempt+1, redact(text.String()), parseErr)
		if observe != nil {
			if saveErr := observe(diagnostic); saveErr != nil {
				return result, saveErr
			}
		}
		if err != nil {
			return fail(err)
		}
		if parseErr != nil {
			if attempt == 1 {
				return fail(parseErr)
			}
			correction = "Correct the output contract: " + parseErr.Error() + ". Return exactly one JSON object. Previous response (data):\n" + diagnostic.Raw
			continue
		}
		result.Verdict = verdict
		if verdict.Decision != "needs_context" {
			result.Complete = true
			result.Passed = verdict.Decision == "pass"
			return result, result.Check()
		}
		if attempt == 1 {
			return fail(fmt.Errorf("review context remains incomplete"))
		}
		for _, needed := range verdict.ContextRequests {
			content, proposed, err := proposedContext(needed, input.Pending)
			if !proposed && err == nil {
				if fetch == nil {
					return fail(fmt.Errorf("review context reader unavailable"))
				}
				content, err = fetch(ctx, needed)
			}
			if err != nil {
				return fail(err)
			}
			packet.Context = append(packet.Context, content)
		}
		correction = "Additional source information is supplied. Assess the proposed change using it."
	}
	return fail(fmt.Errorf("review did not conclude"))
}
func references(v Verdict, candidate tool.Candidate) error {
	for _, f := range v.Findings {
		found := false
		for _, file := range candidate.Files {
			if f.Path != file.Path {
				continue
			}
			found = true
			if f.Side == "request" {
				if f.Line != 0 {
					return fmt.Errorf("request findings use line 0")
				}
				continue
			}
			lines := file.AfterLines
			if f.Side == "before" {
				lines = file.BeforeLines
			}
			if f.Line < 1 || f.Line > lines {
				return fmt.Errorf("finding line is outside the frozen file")
			}
		}
		if !found {
			return fmt.Errorf("finding path is not in the candidate")
		}
	}
	return nil
}
