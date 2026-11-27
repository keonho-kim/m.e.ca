// Purpose: Advance native model decisions, recovery and effects until a final response is ready.
// Pattern: Serial decision loop over one prepared execution.
// Usage: Runner.Run calls drive after conversation admission; owned cleanup remains outside the loop.
// Related: internal/agent/run/run.go, internal/agent/run/codemode.go, internal/agent/run/finish.go.
package run

import (
	"context"
	"errors"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var errEmptyReply = faults.Errorf(diagcatalog.RunEmptyReply)

func (s *execution) drive(ctx context.Context, approve Approver) (resultErr error) {
	var err error
	var draft replyDraft
	defer func() {
		if resultErr != nil && draft.output.bytes() != 0 {
			s.presentation = draft.presentation
			resultErr = errors.Join(resultErr, s.recordReply(ctx, draft.output.text, draft.output.thinking, true))
		}
	}()
	compactionAttempted := false
	correctedReply := false
	argumentCorrections := 0
	var correctionRevision uint64
	for {
		if _, err = s.collectCommands(ctx); err != nil {
			return err
		}
		if _, err = s.admitInstructions(ctx); err != nil {
			return err
		}
		if correctionRevision != s.instructionRevision {
			correctionRevision = s.instructionRevision
			argumentCorrections = 0
		}
		if tail := draft.continuation; tail != nil && tail.revision != s.instructionRevision {
			// The new instruction needs an ordinary tool-enabled decision.
			// Keep the visible draft until that decision produces a validated replacement.
			draft.continuation, draft.truncatedText = nil, false
			correctedReply = false
		}
		if finished, decisionErr := s.prepareDecision(ctx, draft.output.bytes() != 0); finished || decisionErr != nil {
			return decisionErr
		}
		s.checkpoint.Iteration = s.usage.ToolCalls
		s.checkpoint.ModelCalls = s.usage.ModelCalls
		s.checkpoint.Phase = "decision"
		s.checkpoint.Tool = ""
		if err = s.persist(ctx); err != nil {
			return err
		}
		decision, askErr := s.prepareAgentRequest(ctx, draft.continuation)
		var reply modelapi.Response
		if askErr == nil {
			if current, err := s.preparedCurrent(ctx, decision); err != nil {
				return err
			} else if !current {
				continue
			}
			if !compactionAttempted {
				err := s.compactMemory(ctx, decision.prepared.Budget, nil, func() {
					compactionAttempted = true
					decision = decisionRequest{}
				})
				if err != nil {
					return err
				}
				if compactionAttempted {
					// Re-admit changes and measure the committed compacted context.
					continue
				}
			}
			if tail := draft.continuation; tail != nil {
				if !tail.fits(decision.prepared.Budget, s.instructionRevision) {
					return tail.cause
				}
				if err := s.emit(runrecord.Event{Kind: "retry", Text: diagcatalog.Text(diagcatalog.RunContinueAnswer)}); err != nil {
					return err
				}
			}
			reply, askErr = s.askResponse(ctx, decision.request, decision.prepared.Send, &draft)
			decision.prepared.Send = nil
			decision.request.Messages, decision.request.Tools = nil, nil
			compactionAttempted = false
		}
		if askErr == nil {
			s.capacityRecovery = false
			correctedReply = false
		}
		if askErr != nil {
			if ctx.Err() != nil || errors.Is(askErr, errRuntimeRecord) {
				return askErr
			}
			if tail := draft.continuation; tail != nil {
				if errors.Is(askErr, tail.cause) {
					return askErr
				}
				return errors.Join(tail.cause, askErr)
			}
			if draft.truncatedText && decision.request.MaxOutput == 0 && decision.prepared.Budget.InputTokens > 0 &&
				s.usage.ModelCalls < s.limits.MaxModelCalls {
				draft.continuation = &textContinuation{text: draft.output.text, revision: decision.instructionRevision, cause: askErr}
				continue
			}
			var limit *modelapi.OutputLimitError
			if errors.As(askErr, &limit) {
				return askErr
			}
			var malformed *modelapi.ToolCallError
			if errors.As(askErr, &malformed) {
				s.result.HadToolFailure = true
			}
			empty := errors.Is(askErr, errEmptyReply)
			if !empty && malformed == nil && draft.output.bytes() != 0 {
				return askErr
			}
			if handled, imageErr := s.recoverImages(ctx, askErr); handled {
				if imageErr != nil {
					return imageErr
				}
				continue
			}
			var capacity *modelapi.ContextLimitError
			if !s.capacityRecovery && errors.As(askErr, &capacity) && !capacity.Partial {
				s.capacityRecovery = true
				decision = decisionRequest{}
				if err = s.compactMemory(ctx, modelapi.InputBudget{}, capacity, nil); err != nil {
					return errors.Join(askErr, err)
				}
				compactionAttempted = true
				continue
			}
			if (!empty && malformed == nil) || correctedReply {
				return askErr
			}
			if s.usage.ModelCalls >= s.limits.MaxModelCalls {
				return errors.Join(askErr, errModelBudget)
			}
			correctedReply = true
			notice := diagcatalog.RunRetryInvalidToolCall
			s.checkpoint.Observation = "The previous response contained an invalid tool call. No operations from that response executed. Reconsider the next response from the user's requested outcome; a further tool call is appropriate only if it is still needed. Validation error: " + s.runner.Redact(askErr.Error())
			if empty {
				notice = diagcatalog.RunRetryEmptyReply
				s.checkpoint.Observation = "The previous decision ended without an answer or tool call. Provide the final answer or the native tool call needed for the current task."
			}
			if err := s.save(ctx, modelapi.Message{Role: "system", Content: s.checkpoint.Observation}); err != nil {
				return err
			}
			if err := s.emit(runrecord.Event{Kind: "retry", Text: diagcatalog.Text(notice)}); err != nil {
				return err
			}
		} else if len(reply.Calls) > 0 {
			if err = s.checkInstructions(ctx, decision.instructionRevision); err != nil {
				var failure *faults.Failure
				if errors.As(err, &failure) && failure.Code == diagcatalog.InstructionChanged {
					if err = s.returnToolFeedback(ctx, reply, "A new user instruction arrived before execution. Nothing in this response executed. Reconsider the candidates using the latest user request.", nil); err != nil {
						return err
					}
					continue
				}
				return err
			}

			calls, problems := s.validateCodeCalls(reply.Calls)
			if problems != nil {
				if err := s.rejectInvalidCalls(ctx, reply, problems, argumentCorrections); err != nil {
					return err
				}
				argumentCorrections++
				continue
			}
			argumentCorrections = 0
			callCtx := ctx
			if s.webFetchAllowance >= 0 {
				callCtx = toolapi.WithWebFetchDecision(ctx, s.webFetchAllowance, s.webFetchUserInput)
			}
			err = s.codeActions(callCtx, reply, calls, decision.supplied, approve)
			if err != nil {
				return err
			}

		} else {
			if changed, err := s.continueContext(ctx, reply, decision.supplied); err != nil {
				return err
			} else if changed {
				continue
			}
			if queued, err := s.continueSteered(ctx, reply, decision.supplied); err != nil {
				return err
			} else if queued {
				continue
			}
			if decision.commandRevision != s.commandRevision {
				continue
			}
			if pending, pendingErr := s.pendingCommands(ctx); pendingErr != nil {
				return pendingErr
			} else if pending {
				continue
			}
			if err := s.finish(ctx, reply, decision.supplied); err != nil {
				return err
			}
			argumentCorrections = 0
			if s.runner.OnAnswer == nil {
				return nil
			}
			s.projection.ReleaseObservations()
			decision = decisionRequest{}
			again, err := s.runner.OnAnswer(ctx, s.result.Text)
			if err != nil || !again {
				return err
			}
			s.result.Text = ""
			s.checkpoint.Status = "running"
		}
	}
}
