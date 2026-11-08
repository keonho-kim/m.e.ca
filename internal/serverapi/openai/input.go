// Purpose: Parse a bounded, explicit text-only subset of OpenAI request envelopes.
// Pattern: Untrusted JSON input boundary.
// Usage: Chat and Responses handlers parse supported fields before atomic engine submission.
// Related: internal/serverapi/openai/messages.go, internal/commands/server/submission.go.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"github.com/keonho-kim/meca/internal/commands/server"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

type parsed struct {
	input                server.Submission
	stream, includeUsage bool
}

func stringField(fields map[string]json.RawMessage, name string, optional bool) (string, error) {
	raw, ok := fields[name]
	if !ok && optional {
		return "", nil
	}
	return requiredString(raw, name)
}

func requiredString(raw []byte, name string) (string, error) {
	// The common protocol labels already have immutable program-owned
	// strings. Avoid allocating another copy for every message or text part.
	switch string(raw) {
	case `"user"`:
		return "user", nil
	case `"assistant"`:
		return "assistant", nil
	case `"system"`:
		return "system", nil
	case `"developer"`:
		return "developer", nil
	case `"message"`:
		return "message", nil
	case `"text"`:
		return "text", nil
	case `"input_text"`:
		return "input_text", nil
	case `"output_text"`:
		return "output_text", nil
	case `"meca"`:
		return "meca", nil
	}
	var value string
	if !jsonString(raw, &value) {
		return "", faults.Invalid(name, diagcatalog.OpenaiExpectedAUTFString)
	}
	return value, nil
}
func boolField(fields map[string]json.RawMessage, name string, fallback bool) (bool, error) {
	raw, ok := fields[name]
	if !ok {
		return fallback, nil
	}
	if !bytes.Equal(raw, []byte("true")) && !bytes.Equal(raw, []byte("false")) {
		return false, faults.Invalid(name, diagcatalog.OpenaiExpectedABoolean)
	}
	return bytes.Equal(raw, []byte("true")), nil
}

func parse(r *http.Request, w http.ResponseWriter, kind string) (parsed, error) {
	defer r.Body.Close()
	if err := r.Context().Err(); err != nil {
		return parsed{}, err
	}
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return parsed{}, faults.Invalid("", diagcatalog.OpenaiContentTypeMustBeApplicationJson)
	}
	body, err := readInput(w, r)
	if err != nil && faults.Code(err) == diagcatalog.ResourceExhausted {
		return parsed{}, err
	}
	if err != nil || !utf8.Valid(body) || !json.Valid(body) {
		return parsed{}, faults.Invalid("", diagcatalog.OpenaiInvalidOrOversizedUTFJSONBody)
	}
	if err := r.Context().Err(); err != nil {
		return parsed{}, err
	}
	fields, err := object(body, "")
	if err != nil {
		return parsed{}, err
	}
	model, err := stringField(fields, "model", false)
	if err != nil {
		return parsed{}, err
	}
	if model != "meca" {
		return parsed{}, faults.Invalid("model", diagcatalog.OpenaiTheOnlyExposedModelIsMeca)
	}
	p := parsed{input: server.Submission{Kind: kind, Stored: true}}
	keys := r.Header.Values("Idempotency-Key")
	if len(keys) > 1 || len(keys) == 1 && len(keys[0]) == 0 {
		return p, faults.Invalid("Idempotency-Key", diagcatalog.OpenaiProvideOneNonemptyRequestKey)
	}
	p.input.Key = r.Header.Get("Idempotency-Key")
	if len(p.input.Key) > 128 || strings.ContainsAny(p.input.Key, "\r\n\x00") || !utf8.ValidString(p.input.Key) {
		return p, faults.Invalid("Idempotency-Key", diagcatalog.OpenaiProvideOneKeyOfAtMostUTF)
	}
	p.stream, err = boolField(fields, "stream", false)
	if err != nil {
		return p, err
	}
	if kind == "chat" {
		return parseChat(r.Context(), fields, p)
	}
	return parseResponse(r.Context(), fields, p)
}

func readInput(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	const limit = 17 << 20
	if r.ContentLength > limit {
		return nil, faults.Invalid("", diagcatalog.OpenaiJSONBodyExceedsMiB)
	}
	reader := http.MaxBytesReader(w, r.Body, limit)
	if r.ContentLength <= 0 {
		return io.ReadAll(reader)
	}
	// HTTP already owns Content-Length framing. Allocate its bounded length
	// once, then reject truncated or unexpectedly longer test/custom bodies.
	body := make([]byte, int(r.ContentLength))
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, err
	}
	var extra [1]byte
	if _, err := io.ReadFull(reader, extra[:]); err != io.EOF {
		return nil, faults.Invalid("", diagcatalog.OpenaiJSONBodyLengthDoesNotMatchContent)
	}
	return body, nil
}

func parseChat(ctx context.Context, fields map[string]json.RawMessage, p parsed) (parsed, error) {
	if err := known(fields, "", "model", "messages", "stream", "n", "stream_options"); err != nil {
		return p, err
	}
	if raw, ok := fields["n"]; ok && string(raw) != "1" {
		return p, faults.Invalid("n", diagcatalog.OpenaiOnlyNIsSupported)
	}
	if raw, ok := fields["stream_options"]; ok {
		if !p.stream {
			return p, faults.Invalid("stream_options", diagcatalog.OpenaiStreamOptionsRequiresStreamTrue)
		}
		options, err := object(raw, "stream_options")
		if err != nil {
			return p, err
		}
		if err := known(options, "stream_options", "include_usage"); err != nil {
			return p, err
		}
		p.includeUsage, err = boolField(options, "include_usage", false)
		if err != nil {
			return p, faults.InputParam("stream_options.include_usage", err)
		}
	}
	var err error
	p.input.Messages, err = parseMessages(ctx, fields["messages"], "messages", false)
	return p, err
}

func parseResponse(ctx context.Context, fields map[string]json.RawMessage, p parsed) (parsed, error) {
	if err := known(fields, "", "model", "input", "instructions", "stream", "store", "background", "previous_response_id"); err != nil {
		return p, err
	}
	var err error
	p.input.Stored, err = boolField(fields, "store", true)
	if err != nil {
		return p, err
	}
	p.input.Background, err = boolField(fields, "background", false)
	if err != nil {
		return p, err
	}
	if p.input.Background && !p.input.Stored {
		return p, faults.Invalid("store", diagcatalog.OpenaiMECABackgroundResponsesRequireStoreTrue)
	}
	if raw := fields["previous_response_id"]; raw != nil && string(raw) != "null" {
		p.input.Previous, err = stringField(fields, "previous_response_id", true)
		if err != nil {
			return p, err
		}
	}
	p.input.Messages, err = parseMessages(ctx, fields["input"], "input", true)
	if err != nil {
		return p, err
	}
	if _, ok := fields["instructions"]; ok {
		instructions, err := stringField(fields, "instructions", true)
		if err != nil {
			return p, err
		}
		if instructions != "" {
			p.input.Messages = append([]server.InputMessage{{Role: "developer", Text: instructions}}, p.input.Messages...)
		}
	}
	return p, nil
}
