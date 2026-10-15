// Purpose: Hide machine edit anchors in tool displays while preserving source text and stored records.
// Pattern: Pure presentation projection.
// Usage: Formatter.Tool projects known read output and edit-argument fields before formatting.
// Related: internal/tui/transcript/tools.go, internal/agent/tool/anchor.go.
package transcript

import (
	"encoding/json"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"regexp"
	"strconv"
	"strings"
)

var numberedAnchor = regexp.MustCompile(`(?m)^([1-9][0-9]*)#[0-9a-f]{8}( \| )`)

func displayArguments(call toolapi.Call) json.RawMessage {
	if call.Name != "edit_file" {
		return call.Arguments
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(call.Arguments, &fields) != nil || fields == nil {
		return call.Arguments
	}
	for _, pair := range [][2]string{{"start_anchor", "start_line"}, {"end_anchor", "end_line"}} {
		raw, ok := fields[pair[0]]
		if !ok {
			continue
		}
		delete(fields, pair[0])
		fields[pair[1]] = json.RawMessage(`"invalid anchor"`)
		var text string
		if json.Unmarshal(raw, &text) != nil {
			continue
		}
		anchor, err := toolapi.ParseAnchor(text)
		if err == nil {
			fields[pair[1]] = json.RawMessage(strconv.Itoa(anchor.Line))
		}
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return call.Arguments
	}
	return encoded
}

func displayOutput(name string, body json.RawMessage) json.RawMessage {
	if name != "read_file" {
		return body
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return body
	}
	var content string
	if json.Unmarshal(fields["Content"], &content) != nil {
		return body
	}
	// Only the leading tool-generated prefix is removed. Hash-like source text after it stays literal.
	content = numberedAnchor.ReplaceAllString(content, "${1}${2}")
	content = alignLineNumbers(content)
	encoded, err := json.Marshal(content)
	if err != nil {
		return body
	}
	fields["Content"] = encoded
	delete(fields, "Note") // This field instructs the model how to copy internal edit anchors.
	encoded, err = json.Marshal(fields)
	if err != nil {
		return body
	}
	return encoded
}

var numberedLine = regexp.MustCompile(`^([1-9][0-9]*) \| (.*)$`)

func alignLineNumbers(content string) string {
	lines := strings.Split(content, "\n")
	width := 3
	for _, line := range lines {
		if parts := numberedLine.FindStringSubmatch(line); parts != nil {
			width = max(width, len(parts[1]))
		}
	}
	for i, line := range lines {
		if parts := numberedLine.FindStringSubmatch(line); parts != nil {
			lines[i] = fmt.Sprintf("%*s | %s", width, parts[1], parts[2])
		}
	}
	return strings.Join(lines, "\n")
}
