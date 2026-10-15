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
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/agent/verification"
	"strings"
)

type Input struct {
	Checks    []verification.Record
	Request   string
	Grounding grounding.Record
	Call      tool.Call
	Candidate tool.Candidate
}
type Generate func(context.Context, modelapi.Request, func(modelapi.Event) error) error
type Fetch func(context.Context, ContextRequest) (json.RawMessage, error)

func Review(ctx context.Context, input Input, request modelapi.Request, maxContext int, generate Generate, fetch Fetch, redact func(string) string) (Result, error) {
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
		Checks          []verification.Record
		Context         []json.RawMessage
	}{input.Request, input.Grounding.Reason, input.Candidate.Files, input.Checks, nil}
	correction := ""
	fail := func(err error) (Result, error) {
		result.Error = redact(err.Error())
		if len(result.Error) > 2048 {
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
		turn.Messages = append(append([]modelapi.Message(nil), request.Messages...), modelapi.Message{Role: "user", Content: "REVIEW CANDIDATE before execution. JSON data:\n" + string(body) + "\n" + correction})
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
			if text.Len()+len(e.Delta) > 8192 {
				return fmt.Errorf("review response exceeds limit")
			}
			text.WriteString(e.Delta)
			done = done || e.Done
			return ctx.Err()
		})
		if err != nil {
			return fail(err)
		}
		if !done {
			return fail(fmt.Errorf("review response incomplete"))
		}
		if redact(text.String()) != text.String() {
			return fail(fmt.Errorf("protected content in review"))
		}
		verdict, parseErr := Parse(text.String())
		if parseErr == nil {
			parseErr = references(verdict, input.Candidate)
		}
		if parseErr != nil {
			if attempt == 1 {
				return fail(parseErr)
			}
			correction = "Correct the output contract: " + parseErr.Error() + ". Return exactly one JSON object."
			continue
		}
		result.Verdict = verdict
		if verdict.Decision != "needs_context" {
			result.Complete = true
			result.Passed = verdict.Decision == "pass"
			return result, result.Check()
		}
		if attempt == 1 || fetch == nil {
			return fail(fmt.Errorf("review context remains incomplete"))
		}
		for _, needed := range verdict.ContextRequests {
			content, err := fetch(ctx, needed)
			if err != nil {
				return fail(err)
			}
			packet.Context = append(packet.Context, content)
		}
		correction = "Additional scoped observations are supplied. Give pass or specific reject; further context rounds are unavailable."
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
