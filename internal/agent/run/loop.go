// Purpose: Advance native model decisions, recovery and effects until a final response is ready.
// Pattern: Serial decision loop over one prepared execution.
// Usage: Runner.Run calls drive after conversation admission; owned cleanup remains outside the loop.
// Related: internal/agent/run/run.go, internal/agent/run/action.go, internal/agent/run/finish.go.
package run

import (
	"context"
	"errors"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func (s *execution) drive(ctx context.Context, approve Approver) error {
	var err error
	for {
		if _, err = s.collectCommands(ctx); err != nil {
			return err
		}
		if _, err = s.admitContext(ctx); err != nil {
			return err
		}
		if _, err = s.steer(ctx); err != nil {
			return err
		}
		if finished, decisionErr := s.prepareDecision(ctx); finished || decisionErr != nil {
			return decisionErr
		}
		s.checkpoint.Iteration = s.usage.ToolCalls
		s.checkpoint.ModelCalls = s.usage.ModelCalls
		s.checkpoint.Phase = "decision"
		s.checkpoint.Tool = ""
		if err = s.persist(ctx); err != nil {
			return err
		}
		if err = s.refreshMemory(ctx, false); err != nil {
			return err
		}
		// Compaction can take time; admit instructions received during it before
		// generating another answer or proposing effects from the earlier request.
		if changed, steerErr := s.steer(ctx); steerErr != nil {
			return steerErr
		} else if changed {
			if err = s.persist(ctx); err != nil {
				return err
			}
		}
		if _, err = s.admitContext(ctx); err != nil {
			return err
		}
		commandRevision := s.commandRevision
		reply, askErr := s.askResponse(ctx)
		if askErr == nil {
			s.capacityRecovery = false
		}
		if askErr != nil {
			if handled, imageErr := s.recoverImages(ctx, askErr); handled {
				if imageErr != nil {
					return imageErr
				}
				continue
			}
			var capacity *modelapi.ContextLimitError
			if !s.capacityRecovery && errors.As(askErr, &capacity) {
				s.capacityRecovery = true
				if m := s.runner.Memory; m != nil && capacity.Capacity > 0 && (m.ContextWindow == 0 || capacity.Capacity < m.ContextWindow) {
					m.ContextWindow = capacity.Capacity
				}
				if err = s.refreshMemory(ctx, true); err != nil {
					return errors.Join(askErr, err)
				}
				continue
			}
			var malformed *modelapi.ToolCallError
			var truncated *modelapi.OutputLimitError
			if !errors.As(askErr, &malformed) && !errors.As(askErr, &truncated) {
				return askErr
			}
			s.result.HadToolFailure = true
			s.checkpoint.Observation = "Model response failed before execution: " + s.runner.Redact(askErr.Error())
			if err := s.save(ctx, modelapi.Message{Role: "system", Content: s.checkpoint.Observation}); err != nil {
				return err
			}
		} else if len(reply.Calls) > 0 {
			if s.runner.PollContext != nil {
				objective := s.checkpoint.Objective
				if _, err = s.admitContext(ctx); err != nil {
					return err
				}
				if objective != s.checkpoint.Objective {
					if err = s.returnToolFeedback(ctx, reply, "A new user instruction arrived before execution. Nothing in this response executed. Reconsider the candidates using the latest user request."); err != nil {
						return err
					}
					continue
				}
			}

			if len(reply.Calls) > 1 {
				err = s.batch(ctx, reply, approve)
			} else {
				err = s.action(ctx, reply, approve)
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
			if commandRevision != s.commandRevision {
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
