// Purpose: Preserve complete outputs and maximize UTF-8 excerpts within the shared fallback budget.
// Pattern: Functional JSON projection checks against standard-library encoding.
// Usage: go test ./internal/agent/run -run TestCodeFallback.
// Related: internal/agent/run/code_fallback.go, internal/agent/run/code_recovery_test.go.
package run

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func fallbackTextJSON(t *testing.T, text string) json.RawMessage {
	t.Helper()
	var body bytes.Buffer
	encoder := json.NewEncoder(&body)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(map[string]string{"text": text}); err != nil {
		t.Fatal(err)
	}
	return body.Bytes()
}

func TestCodeFallbackPreservesCompleteResultsWithinBudget(t *testing.T) {
	const budget = 512 << 10
	empty, _ := json.Marshal(codeObservation{Tool: "inspect_fixture", Value: json.RawMessage(`""`)})
	exact := json.RawMessage(`"` + strings.Repeat("a", budget-2-len(empty)) + `"`)
	for _, test := range []struct {
		name string
		body json.RawMessage
	}{
		{"large", fallbackTextJSON(t, strings.Repeat("a", 300<<10)+"END_OF_RESULT")},
		{"escaped", fallbackTextJSON(t, strings.Repeat("한글<&>\u2028\u2029\"\\\t\n", 5000))},
		{"whitespace", json.RawMessage(strings.Repeat(" \n\t", 200<<10) + `{"done":true}`)},
		{"exact boundary", exact},
	} {
		t.Run(test.name, func(t *testing.T) {
			observation := codeObservation{Tool: "inspect_fixture", Value: test.body}
			want, err := json.Marshal([]codeObservation{observation})
			if err != nil || len(want) > budget {
				t.Fatalf("invalid complete-output fixture: bytes=%d err=%v", len(want), err)
			}
			var fallback codeFallback
			if err := fallback.retain(observation); err != nil {
				t.Fatal(err)
			}
			got, err := json.Marshal(fallback.Outputs)
			if err != nil || fallback.Truncated || !bytes.Equal(got, want) {
				t.Fatalf("complete result was not preserved: got=%d want=%d truncated=%t err=%v", len(got), len(want), fallback.Truncated, err)
			}
		})
	}
}

func TestCodeFallbackUsesRemainingBudgetForAnExcerpt(t *testing.T) {
	const budget = 512 << 10
	for _, sample := range []string{"plain text ", "한글🙂<&>\u2028\u2029\"\\\t\n"} {
		t.Run(sample, func(t *testing.T) {
			first := codeObservation{Tool: "first", Value: fallbackTextJSON(t, strings.Repeat("x", 100<<10))}
			second := codeObservation{Tool: "second", Value: fallbackTextJSON(t, strings.Repeat(sample, 100<<10))}
			var fallback codeFallback
			for _, output := range []codeObservation{first, second} {
				if err := fallback.retain(output); err != nil {
					t.Fatal(err)
				}
			}
			body, err := json.Marshal(fallback.Outputs)
			if err != nil || !fallback.Truncated || len(body) > budget || len(fallback.Outputs) != 2 {
				t.Fatalf("incorrect shared output limit: bytes=%d count=%d truncated=%t err=%v", len(body), len(fallback.Outputs), fallback.Truncated, err)
			}
			var outputs []codeObservation
			if err := json.Unmarshal(body, &outputs); err != nil {
				t.Fatal(err)
			}
			wantFirst, _ := json.Marshal(first)
			if outputs[0].Tool != "first" || outputs[1].Tool != "second" || !bytes.Equal(fallback.Outputs[0], wantFirst) {
				t.Fatal("later output replaced or reordered a complete result")
			}
			var excerpt struct {
				Excerpt   string `json:"excerpt"`
				Truncated bool   `json:"truncated"`
			}
			if err := json.Unmarshal(outputs[1].Value, &excerpt); err != nil || !excerpt.Truncated || !utf8.ValidString(excerpt.Excerpt) || !bytes.HasPrefix(second.Value, []byte(excerpt.Excerpt)) {
				t.Fatalf("excerpt did not preserve original UTF-8 text: %v", err)
			}
			// The next complete character must exceed the wire budget, rather
			// than leaving usable room because of a worst-case escape estimate.
			remaining := second.Value[len(excerpt.Excerpt):]
			if len(remaining) == 0 {
				t.Fatal("complete output was marked truncated")
			}
			_, size := utf8.DecodeRune(remaining)
			excerpt.Excerpt += string(remaining[:size])
			outputs[1].Value, _ = json.Marshal(excerpt)
			expanded, err := json.Marshal(outputs)
			if err != nil || len(expanded) <= budget {
				t.Fatalf("usable output space was discarded: expanded=%d budget=%d err=%v", len(expanded), budget, err)
			}
			if err := fallback.retain(codeObservation{Tool: "third", Value: json.RawMessage(`true`)}); err != nil {
				t.Fatal(err)
			}
			after, _ := json.Marshal(fallback.Outputs)
			if !bytes.Equal(after, body) {
				t.Fatal("exhausted budget replaced or enlarged retained outputs")
			}
		})
	}
}
