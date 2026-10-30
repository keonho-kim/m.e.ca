// Purpose: Show compact image metadata without rendering or exposing binary request data.
// Pattern: Pure presentation projection.
// Usage: Both live and restored read_image tool results use imageOutput.
// Related: internal/tui/transcript/details.go, internal/tools/local/image.go.
package transcript

import (
	"encoding/json"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"strings"
)

func (f Formatter) imageOutput(event runrecord.Event, body []byte) toolOutput {
	if event.Kind != "tool_succeeded" {
		return toolOutput{}
	}
	var info struct {
		Format, MIME   string
		Width, Height  int
		Representative bool
	}
	if json.Unmarshal(body, &info) != nil || info.Width < 1 || info.Height < 1 {
		return toolOutput{}
	}
	text := indentTool(f.Language.Text("Image loaded")) + indentTool(f.Language.Format("%s · %d × %d", strings.ToUpper(info.Format), info.Width, info.Height))
	if info.Representative {
		text += indentTool(f.Language.Text("Representative first frame / page"))
	}
	return toolOutput{text: text}
}
