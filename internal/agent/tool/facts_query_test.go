// Purpose: Keep observation-only queries from copying unchanged accumulated file facts.
// Pattern: State and allocation regression tests.
// Usage: Run TestQueryFacts and BenchmarkQueryFacts.
// Related: internal/agent/tool/facts_observation.go, internal/agent/run/observations.go.
package tool

import (
	"fmt"
	"testing"
)

func TestQueryFactsPreservesBackingStorage(t *testing.T) {
	facts := Facts{Files: []FileFact{{Path: "a.go", Version: "v", Observation: "present"}}}
	before := &facts.Files[0]
	if facts.Observe(Operation{Name: "list_command_runs"}) || &facts.Files[0] != before {
		t.Fatal("query replaced unchanged file observations")
	}
	if allocs := testing.AllocsPerRun(10, func() { facts.Observe(Operation{Name: "list_command_runs"}) }); allocs != 0 {
		t.Fatalf("query allocated %g objects", allocs)
	}
}

func TestFileFactsUpdateInPlaceAndMergeNewPaths(t *testing.T) {
	facts := Facts{Files: []FileFact{{Path: "b.go", Version: "old", Changed: true, Observation: "present"}, {Path: "z.go", Version: "v", Observation: "present"}}}
	first := &facts.Files[0]
	facts.Observe(Operation{CallID: "read", RunID: "run_main", Versions: map[string]string{"b.go": "new"}})
	if &facts.Files[0] != first || facts.Files[0].Version != "new" || !facts.Files[0].Changed {
		t.Fatal("single update rebuilt or reset observed state")
	}
	facts.Observe(Operation{CallID: "write", RunID: "run_main", Versions: map[string]string{"a.go": "1", "c.go": "2", "zz.go": "3"}})
	for i, want := range []string{"a.go", "b.go", "c.go", "z.go", "zz.go"} {
		if facts.Files[i].Path != want {
			t.Fatalf("merge reordered facts: %+v", facts.Files)
		}
	}
}

func BenchmarkQueryFacts(b *testing.B) {
	for _, count := range []int{1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			facts := Facts{}
			for i := range count {
				facts.Files = append(facts.Files, FileFact{Path: fmt.Sprintf("file_%06d.go", i), Version: "v", Observation: "present"})
			}
			b.ReportAllocs()
			for b.Loop() {
				facts.Observe(Operation{Name: "list_command_runs"})
			}
		})
	}
}
