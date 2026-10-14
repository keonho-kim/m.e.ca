// Purpose: Merge runtime-reported task states and exact changed-file facts without model inference.
// Pattern: Pure projection.
// Usage: Compaction merges each archived source record into its evidence ledger.
// Related: internal/agent/compact/types.go, internal/agent/compact/controller.go.
package compact

func (l *Ledger) merge(record Record) {
	if record.Goal != nil {
		fact := GoalFact{State: *record.Goal, Source: record.Sequence}
		found := false
		for i := range l.Goals {
			if l.Goals[i].State.ID == fact.State.ID {
				l.Goals[i] = fact
				found = true
				break
			}
		}
		if !found {
			l.Goals = append(l.Goals, fact)
		}
	}
	if record.Review != nil {
		l.Reviews = append(l.Reviews, ReviewFact{Result: *record.Review, Source: record.Sequence})
	}
	if record.Grounding != nil {
		l.Groundings = append(l.Groundings, GroundingFact{Record: *record.Grounding, Source: record.Sequence})
	}
	if record.Task != nil {
		fact := *record.Task
		fact.Source = record.Sequence
		for i := range l.Tasks {
			if l.Tasks[i].ID == fact.ID {
				l.Tasks[i] = fact
				return
			}
		}
		l.Tasks = append(l.Tasks, fact)
	}
	if record.Operation != nil {
		op := *record.Operation
		op.Source = record.Sequence
		l.Operations = append(l.Operations, op)
	}
}
