// Purpose: Restore selected JSON fields from content references without depending on a database.
// Pattern: Record transformation with a caller-owned payload read callback.
// Usage: SQLite releases its result rows before resolving record references.
// Related: internal/storage/content/pack.go, internal/storage/sqlite/record_codec.go.
package content

import (
	"bytes"

	"strconv"

	json "github.com/goccy/go-json"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func Restore(body, refsBody []byte, fields map[string]bool, load func(string) (json.RawMessage, error)) ([]byte, error) {
	if fields == nil && bytes.Equal(refsBody, []byte("{}")) {
		if !json.Valid(body) {
			return nil, faults.Errorf(diagcatalog.ContentInvalidStoredRecordJSON)
		}
		return body, nil
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	if fields != nil {
		for key := range root {
			if !fields[key] {
				delete(root, key)
			}
		}
	}
	var refs map[string]string
	if err := json.Unmarshal(refsBody, &refs); err != nil {
		return nil, err
	}
	loaded := map[string]json.RawMessage{}
	patches := &restoreNode{}
	for key, id := range refs {
		id, quote := Reference(id)
		var path []string
		if err := json.Unmarshal([]byte(key), &path); err != nil || len(path) == 0 {
			return nil, faults.Errorf(diagcatalog.ContentInvalidStoredPayloadPath)
		}
		if fields != nil && !fields[path[0]] {
			continue
		}
		value := loaded[id]
		if value == nil {
			var err error
			value, err = load(id)
			if err != nil {
				return nil, err
			}
			loaded[id] = value
		}
		if quote {
			var err error
			value, err = json.Marshal(string(value))
			if err != nil {
				return nil, err
			}
		}
		if err := patches.add(path, value); err != nil {
			return nil, err
		}
	}
	for field, patch := range patches.children {
		raw, ok := root[field]
		if !ok {
			return nil, faults.Errorf(diagcatalog.ContentMissingStoredPayloadField)
		}
		restored, err := patch.apply(raw)
		if err != nil {
			return nil, err
		}
		root[field] = restored
	}
	return json.Marshal(root)
}

// Each ancestor is decoded and encoded once, regardless of its descendant count.
type restoreNode struct {
	value    json.RawMessage
	children map[string]*restoreNode
}

func (n *restoreNode) add(path []string, value json.RawMessage) error {
	for _, key := range path {
		if n.value != nil {
			return faults.Errorf(diagcatalog.ContentInvalidStoredPayloadPath)
		}
		if n.children == nil {
			n.children = make(map[string]*restoreNode)
		}
		if n.children[key] == nil {
			n.children[key] = &restoreNode{}
		}
		n = n.children[key]
	}
	if n.value != nil || len(n.children) > 0 || len(value) == 0 {
		return faults.Errorf(diagcatalog.ContentInvalidStoredPayloadPath)
	}
	n.value = value
	return nil
}

func (n *restoreNode) apply(raw json.RawMessage) (json.RawMessage, error) {
	if n.value != nil {
		if string(raw) != "null" {
			return nil, faults.Errorf(diagcatalog.ContentInvalidStoredPayloadPlaceholder)
		}
		return n.value, nil
	}
	if len(raw) == 0 {
		return nil, faults.Errorf(diagcatalog.ContentInvalidStoredPayloadPath)
	}
	if raw[0] == '{' {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, err
		}
		for key, patch := range n.children {
			child, ok := fields[key]
			if !ok {
				return nil, faults.Errorf(diagcatalog.ContentMissingStoredPayloadField)
			}
			next, err := patch.apply(child)
			if err != nil {
				return nil, err
			}
			fields[key] = next
		}
		return json.Marshal(fields)
	}
	if raw[0] == '[' {
		var values []json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return nil, err
		}
		for key, patch := range n.children {
			i, err := strconv.Atoi(key)
			if err != nil || i < 0 || i >= len(values) || strconv.Itoa(i) != key {
				return nil, faults.Errorf(diagcatalog.ContentInvalidStoredPayloadIndex)
			}
			next, err := patch.apply(values[i])
			if err != nil {
				return nil, err
			}
			values[i] = next
		}
		return json.Marshal(values)
	}
	return nil, faults.Errorf(diagcatalog.ContentInvalidStoredPayloadContainer)
}
