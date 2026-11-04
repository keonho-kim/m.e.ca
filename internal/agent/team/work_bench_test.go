// Purpose: Measure repeated team work projections without changing membership or contributions.
// Pattern: Request-scoped allocation benchmark.
// Usage: Compare current-work snapshot and covered-context allocations.
// Related: internal/agent/team/work.go, internal/agent/team/contribution/context.go.
package team

import (
	"fmt"
	"strings"
	"testing"

	model "github.com/keonho-kim/meca/internal/agent/model"
	record "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/contribution"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
)

func BenchmarkTeamWorkContext(b *testing.B) {
	r := &Room{active: make(map[string]*participant)}
	for i := range MaxActive {
		name := identity.Name(i)
		r.active[name] = &participant{}
		value, err := r.work.Prepare(contribution.Describe(contribution.Scope{Member: name, Summary: "Implement the independent contribution", Paths: []string{fmt.Sprintf("/project/part%d", i)}}))
		if err != nil {
			b.Fatal(err)
		}
		r.work.Commit(value, int64(i+1))
	}
	r.workSnapshot()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		r.workSnapshot()
	}
}

func BenchmarkTeamCoveredContext(b *testing.B) {
	r := &Room{active: make(map[string]*participant)}
	var history []model.Message
	for i := range MaxActive {
		name := identity.Name(i)
		r.active[name] = &participant{}
		summary, contract := strings.Repeat("summary ", 128), strings.Repeat("contract ", 128)
		candidate, err := r.work.Prepare(contribution.Describe(contribution.Scope{Member: name, Summary: summary, Contract: contract, Paths: []string{fmt.Sprintf("/project/part%d", i)}}))
		if err != nil {
			b.Fatal(err)
		}
		seq := int64(i + 1)
		r.work.Commit(candidate, seq)
		history = append(history, channel.Message{Sequence: seq, From: name, Text: summary, Declaration: true, Note: &record.TeamNote{Work: &record.TeamWork{Paths: candidate.Paths, Contract: contract, Revision: 1}}}.Context())
	}
	p := &Peer{ID: identity.Name(0), room: r, decisionView: r.workSnapshot()}
	full := p.ContextObservations(nil)
	reduced := p.ContextObservations(history)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.ContextObservations(history)
	}
	b.ReportMetric(float64(len(full)-len(reduced)), "context-bytes-saved")
}
