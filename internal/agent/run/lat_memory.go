// Purpose: Coordinate one background review-memory job without sharing main compaction state.
// Pattern: Request-owned resource lifecycle and serial persistence handoff.
// Usage: Tool and model boundaries poll receipts; final review or shutdown joins the job.
// Related: internal/agent/lat/memory/controller.go, internal/agent/run/lat_memory_usage.go.
package run

import (
	"context"
	"errors"
	"time"

	"github.com/keonho-kim/meca/internal/agent/inference"
	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	"github.com/keonho-kim/meca/internal/agent/prompt"
)

type reviewMemory struct {
	ctx        context.Context
	stop       context.CancelFunc
	controller latmemory.Controller
	failure    error
	closed     bool
	dirty      bool
}

func (s *execution) activeLATMemory() bool {
	return s.latMemory != nil && s.latMemory.controller.Active()
}

func (s *execution) maintainLATMemory(ctx context.Context, force bool) error {
	if s.latMemory == nil || s.latMemory.closed || !s.latObserved || s.latReviewed && !force || ctx.Err() != nil {
		return nil
	}
	if err := s.pollLATMemory(ctx); err != nil {
		return err
	}
	m := s.latMemory
	if m.failure != nil || m.controller.Active() {
		return nil
	}
	if !force && !m.dirty {
		return nil
	}
	parameters, err := inference.Resolve(s.runner.Inference, inference.LATCompaction)
	if err != nil {
		return s.failLATMemory(ctx, err)
	}
	m.dirty = false
	instructions := prompt.Defaults().LATCompaction
	if s.runner.Prompts != nil && s.runner.Prompts.LATCompaction != "" {
		instructions = s.runner.Prompts.LATCompaction
	}
	plan, err := m.controller.Prepare(ctx, s.runner.Redact(instructions), s.latWindow(), parameters.MaxOutputTokens, force)
	if err != nil {
		return s.failLATMemory(ctx, err)
	}
	if plan.Through == 0 {
		return nil
	}
	plan.Request.ModelID = s.runner.Model
	plan.Request.Temperature = &parameters.Temperature
	plan.Request.Thinking, plan.Request.ReasoningEffort = s.runner.Thinking, s.runner.ReasoningEffort
	plan.Request = parameters.ApplyThinking(plan.Request, s.runner.ThinkingSupported)
	plan.Request.IdleTimeout = time.Duration(s.limits.MaxModelSeconds) * time.Second
	// Leave one call available to the foreground decision or final response.
	budget, limit := s.usage, s.limits.MaxModelCalls-1
	owner := m.ctx
	err = m.controller.Start(owner, plan, s.reviewGenerator, func() bool { return owner.Err() == nil && budget.model(limit) })
	if err != nil {
		return s.failLATMemory(ctx, err)
	}
	return nil
}

func (s *execution) pollLATMemory(ctx context.Context) error {
	if s.latMemory == nil {
		return nil
	}
	update, compactErr := s.latMemory.controller.Poll(ctx)
	if err := s.recordLATMemoryUsage(ctx, update); err != nil {
		return err
	}
	if compactErr != nil && !errors.Is(compactErr, context.Canceled) {
		return s.failLATMemory(ctx, compactErr)
	}
	return nil
}

func (s *execution) failLATMemory(ctx context.Context, failure error) error {
	if s.latMemory.failure != nil {
		return nil
	}
	s.latMemory.failure = failure
	return s.reviewEvent(ctx, "lat_memory_unavailable", s.runner.Redact(failure.Error()))
}

func (s *execution) joinLATMemory(ctx context.Context) error {
	if !s.activeLATMemory() {
		return nil
	}
	s.latMemory.controller.Cancel()
	<-s.latMemory.controller.Done()
	flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return s.pollLATMemory(flush)
}

func (s *execution) closeLATMemory(ctx context.Context) error {
	if s.latMemory == nil {
		return nil
	}
	s.latMemory.closed = true
	defer s.latMemory.stop()
	return s.joinLATMemory(ctx)
}

func (s *execution) latWindow() int {
	window := s.runner.ContextWindow
	if m := s.runner.Memory; m != nil && m.ContextWindow > 0 && (window == 0 || m.ContextWindow < window) {
		window = m.ContextWindow
	}
	return window
}
