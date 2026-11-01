// Purpose: Verify repeated LAT evidence is shared while original native input bytes restore exactly.
// Pattern: Storage and integrity regression tests.
// Usage: Run TestLATInputFragments.
// Related: internal/storage/sqlite/lat_input.go, internal/storage/sqlite/payload_fragments.go.
package sqlite

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/lat"
	"github.com/keonho-kim/meca/internal/agent/lat/transcript"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestLATInputFragmentsShareEvidenceAndPreserveExactSource(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	random := rand.New(rand.NewPCG(1, 2))
	source := make([]byte, 36*1024)
	for i := range source {
		source[i] = byte(random.Uint32())
	}
	evidence := json.RawMessage(`{"observation":"` + base64.StdEncoding.EncodeToString(source) + `","label":"<html> 한글 😀"}`)
	var originals []string
	for _, phase := range []string{"plan", "critique", "refine", "review"} {
		frame, err := lat.Prepare(lat.Input{Prompt: "Inspect files.", Objective: "Implement the requested behavior.", Root: "/project", Plan: phase, ContextWindow: 262144})
		if err != nil {
			t.Fatal(err)
		}
		packet, err := frame.Build(evidence)
		if err != nil {
			t.Fatal(err)
		}
		originals = append(originals, packet.Content)
		journal, err := transcript.New(s, id, "run_main", phase)
		if err != nil {
			t.Fatal(err)
		}
		message := modelapi.Message{Role: "user", Content: packet.Content}
		if err := journal.Commit(ctx, id, []runrecord.Entry{{Message: &message}}); err != nil {
			t.Fatal(err)
		}
		journal.Close()
	}
	var count, fragments, stored int
	if err := s.db.QueryRow("SELECT COUNT(*),SUM(codec=3),SUM(length(body)) FROM payloads").Scan(&count, &fragments, &stored); err != nil {
		t.Fatal(err)
	}
	if count > 10 || fragments != 4 {
		t.Fatalf("unexpected fragments: payloads=%d manifests=%d", count, fragments)
	}
	unshared := 0
	for _, original := range originals {
		raw, err := json.Marshal(original)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := s.codec.Encode(raw)
		if err != nil {
			t.Fatal(err)
		}
		unshared += min(len(raw), len(encoded))
	}
	if stored >= unshared {
		t.Fatalf("shared input used more space: shared=%d whole=%d", stored, unshared)
	}
	t.Logf("four phase inputs: shared payloads=%d bytes; independent whole payloads=%d bytes", stored, unshared)
	page, err := s.recordPage(ctx, "SELECT sequence,'event',body,refs,raw_bytes FROM tool_events ORDER BY sequence")
	if err != nil {
		t.Fatal(err)
	}
	for i, record := range page {
		body, err := s.unpackRecord(ctx, record, nil)
		if err != nil {
			t.Fatal(err)
		}
		var event runrecord.Event
		if err := json.Unmarshal(body, &event); err != nil {
			t.Fatal(err)
		}
		if event.Message.Content != originals[i] {
			t.Fatal("native input changed on hydration")
		}
	}
	var manifest string
	if err := s.db.QueryRow("SELECT id FROM payloads WHERE codec=3 LIMIT 1").Scan(&manifest); err != nil {
		t.Fatal(err)
	}
	cycle, _ := json.Marshal([]string{manifest, manifest, manifest})
	if _, err := s.db.Exec("UPDATE payloads SET body=? WHERE id=?", cycle, manifest); err != nil {
		t.Fatal(err)
	}
	if _, err := s.payload(ctx, manifest); err == nil {
		t.Fatal("recursive text fragments accepted")
	}
}

func TestLATInputKeepsEfficientWholeCompression(t *testing.T) {
	s, err := newTestStore(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx, id := context.Background(), s.Session().ID
	frame, err := lat.Prepare(lat.Input{Prompt: "Inspect files.", Objective: "Implement request.", ContextWindow: 262144})
	if err != nil {
		t.Fatal(err)
	}
	packet, err := frame.Build(json.RawMessage(`{"evidence":"` + strings.Repeat("same observation ", 3000) + `"}`))
	if err != nil {
		t.Fatal(err)
	}
	j, err := transcript.New(s, id, "run_main", "plan")
	if err != nil {
		t.Fatal(err)
	}
	defer j.Close()
	m := modelapi.Message{Role: "user", Content: packet.Content}
	if err := j.Commit(ctx, id, []runrecord.Entry{{Message: &m}}); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM payloads WHERE codec=3").Scan(&count); err != nil || count != 0 {
		t.Fatalf("compression regression: manifests=%d error=%v", count, err)
	}
}
