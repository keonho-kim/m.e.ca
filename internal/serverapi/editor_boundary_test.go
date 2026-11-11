// Purpose: Verify full-sized editable documents survive Connect JSON response encoding.
// Pattern: Connect JSON response boundary regression test.
// Usage: go test ./internal/serverapi -run TestEditorConnectJSONBoundary.
// Related: internal/serverapi/rpc.go, internal/fileview/editor.go.
package serverapi

import (
	"bytes"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestEditorConnectJSONBoundary(t *testing.T) {
	f := newFixture(t, nil, false)
	transport := nativeTransport(t, f, "grpc")
	text := strings.Repeat(`"`, 2<<20)
	if err := os.WriteFile(filepath.Join(f.project, "boundary.txt"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	body, err := protojson.Marshal(&v1.Command{Operation: &v1.Command_EditorRead{EditorRead: &v1.EditFile{Path: "boundary.txt"}}})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequestWithContext(deadline(t), "POST", "http://localhost:8787/meca.v1.MecaService/Call", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+testToken)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Connect-Protocol-Version", "1")
	response := httptest.NewRecorder()
	transport.ServeHTTP(response, request)
	var reply v1.Reply
	if err := protojson.Unmarshal(response.Body.Bytes(), &reply); err != nil {
		t.Fatalf("2MiB editor response failed: status=%d bodyBytes=%d error=%v", response.Code, response.Body.Len(), err)
	}
	if response.Code != 200 || reply.GetEditor().GetText() != text {
		t.Fatalf("2MiB editor response mismatch: status=%d textBytes=%d editable=%v reason=%q error=%v", response.Code, len(reply.GetEditor().GetText()), reply.GetEditor().GetEditable(), reply.GetEditor().GetReason(), reply.GetError())
	}
}
