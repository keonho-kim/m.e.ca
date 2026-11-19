// Purpose: Verify compaction avoids serializing equivalent sources and preserves selection semantics.
// Pattern: Source selection behavior and allocation tests.
// Usage: go test ./internal/agent/compact -run TestSource.
// Related: internal/agent/compact/source.go, internal/agent/compact/source_bench_test.go.
package compact

import (
	"strings"
	"testing"
)

func TestSourceDuplicateSkipsEncoding(t *testing.T) {
	sources := sourceBuffer{limit: 1 << 20}
	text := strings.Repeat("Same redacted observation.\n", 700)
	redact := func(string) string { return text }
	if err := sources.add(Record{Role: "observation", Text: "original one"}, redact); err != nil {
		t.Fatal(err)
	}
	duplicate := Record{Role: "observation", Text: "original two"}
	allocations := testing.AllocsPerRun(16, func() {
		if err := sources.add(duplicate, redact); err != nil {
			t.Fatal(err)
		}
	})
	if allocations != 0 || len(sources.packet()) != 1 || sources.omitted != 0 {
		t.Fatalf("duplicate allocations=%g entries=%d omitted=%d", allocations, sources.entries.Len(), sources.omitted)
	}
}

func TestSourceDuplicateRespectsEviction(t *testing.T) {
	sources := sourceBuffer{limit: 512}
	redact := func(s string) string { return s }
	note := Record{Role: "observation", Text: strings.Repeat("a", 80)}
	for _, record := range []Record{
		{Role: "failure", Text: strings.Repeat("b", 200)},
		note,
		{Role: "observation", Text: strings.Repeat("x", 350)},
		note,
	} {
		if err := sources.add(record, redact); err != nil {
			t.Fatal(err)
		}
	}
	if sources.entries.Len() != 2 || !strings.Contains(string(sources.packet()[1]), note.Text) || sources.omitted != 2 {
		t.Fatalf("evicted observation was treated as a duplicate: entries=%d omitted=%d", sources.entries.Len(), sources.omitted)
	}
}

func TestSourceDuplicatesPreserveClippingAndUserRecords(t *testing.T) {
	sources := sourceBuffer{limit: 1024}
	redact := func(s string) string { return s }
	head, tail := strings.Repeat("<", 600), strings.Repeat(">", 600)
	for _, record := range []Record{
		{Role: "observation", Text: head + "different middle one" + tail},
		{Role: "observation", Text: head + "different middle two" + tail},
		{Role: "user", Text: "Keep this user constraint."},
		{Role: "user", Text: "Keep this user constraint."},
	} {
		if err := sources.add(record, redact); err != nil {
			t.Fatal(err)
		}
	}
	packet := sources.packet()
	if len(packet) != 3 || sources.omitted != 0 || string(packet[1]) != string(packet[2]) {
		t.Fatalf("clipping or user repetition changed: entries=%d omitted=%d", len(packet), sources.omitted)
	}
}

func TestSourceSameEndingDoesNotHideChanges(t *testing.T) {
	sources := sourceBuffer{limit: 1024}
	for _, text := range []string{"one status, same ending", "two status, same ending"} {
		if err := sources.add(Record{Role: "observation", Text: text}, func(s string) string { return s }); err != nil {
			t.Fatal(err)
		}
	}
	if sources.entries.Len() != 2 {
		t.Fatal("distinct sources with a common ending were collapsed")
	}
}
