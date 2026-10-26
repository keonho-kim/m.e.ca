// Purpose: Verify all tool families show readable results and aligned source positions.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/transcript -run TestToolPresentation.
// Related: internal/tui/transcript/details.go, internal/tui/transcript/process_result.go.
package transcript

import (
	"bytes"
	"encoding/json"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"strings"
	"testing"
)

func TestToolPresentationResults(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return strings.ReplaceAll(s, "secret", "[redacted]") }}
	for _, tc := range []struct {
		name, body string
		want       []string
	}{
		{"write_file", `{"path":"a","changed":true}`, []string{"파일 저장 완료"}},
		{"edit_file", `{"path":"a","changed":true,"lines":[{"OldStartLine":10,"OldEndLine":12,"NewStartLine":10,"NewEndLine":11}]}`, []string{"수정 완료", "기존 10–12줄 → 10–11줄"}},
		{"edit_file", `{"path":"a","changed":true,"lines":[{"OldStartLine":10,"OldEndLine":12,"Removed":true}]}`, []string{"삭제한 줄: 10–12"}},
		{"create_directory", `{"created":["a","a/b"],"changed":true}`, []string{"폴더 생성 완료", "2개", "a/b"}},
		{"execute_command", `{"Stdout":"hello","Stderr":"warning","ExitCode":0,"Status":"exited","Truncated":false}`, []string{"hello", "표준 오류: warning", "실행 성공"}},
	} {
		t.Run(tc.name+tc.body, func(t *testing.T) {
			raw := json.RawMessage(tc.body)
			original := bytes.Clone(raw)
			event := runrecord.Event{Kind: "tool_succeeded", Text: "Completed", Call: &tool.Call{Name: tc.name, Arguments: []byte(`{"path":"a"}`)}, Output: raw}
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
	shown := f.Tool(runrecord.Event{Kind: "tool_succeeded", Call: &tool.Call{Name: "execute_command", Arguments: []byte(`{"command":"nl a"}`)}, Output: []byte(`{"Stdout":"  1 | one\n 10 | ten\n100 | hundred\n","ExitCode":0,"Status":"exited"}`)})
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
	for _, tc := range []struct{ name, body, kind string }{{"execute_command", `{"Stdout":{"private":"raw"}}`, "tool_succeeded"}, {"edit_file", `{"changed":true}`, "tool_failed"}} {
		shown := f.Tool(runrecord.Event{Kind: tc.kind, Text: "operation could not complete", Call: &tool.Call{Name: tc.name}, Output: []byte(tc.body)})
		if !strings.Contains(shown, "operation could not complete") || strings.Contains(shown, "private") || strings.Contains(shown, "수정 완료") {
			t.Fatal(shown)
		}
	}
}

func TestToolPresentationDirectoryPartialResult(t *testing.T) {
	f := Formatter{Redact: func(s string) string { return s }}
	shown := f.Tool(runrecord.Event{Kind: "tool_failed", Text: "creation interrupted", Call: &tool.Call{Name: "create_directory", Arguments: []byte(`{"paths":["src/app","tests"]}`)}, Output: []byte(`{"changed":true,"created":["src"]}`)})
	for _, want := range []string{"경로: src/app", "경로: tests", "creation interrupted", "중단 전 생성한 폴더", "src"} {
		if !strings.Contains(shown, want) {
			t.Fatalf("missing %q: %s", want, shown)
		}
	}
}
