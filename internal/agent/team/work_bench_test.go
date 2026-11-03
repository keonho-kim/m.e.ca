// Purpose: Measure repeated team work projections without changing membership or contributions.
// Pattern: Request-scoped allocation benchmark.
// Usage: Compare WorkContext allocations before and after projection changes.
// Related: internal/agent/team/work.go, internal/agent/team/contribution/context.go.
package team

import (
	"fmt"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/team/identity"
)

func BenchmarkTeamWorkContext(b *testing.B) {
	r := &Room{active: make(map[string]*participant)}
	for i := range MaxActive {
		name := identity.Name(i)
		r.active[name] = &participant{}
		if err := r.work.Set(name, "Implement the independent contribution", []string{fmt.Sprintf("/project/part%d", i)}); err != nil {
			b.Fatal(err)
		}
	}
	p := &Peer{ID: identity.Name(0), room: r}
	p.WorkContext()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		p.WorkContext()
	}
}
