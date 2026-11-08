// Purpose: Traverse validated JSON without copying nested objects and arrays into decoder buffers.
// Pattern: Bounded wire-format parser.
// Usage: parse validates the complete UTF-8 JSON document before these private projections run.
// Related: internal/serverapi/openai/input.go, internal/serverapi/openai/messages.go.
package openai

import (
	"bytes"
	"encoding/json"

	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func jsonString(raw []byte, value *string) bool {
	if len(raw) < 2 || raw[0] != '"' {
		return false
	}
	if !bytes.ContainsRune(raw, '\\') {
		*value = string(raw[1 : len(raw)-1])
		return true
	}
	return json.Unmarshal(raw, value) == nil
}

// rawEnd receives one value within a document already checked by json.Valid.
func rawEnd(raw []byte, start int) int {
	depth := 0
	for i := start; i < len(raw); i++ {
		switch raw[i] {
		case '"':
			for i++; i < len(raw); i++ {
				if raw[i] == '\\' {
					i++
				} else if raw[i] == '"' {
					break
				}
			}
			if depth == 0 {
				return i + 1
			}
		case '[', '{':
			depth++
		case ']', '}':
			depth--
			if depth <= 0 {
				if depth == 0 {
					return i + 1
				}
				return i
			}
		case ',', ' ', '\n', '\r', '\t':
			if depth == 0 {
				return i
			}
		}
	}
	return len(raw)
}

func skipSpace(raw []byte, i int) int {
	for i < len(raw) && (raw[i] == ' ' || raw[i] == '\n' || raw[i] == '\r' || raw[i] == '\t') {
		i++
	}
	return i
}

type jsonObject struct {
	raw   []byte
	index int
}

func newObject(raw []byte) (jsonObject, bool) {
	raw = bytes.TrimSpace(raw)
	return jsonObject{raw: raw, index: 1}, len(raw) > 1 && raw[0] == '{'
}

func (o *jsonObject) next() (string, json.RawMessage, bool) {
	i := skipSpace(o.raw, o.index)
	if i >= len(o.raw)-1 {
		return "", nil, false
	}
	end := rawEnd(o.raw, i)
	key := jsonKey(o.raw[i:end])
	i = skipSpace(o.raw, skipSpace(o.raw, end)+1)
	end = rawEnd(o.raw, i)
	value := o.raw[i:end]
	o.index = skipSpace(o.raw, end)
	if o.index < len(o.raw) && o.raw[o.index] == ',' {
		o.index++
	}
	return key, value, true
}

func jsonKey(raw []byte) string {
	switch string(raw) {
	case `"role"`:
		return "role"
	case `"content"`:
		return "content"
	case `"type"`:
		return "type"
	case `"text"`:
		return "text"
	}
	var key string
	jsonString(raw, &key)
	return key
}

func object(raw []byte, param string) (map[string]json.RawMessage, error) {
	object, ok := newObject(raw)
	if !ok {
		return nil, faults.Invalid(param, diagcatalog.OpenaiExpectedAJSONObject)
	}
	fields := make(map[string]json.RawMessage, 4)
	for {
		key, value, ok := object.next()
		if !ok {
			break
		}
		if fields[key] != nil {
			return nil, faults.Invalid(param, diagcatalog.OpenaiDuplicateJSONField)
		}
		if len(fields) == 16 {
			return nil, faults.Invalid(param, diagcatalog.OpenaiTooManyObjectFields)
		}
		fields[key] = value
	}
	return fields, nil
}

func known(fields map[string]json.RawMessage, param string, names ...string) error {
	unknown := ""
	foundUnknown := false
	for key := range fields {
		found := false
		for _, name := range names {
			if key == name {
				found = true
				break
			}
		}
		if !found && (!foundUnknown || key < unknown) {
			unknown, foundUnknown = key, true
		}
	}
	if foundUnknown {
		return prefixError(faults.Invalid(unknown, diagcatalog.OpenaiThisParameterIsNotSupportedByMECA), param)
	}
	return nil
}

func prefixError(err error, prefix string) error {
	if value, ok := err.(*faults.InputError); ok && prefix != "" {
		if value.Param != "" {
			if value.Param[0] != '[' {
				prefix += "."
			}
			prefix += value.Param
		}
		return faults.InputParam(prefix, value.Cause)
	}
	return err
}

type array struct {
	raw   []byte
	index int
}

func jsonArray(raw []byte) (array, bool) {
	raw = bytes.TrimSpace(raw)
	return array{raw, 1}, len(raw) > 1 && raw[0] == '['
}
func (a *array) next() []byte {
	a.index = skipSpace(a.raw, a.index)
	if a.index >= len(a.raw)-1 {
		return nil
	}
	end := rawEnd(a.raw, a.index)
	value := a.raw[a.index:end]
	a.index = skipSpace(a.raw, end)
	if a.index < len(a.raw) && a.raw[a.index] == ',' {
		a.index++
	}
	return value
}
