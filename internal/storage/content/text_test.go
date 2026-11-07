// Purpose: Verify selective public text restoration and preservation of payload failures.
// Pattern: Deterministic content boundary tests.
// Usage: go test ./internal/storage/content -run TestText.
// Related: internal/storage/content/text.go, internal/storage/content/payload.go.
package content

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestTextRestoresOnlyCommittedContent(t *testing.T) {
	reads := 0
	load := func(id string) (json.RawMessage, error) {
		reads++
		if id != "public" {
			t.Fatal("loaded private payload", id)
		}
		return json.RawMessage(`"original\ntext"`), nil
	}
	text, err := Text([]byte(`{"content":null,"calls":null}`), []byte(`{"[\"content\"]":"public","[\"calls\"]":"private"}`), load)
	if err != nil || text != "original\ntext" || reads != 1 {
		t.Fatal(text, reads, err)
	}
	text, err = Text([]byte(`{"content":null,"incomplete":true}`), []byte(`{"[\"content\"]":"private"}`), load)
	if err != nil || text != "" || reads != 1 {
		t.Fatal(text, reads, err)
	}
	broken := errors.New("digest check failed")
	_, err = Text([]byte(`{"content":null}`), []byte(`{"[\"content\"]":"public"}`), func(string) (json.RawMessage, error) { return nil, broken })
	if !errors.Is(err, broken) {
		t.Fatal("payload integrity failure lost", err)
	}
	if _, err := Text([]byte(`{"content":"inline"}`), []byte(`{"[\"content\"]":"public"}`), load); err == nil {
		t.Fatal("non-placeholder content accepted")
	}
}
