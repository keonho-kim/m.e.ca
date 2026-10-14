// Purpose: Analyze one user request into hidden Goal Tasks and execute them serially to verified completion.
// Pattern: Goal orchestration use case.
// Usage: Ordinary session submissions call RunGoal; explicit Plan execution keeps its approval workflow.
// Related: internal/agent/goal/state.go, internal/agent/run/goal_model.go, internal/agent/run/run.go.
package run

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/goal"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/monitor"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"strings"
	"time"
)

func (r Runner) RunGoal(ctx context.Context, input string, approve Approver, emit func(Event) error) (result Result, err error) {
	if r.Generator == nil || r.Tools == nil || r.Journal == nil || r.Redact == nil || approve == nil || emit == nil {
		return result, fmt.Errorf("Goal dependencies are required")
	}
	if len(input) > 64*1024 {
		return result, fmt.Errorf("objective exceeds 64 KiB")
	}
	if err := r.Settings.Check(); err != nil {
		return result, err
	}
	if r.Inference != nil {
		if err := r.Inference.Check(); err != nil {
			return result, err
		}
	}
	limits := r.Limits
	if limits == (Limits{}) {
		limits = DefaultLimits()
	}
	if err := limits.Check(); err != nil {
		return result, err
	}
	r.Limits = limits
	if r.Usage == nil {
		r.Usage = &Usage{}
	}
	if r.Usage.deadline.IsZero() {
		r.Usage.deadline = time.Now().Add(time.Duration(limits.MaxSeconds) * time.Second)
	}
	if r.observer == nil {
		r.observer, err = r.startMonitor(ctx)
		if err != nil {
			return result, err
		}
		defer r.observer.Close()
	}
	templates := r.templates()
	base := templates.System
	system, err := prompt.Compose(base, r.ProjectInstructions, false)
	if err != nil {
		return result, err
	}
	id := "goal_" + rand.Text()
	phase := &task{runner: r, limits: limits, system: system, usage: r.Usage, emit: emit}
	phase.checkpoint = Checkpoint{ID: id + "_analysis", GoalID: id, Objective: r.Redact(input), Status: Running, Phase: "goal", InstructionsDigest: fmt.Sprintf("%x", sha256.Sum256([]byte(r.ProjectInstructions)))}
	phase.runner.suppressStream = true
	if err := phase.save(ctx, modelapi.Message{Role: "user", Content: input}); err != nil {
		return result, err
	}
	if err := r.observer.Observe(ctx, monitor.Record{GoalID: id, Kind: "goal_requested", Detail: r.Redact(input)}); err != nil {
		return result, err
	}
	state, err := phase.analyzeGoal(ctx, id, r.Redact(input))
	if err != nil {
		return result, err
	}
	defer func() { err = r.finishGoalError(ctx, state, err) }()
	if err = r.recordGoal(ctx, state); err != nil {
		return result, err
	}
	var reports []string
	hadToolFailure := false
	for index := range state.Tasks {
		if err = state.Start(index); err != nil {
			return result, err
		}
		if err = r.recordGoal(ctx, state); err != nil {
			return result, err
		}
		assignment := state.Tasks[index]
		child := r
		child.assignment = &goal.Assignment{GoalID: id, TaskID: assignment.ID, GoalObjective: state.Request, TaskObjective: assignment.Objective, Kind: assignment.Kind}
		child.internalTask = true
		child.suppressStream = len(state.Tasks) > 1
		result, err = child.Run(ctx, assignment.Objective, false, approve, func(event Event) error {
			if event.Kind == "response" {
				return nil
			}
			return emit(event)
		})
		hadToolFailure = hadToolFailure || result.HadToolFailure
		result.HadToolFailure = hadToolFailure
		if finishErr := state.Finish(index, string(result.Status), result.Text, result.Evidence); finishErr != nil {
			return result, errors.Join(err, finishErr)
		}
		if err != nil {
			return result, err
		}
		if state.Tasks[index].Status != "completed" {
			if err = r.recordGoal(ctx, state); err != nil {
				return result, err
			}
			if result.Status != NeedsInput {
				return result, fmt.Errorf("work could not be verified; no completion was recorded")
			}
			return result, phase.publishGoalReply(ctx, result.Text)
		}
		reports = append(reports, result.Text)
		// Final user reporting is part of Goal completion; child completion alone is not.
		if state.Status == "completed" {
			state.Status = "running"
		}
		if err = r.recordGoal(ctx, state); err != nil {
			return result, err
		}
	}
	result.Text = strings.Join(reports, "\n\n")
	if len(reports) > 1 {
		result.Text, err = phase.summarizeGoal(ctx, state)
		if err != nil {
			return result, err
		}
	}
	state.Status = "completed"
	result.Status = Answered
	for _, task := range state.Tasks {
		if task.Kind == "work" {
			result.Status = Verified
			result.Evidence = "All required work Tasks have current verification evidence"
		}
	}
	// Save the user report before recording completion so a crash cannot leave
	// a completed Goal with no durable final answer.
	if err = phase.save(ctx, modelapi.Message{Role: "assistant", Content: result.Text}); err != nil {
		return result, err
	}
	if err = r.recordGoal(ctx, state); err != nil {
		return result, err
	}
	err = emit(Event{Kind: "response", Text: result.Text})
	return result, err
}

func (r Runner) recordGoal(ctx context.Context, state goal.State) error {
	if err := state.Check(); err != nil {
		return err
	}
	if err := r.Journal.RecordEvent(ctx, r.Session, Event{Kind: "goal_state", Goal: &state, Internal: true}); err != nil {
		return err
	}
	detail := state.Objective + "; status=" + state.Status
	for _, task := range state.Tasks {
		detail += "\n" + task.ID + ": " + task.Status + "; " + task.Objective
	}
	return r.observer.Observe(ctx, monitor.Record{GoalID: state.ID, Kind: "goal_state", Detail: detail})
}

// A failed final report is also an incomplete Goal, even after child Tasks finished.
func (r Runner) finishGoalError(ctx context.Context, state goal.State, cause error) error {
	if cause == nil {
		return nil
	}
	state.Status = "stopped"
	if errors.Is(cause, context.Canceled) || errors.Is(cause, context.DeadlineExceeded) {
		state.Status = "interrupted"
	}
	state.Tasks = append([]goal.Task(nil), state.Tasks...)
	for i := range state.Tasks {
		if state.Tasks[i].Status == "running" {
			state.Tasks[i].Status = state.Status
		}
	}
	flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return errors.Join(cause, r.recordGoal(flush, state))
}
