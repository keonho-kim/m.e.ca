// Purpose: Verify localized image results remain compact in expanded and collapsed views.
// Pattern: Presentation regression tests.
// Usage: go test ./internal/tui/transcript -run TestImage.
// Related: internal/tui/transcript/image.go, internal/tui/locale/tools.go.
package transcript

import (
	"encoding/json"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"strings"
	"testing"
)

func TestImageMetadataPresentation(t *testing.T) {
	e := runrecord.Event{Kind: "tool_succeeded", Text: "Image loaded", Call: &toolapi.Call{ID: "read", Name: "read_image", Arguments: json.RawMessage(`{"path":"shot.gif"}`)}, Output: json.RawMessage(`{"Path":"shot.gif","Format":"gif","MIME":"image/png","Width":100,"Height":200,"Representative":true}`)}
	for _, language := range []locale.Language{locale.Korean, locale.English} {
		f := Formatter{Language: language, Redact: func(s string) string { return s }}
		for _, text := range []string{f.Tool(e), f.toolSummary(e)} {
			if !strings.Contains(text, "100 × 200") || !strings.Contains(text, "shot.gif") || strings.Contains(text, `"Width"`) {
				t.Fatalf("raw or incomplete image presentation: %s", text)
			}
		}
		if language == locale.Korean && !strings.Contains(f.Tool(e), "이미지 읽기") {
			t.Fatal("untranslated image result")
		}
	}
}
