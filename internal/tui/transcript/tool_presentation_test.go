// Purpose: Verify all tool families show readable results and aligned source positions.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/transcript -run TestToolPresentation.
// Related: internal/tui/transcript/details.go, internal/tui/transcript/anchors.go.
package transcript

import (
	"bytes"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestToolPresentationResults(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return strings.ReplaceAll(s, "secret", "[redacted]") }}
	for _, tc := range []struct {
		name, body string
		want       []string
	}{
		{"search_files", `{"matches":[{"path":"src/a.go"},{"path":"src/b.go"}],"truncated":true}`, []string{"파일 2개", "src/a.go", "src/b.go", "일부"}},
		{"grep", `{"matches":[{"path":"src/a.go","line":1,"text":"first"},{"path":"src/a.go","line":10,"text":"second"},{"path":"src/a.go","line":100,"text":"secret"}],"truncated":false}`, []string{"일치 3건", "  1 | first", " 10 | second", "100 | [redacted]"}},
		{"search_files", `{"matches":null,"truncated":false}`, []string{"검색 결과 없음"}},
		{"list_directory", `{"entries":[{"name":"src","directory":true},{"name":"a.go","directory":false}],"more":false,"next_offset":200}`, []string{"항목 2개", "src/", "a.go"}},
		{"write_file", `{"path":"a","changed":true}`, []string{"파일 저장 완료"}},
		{"edit_file", `{"path":"a","changed":true,"lines":{"OldStartLine":10,"OldEndLine":12,"NewStartLine":10,"NewEndLine":11}}`, []string{"수정 완료", "기존 10–12줄 → 10–11줄"}},
		{"edit_file", `{"path":"a","changed":true,"lines":{"OldStartLine":10,"OldEndLine":12,"Removed":true}}`, []string{"삭제한 줄: 10–12"}},
		{"delete_file", `{"path":"a","changed":true}`, []string{"삭제 완료"}},
		{"create_directory", `{"path":"a","changed":true}`, []string{"폴더 생성 완료"}},
		{"move_file", `{"path":"a","changed":true}`, []string{"이동 완료"}},
		{"execute_command", `{"Stdout":"hello","Stderr":"warning","ExitCode":0,"Status":"exited","Truncated":false}`, []string{"hello", "stderr: warning", "exit: 0"}},
	} {
		t.Run(tc.name+tc.body, func(t *testing.T) {
			raw := json.RawMessage(tc.body)
			original := bytes.Clone(raw)
			event := run.Event{Kind: "tool_succeeded", Text: "Completed", Call: &tool.Call{Name: tc.name, Arguments: []byte(`{"path":"a","reason":"hidden intent"}`)}, Output: raw}
			shown := f.Tool(event)
			for _, want := range tc.want {
				if !strings.Contains(shown, want) {
					t.Fatalf("missing %q: %s", want, shown)
				}
			}
			for _, hidden := range []string{"matches:", "entries:", "next_offset:", "NewStartLine", "hidden intent", "secret"} {
				if strings.Contains(shown, hidden) {
					t.Fatal(shown)
				}
			}
			if !bytes.Equal(raw, original) {
				t.Fatal("raw result changed")
			}
			restored := f.Event(event)
			if len(restored) != 1 || restored[0].Text != shown {
				t.Fatal("restored projection differs")
			}
		})
	}
}

func TestToolPresentationReadAlignsNumbers(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }}
	shown := f.Tool(run.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "read_file", Arguments: []byte(`{"path":"a"}`)}, Output: []byte(`{"Content":"1#12345678 | one\n10#12345678 | ten\n100#12345678 | hundred"}`)})
	position := -1
	for _, line := range strings.Split(shown, "\n") {
		if at := strings.Index(line, " | "); at >= 0 {
			if position >= 0 && at != position {
				t.Fatal(shown)
			}
			position = at
		}
	}
	if position < 0 || strings.Contains(shown, "12345678") {
		t.Fatal(shown)
	}
}

func TestToolPresentationMalformedAndFailedResults(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }}
	for _, tc := range []struct{ name, body, kind string }{{"search_files", `{"matches":{"private":"raw"}}`, "tool_succeeded"}, {"edit_file", `{"changed":true}`, "tool_failed"}} {
		shown := f.Tool(run.Event{Kind: tc.kind, Text: "operation could not complete", Call: &tool.Call{Name: tc.name}, Output: []byte(tc.body)})
		if !strings.Contains(shown, "operation could not complete") || strings.Contains(shown, "private") || strings.Contains(shown, "수정 완료") {
			t.Fatal(shown)
		}
	}
}
