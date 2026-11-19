// Purpose: Advance native model decisions, recovery and effects until a final response is ready.
// Pattern: Serial decision loop over one prepared execution.
// Usage: Runner.Run calls drive after conversation admission; owned cleanup remains outside the loop.
// Related: internal/agent/run/run.go, internal/agent/run/action.go, internal/agent/run/finish.go.
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
	correctedToolCall := false
	for {
		if _, err = s.collectCommands(ctx); err != nil {
			return err
		}
		if _, err = s.admitInstructions(ctx); err != nil {
			return err
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
		decision, askErr := s.prepareAgentRequest(ctx)
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
			reply, askErr = s.askResponse(ctx, decision.request, decision.prepared.Send, &draft)
			decision.request, decision.prepared = modelapi.Request{}, modelapi.PreparedGeneration{}
			compactionAttempted = false
		}
		if askErr == nil {
			s.acknowledgeObservations(decision.supplied)
			s.capacityRecovery = false
			correctedToolCall = false
		}
		if askErr != nil {
			if ctx.Err() != nil || errors.Is(askErr, errRuntimeRecord) {
				return askErr
			}
			var malformed *modelapi.ToolCallError
			if errors.As(askErr, &malformed) {
				s.result.HadToolFailure = true
			}
			if malformed == nil && draft.output.bytes() != 0 {
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
			if malformed == nil || correctedToolCall {
				return askErr
			}
			correctedToolCall = true
			s.checkpoint.Observation = "Correct the native tool call using the declared schema and continue the task. Validation error: " + s.runner.Redact(askErr.Error())
			if err := s.save(ctx, modelapi.Message{Role: "system", Content: s.checkpoint.Observation}); err != nil {
				return err
			}
			if err := s.emit(runrecord.Event{Kind: "retry", Text: diagcatalog.Text(diagcatalog.RunRetryInvalidToolCall)}); err != nil {
				return err
			}
		} else if len(reply.Calls) > 0 {
			if err = s.checkInstructions(ctx, decision.instructionRevision); err != nil {
				var failure *faults.Failure
				if errors.As(err, &failure) && failure.Code == diagcatalog.InstructionChanged {
					if err = s.returnToolFeedback(ctx, reply, "A new user instruction arrived before execution. Nothing in this response executed. Reconsider the candidates using the latest user request."); err != nil {
						return err
					}
					continue
				}
				return err
			}

			callCtx := ctx
			if s.webFetchAllowance >= 0 {
				callCtx = toolapi.WithWebFetchDecision(ctx, s.webFetchAllowance, s.webFetchUserInput)
			}
			if len(reply.Calls) > 1 {
				err = s.batch(callCtx, reply, approve)
			} else {
				err = s.action(callCtx, reply, approve)
			}
			if err != nil {
				return err
			}

		} else {
			if changed, err := s.continueContext(ctx, reply); err != nil {
				return err
			} else if changed {
				continue
			}
			if queued, err := s.continueSteered(ctx, reply); err != nil {
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
			if err := s.finish(ctx, reply); err != nil {
				return err
			}
			if s.runner.OnAnswer == nil {
				return nil
			}
			again, err := s.runner.OnAnswer(ctx, s.result.Text)
			if err != nil || !again {
				return err
			}
			s.result.Text = ""
			s.checkpoint.Status = "running"
		}
	}
}
