// Purpose: Reject unknown fields and trailing values in non-Protobuf wire envelopes.
// Pattern: Strict standard-library boundary parser.
// Usage: Browser subscriptions and opaque search cursors decode through this contract.
// Related: internal/serverapi/ui_observer.go, internal/serverapi/history_search.go.
package serverapi

import (
	"bytes"
	"encoding/json"
	"io"
)

func decodeExactJSON(body []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		if err == nil {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	return nil
}
