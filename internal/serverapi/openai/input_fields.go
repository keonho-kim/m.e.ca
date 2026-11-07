// Purpose: Parse the fixed message and content-part fields without per-item maps.
// Pattern: Typed strict input projection.
// Usage: messages.go reads validated JSON and rejects unknown or repeated fields at this boundary.
// Related: internal/serverapi/openai/json.go, internal/serverapi/openai/messages.go.
package openai

import "encoding/json"

type messageInput struct{ role, content, kind json.RawMessage }
type partInput struct{ kind, text json.RawMessage }

func assignInput(target *json.RawMessage, raw json.RawMessage, name string) error {
	if *target != nil {
		return invalid(name, "duplicate JSON field")
	}
	*target = raw
	return nil
}

func messageFields(raw []byte, responses bool) (messageInput, error) {
	object, ok := newObject(raw)
	if !ok {
		return messageInput{}, invalid("", "expected a JSON object")
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
			return fields, invalid(key, "this parameter is not supported by MECA")
		}
		if err := assignInput(target, value, key); err != nil {
			return fields, err
		}
	}
}

func partFields(raw []byte) (partInput, error) {
	object, ok := newObject(raw)
	if !ok {
		return partInput{}, invalid("", "expected a JSON object")
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
			return fields, invalid(key, "this parameter is not supported by MECA")
		}
		if err := assignInput(target, value, key); err != nil {
			return fields, err
		}
	}
}
