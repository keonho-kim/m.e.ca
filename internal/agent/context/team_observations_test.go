// Purpose: Keep current contribution facts fresh and separate from retained conversation history.
// Pattern: Context ownership regression test.
// Usage: Context tests replace work observations across ordinary and summarized requests.
// Related: internal/agent/context/request.go, internal/agent/context/projection.go.
package context

import (
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestCurrentTeamWorkReplacesOnlyTransientContext(t *testing.T) {
	history := []modelapi.Message{{Role: "user", Content: "Build the service"}}
	var projection Projection
	in := Input{System: "rules", Messages: history, Memory: "Earlier completed work", Objective: history[0].Content, MaxBytes: 1 << 20, Projection: &projection, RedactionVersion: func() int { return 0 }, Redact: func(s string) string { return s }, Observations: "Hana owns api/"}
	first, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	in.Observations = "Ren owns tests/"
	second, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(first.Messages[len(first.Messages)-1].Content, "Hana") || !strings.Contains(second.Messages[len(second.Messages)-1].Content, "Ren") {
		t.Fatal("work observations mutated or stayed stale")
	}
	if len(history) != 1 || history[0].Content != "Build the service" || len(projection.entries) != 1 {
		t.Fatal("working state accumulated in history cache")
	}
}

func TestTeamWorkProjectionReusesRedactionAndInvalidatesCredentialChanges(t *testing.T) {
	var projection Projection
	version, scans := 0, 0
	in := Input{System: "rules", MaxBytes: 1 << 20, Projection: &projection, Observations: "Hana owns api/", RedactionVersion: func() int { return version }}
	in.Redact = func(text string) string {
		if text == in.Observations {
			scans++
		}
		return text
	}
	for range 3 {
		if _, err := Build(in); err != nil {
			t.Fatal(err)
		}
	}
	if scans != 1 {
		t.Fatal("unchanged observations were rescanned", scans)
	}
	version++
	if _, err := Build(in); err != nil || scans != 2 {
		t.Fatal("credential change reused the old sanitized view", scans, err)
	}
	in.Observations = ""
	if _, err := Build(in); err != nil || projection.work.source != "" || projection.work.text != "" {
		t.Fatal("removed observations retained their prior snapshot", err)
	}
}
