// Purpose: Merge runtime-reported task states and exact changed-file facts without model inference.
// Pattern: Pure projection.
// Usage: Compaction merges each archived source record into its evidence ledger.
// Related: internal/agent/compact/types.go, internal/agent/compact/controller.go.
package compact

func (l *Ledger) merge(record Record) {
	if record.Review != nil {
		l.Reviews = append(l.Reviews, ReviewFact{Result: *record.Review, Source: record.Sequence})
	}
	if record.Grounding != nil {
		l.Groundings = append(l.Groundings, GroundingFact{Record: *record.Grounding, Source: record.Sequence})
	}
	if record.Run != nil {
		fact := *record.Run
		fact.Source = record.Sequence
		for i := range l.Runs {
			if l.Runs[i].ID == fact.ID {
				l.Runs[i] = fact
				return
			}
		}
		l.Runs = append(l.Runs, fact)
	}
	if record.Operation != nil {
		op := *record.Operation
		op.Source = record.Sequence
		l.Operations = append(l.Operations, op)
	}
}
