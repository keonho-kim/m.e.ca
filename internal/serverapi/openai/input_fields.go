// Purpose: Parse the fixed message and content-part fields without per-item maps.
// Pattern: Typed strict input projection.
// Usage: messages.go reads validated JSON and rejects unknown or repeated fields at this boundary.
// Related: internal/serverapi/openai/json.go, internal/serverapi/openai/messages.go.
package openai

import (
	"encoding/json"

	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

type messageInput struct{ role, content, kind json.RawMessage }
type partInput struct{ kind, text json.RawMessage }

func assignInput(target *json.RawMessage, raw json.RawMessage, name string) error {
	if *target != nil {
		return faults.Invalid(name, diagcatalog.OpenaiDuplicateJSONField)
	}
	*target = raw
	return nil
}

func messageFields(raw []byte, responses bool) (messageInput, error) {
	object, ok := newObject(raw)
	if !ok {
		return messageInput{}, faults.Invalid("", diagcatalog.OpenaiExpectedAJSONObject)
	}
	var fields messageInput
	for {
		key, value, ok := object.next()
		if !ok {
			return fields, nil
		}
		var target *json.RawMessage
		switch key {
		case "role":
			target = &fields.role
		case "content":
			target = &fields.content
		case "type":
			if responses {
				target = &fields.kind
			}
		}
		if target == nil {
			return fields, faults.Invalid(key, diagcatalog.OpenaiThisParameterIsNotSupportedByMECA)
		}
		if err := assignInput(target, value, key); err != nil {
			return fields, err
		}
	}
}

func partFields(raw []byte) (partInput, error) {
	object, ok := newObject(raw)
	if !ok {
		return partInput{}, faults.Invalid("", diagcatalog.OpenaiExpectedAJSONObject)
	}
	var fields partInput
	for {
		key, value, ok := object.next()
		if !ok {
			return fields, nil
		}
		var target *json.RawMessage
		switch key {
		case "type":
			target = &fields.kind
		case "text":
			target = &fields.text
		}
		if target == nil {
			return fields, faults.Invalid(key, diagcatalog.OpenaiThisParameterIsNotSupportedByMECA)
		}
		if err := assignInput(target, value, key); err != nil {
			return fields, err
		}
	}
}
