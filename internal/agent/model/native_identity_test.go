// Purpose: Preserve defensive mutation detection and invalidate owned identities on replacement.
// Pattern: Ownership boundary regression tests.
// Usage: go test ./internal/agent/model -run TestNativeIdentity.
// Related: internal/agent/model/native_identity.go, internal/provider/protocol/source_history.go.
package model

import (
	"bytes"
	"encoding/json"
	"testing"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func TestNativeIdentityTracksUnsealedBytesAndOwnedReplacements(t *testing.T) {
	m := Message{Role: "assistant", Calls: []toolapi.Call{{ID: "one", Name: "write_file", Arguments: []byte(`{"path":"a"}`)}}, ReasoningState: []byte(`{"state":1}`)}
	a, b := m.NativeFingerprint()
	m.Calls[0].Arguments[9] = 'b'
	changed, _ := m.NativeFingerprint()
	if changed == a {
		t.Fatal("unsealed byte mutation was trusted")
	}
	m = m.SealHistory()
	owned, state := m.NativeFingerprint()
	if owned != changed || state != b {
		t.Fatal("ownership changed the native values")
	}
	m.Calls[0].Arguments = []byte(`{"path":"c"}`)
	m.ReasoningState = nil
	next, cleared := m.NativeFingerprint()
	if next == owned || cleared == state {
		t.Fatal("replaced arguments or continuation reused a stale fingerprint")
	}
	raw, err := json.Marshal(m)
	if err != nil || bytes.Contains(raw, []byte("native")) || bytes.Contains(raw, []byte("identity")) {
		t.Fatal("transient identity leaked to storage", err)
	}
}
