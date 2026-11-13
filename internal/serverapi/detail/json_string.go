// Purpose: Decode and mask JSON strings in bounded chunks without a second full-sized string.
// Pattern: Streaming scalar adapter preserving UTF-8 and escaped surrogate boundaries.
// Usage: Raw detail serialization and semantic text fields share this decoder.
// Related: internal/serverapi/detail/write_string.go, internal/serverapi/detail/view.go.
package detail

import (
	"context"
	"encoding/json"
	"io"
	"unicode/utf8"
)

func walkJSONString(ctx context.Context, raw []byte, emit func(string, bool) error) error {
	// The caller has validated the complete JSON scalar.
	for at := 1; at < len(raw)-1; {
		if err := ctx.Err(); err != nil {
			return err
		}
		start := at
		for at < len(raw)-1 && at-start < 8<<10 {
			if raw[at] == '\\' {
				if raw[at+1] == 'u' {
					high := (raw[at+2] == 'd' || raw[at+2] == 'D') && (raw[at+3] == '8' || raw[at+3] == '9' || raw[at+3] == 'a' || raw[at+3] == 'A' || raw[at+3] == 'b' || raw[at+3] == 'B')
					at += 6
					// Include an adjacent Unicode escape so a surrogate pair
					// cannot be split between independently decoded chunks.
					if high && at+6 < len(raw) && raw[at] == '\\' && raw[at+1] == 'u' {
						at += 6
					}
				} else {
					at += 2
				}
			} else {
				_, size := utf8.DecodeRune(raw[at:])
				at += size
			}
		}
		part := make([]byte, 0, at-start+2)
		part = append(part, '"')
		part = append(part, raw[start:at]...)
		part = append(part, '"')
		var text string
		if err := json.Unmarshal(part, &text); err != nil {
			return err
		}
		if err := emit(text, false); err != nil {
			return err
		}
	}
	return emit("", true)
}

func writeJSONText(ctx context.Context, output io.Writer, raw []byte, newMask func() func(string, bool) string) error {
	mask := newMask()
	return walkJSONString(ctx, raw, func(text string, final bool) error {
		_, err := io.WriteString(output, mask(text, final))
		return err
	})
}
