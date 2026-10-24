// Purpose: Distinguish known server defaults and mandatory states from unknown toggle capabilities.
// Pattern: Pure model-capability tests.
// Usage: go test ./internal/provider/connection -run TestReasoningDefault.
// Related: internal/provider/connection/reasoning.go, internal/tui/session/model_form.go.
package connection

import "testing"

func TestReasoningDefault(t *testing.T) {
	on, off := true, false
	for _, test := range []struct {
		name    string
		options Reasoning
		want    *bool
	}{
		{"unknown", Reasoning{On: true, Off: true}, nil},
		{"declared on", Reasoning{On: true, Off: true, DefaultOn: &on}, &on},
		{"declared off", Reasoning{On: true, Off: true, DefaultOn: &off}, &off},
		{"mandatory", Reasoning{On: true}, &on},
		{"mandatory level", Reasoning{Levels: []string{"Light", "Deep"}}, &on},
		{"default level", Reasoning{Off: true, Levels: []string{"low"}, Default: "low"}, &on},
		{"off only", Reasoning{Off: true}, &off},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.options.Check(); err != nil {
				t.Fatal(err)
			}
			got := test.options.KnownDefault()
			if (got == nil) != (test.want == nil) || got != nil && *got != *test.want {
				t.Fatalf("got=%v want=%v", got, test.want)
			}
		})
	}
	if err := (Reasoning{On: true, DefaultOn: &off}).Check(); err == nil {
		t.Fatal("mandatory reasoning advertised an OFF default")
	}
}
