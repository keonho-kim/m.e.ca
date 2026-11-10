// Purpose: Escape and mask detail strings in bounded chunks while preserving raw JSON numbers.
// Pattern: Streaming JSON string projection.
// Usage: documentWriter handles Go strings and raw tool JSON through these methods.
// Related: internal/serverapi/detail/write.go, internal/redaction/chunks.go.
package detail

import (
	"encoding/json"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (w *documentWriter) quotedPart(text string) error {
	if text == "" {
		return nil
	}
	if w.encoder == nil {
		w.encoder = json.NewEncoder(&w.quoted)
		w.encoder.SetEscapeHTML(false)
	}
	w.quoted.Reset()
	if err := w.encoder.Encode(text); err != nil {
		return err
	}
	body := w.quoted.Bytes()
	_, err := w.output.Write(body[1 : len(body)-2])
	return err
}

func (w *documentWriter) text(text string) error {
	if err := w.literal("\""); err != nil {
		return err
	}
	mask := w.newMask()
	for len(text) > 0 {
		if err := w.ctx.Err(); err != nil {
			return err
		}
		n := min(len(text), 8<<10)
		if err := w.quotedPart(mask(text[:n], false)); err != nil {
			return err
		}
		text = text[n:]
	}
	if err := w.quotedPart(mask("", true)); err != nil {
		return err
	}
	return w.literal("\"")
}

func (w *documentWriter) byteText(body []byte) error {
	if err := w.literal("\""); err != nil {
		return err
	}
	mask := w.newMask()
	for len(body) > 0 {
		if err := w.ctx.Err(); err != nil {
			return err
		}
		n := min(len(body), 8<<10)
		if err := w.quotedPart(mask(string(body[:n]), false)); err != nil {
			return err
		}
		body = body[n:]
	}
	if err := w.quotedPart(mask("", true)); err != nil {
		return err
	}
	return w.literal("\"")
}

func (w *documentWriter) raw(body []byte) error {
	if len(body) == 0 {
		return w.literal("null")
	}
	if !utf8.Valid(body) || !json.Valid(body) {
		return faults.Errorf(diagcatalog.RedactionInvalidCompleteUTFJSON)
	}
	copied := 0
	for i := 0; i < len(body); i++ {
		if body[i] != '"' {
			continue
		}
		start, escaped := i, false
		for i++; i < len(body); i++ {
			if body[i] == '\\' {
				escaped = true
				i++
			} else if body[i] == '"' {
				break
			}
		}
		if _, err := w.output.Write(body[copied:start]); err != nil {
			return err
		}
		if escaped {
			var text string
			if err := json.Unmarshal(body[start:i+1], &text); err != nil {
				return err
			}
			if err := w.text(text); err != nil {
				return err
			}
		} else if err := w.byteText(body[start+1 : i]); err != nil {
			return err
		}
		copied = i + 1
	}
	_, err := w.output.Write(body[copied:])
	return err
}
