// Purpose: Reduce live tool, participant and usage state for bounded reconnect snapshots.
// Pattern: Request-owned state projection.
// Usage: emit records state under the service mutex; snapshots copy immutable public values.
// Related: internal/commands/server/projection.go, internal/commands/server/snapshot.go.
package server

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	"strings"
	"time"
)

type usageSample struct {
	value          modelapi.TokenUsage
	measuredMillis int64
}

func (r *request) observe(value record.Event) {
	if value.Kind == "team_notice" && value.Text == "left" {
		for key := range r.activity {
			if strings.HasPrefix(key, value.Agent+"/") {
				delete(r.activity, key)
			}
		}
	}
	if r.liveUsage == nil {
		r.liveUsage = make(map[string]usageSample)
	}
	if value.Kind == "usage_progress" && value.Usage != nil {
		r.liveUsage[value.Agent] = usageSample{value: value.Usage.Clone(), measuredMillis: value.MeasuredDuration.Milliseconds()}
	}
	if value.Kind == "usage" && value.Usage != nil {
		r.completed = r.completed.Add(*value.Usage)
		delete(r.liveUsage, value.Agent)
		if value.Generation != nil {
			r.generationMillis += value.Generation.MeasuredMillis
		}
	}
	if value.Kind == "generation_finished" {
		delete(r.liveUsage, value.Agent)
	}
	key := ""
	switch value.Kind {
	case "tool_started":
		id := value.CallID
		if value.Call != nil {
			id = value.Call.ID
		}
		key = value.Agent + "/tool/" + id
	case "tool_succeeded", "tool_failed", "tool_interrupted", "denied":
		id := value.CallID
		if value.Call != nil {
			id = value.Call.ID
		}
		delete(r.activity, value.Agent+"/tool/"+id)
	case "team_notice", "team_work", "progress", "team_progress", "retry", "compaction_started", "compaction_finished":
		key = value.Agent + "/" + value.Kind
	}
	if key == "" {
		return
	}
	if r.activity == nil {
		r.activity = make(map[string]Event)
	}
	if _, exists := r.activity[key]; !exists && len(r.activity) >= 128 {
		return
	}
	value.Text, value.Thinking = clipped(value.Text, 1024), ""
	projected := ProjectEvent(Event{Task: r.task.ID, Session: r.task.Session, Record: value})
	if eventBytes(projected) > 8192 {
		projected.Record.Call = nil
		projected.Truncated = true
	}
	r.activity[key] = projected
}

func (r *request) statistics() record.Statistics {
	usage := r.completed.Clone()
	generationMillis := r.generationMillis
	for _, sample := range r.liveUsage {
		usage = usage.Add(sample.value)
		generationMillis += sample.measuredMillis
	}
	elapsed := int64(0)
	if !r.started.IsZero() {
		elapsed = time.Since(r.started).Milliseconds()
	}
	return record.Statistics{Usage: &usage, ElapsedMillis: elapsed, GenerationMillis: generationMillis, Concurrent: r.teamEnabled}
}
