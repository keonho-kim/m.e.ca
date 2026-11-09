// Purpose: Reject native command input before admission when the control byte budget is occupied.
// Pattern: Authenticated HTTP resource boundary regression test.
// Usage: go test ./internal/serverapi -run TestInputBudgetRejectsBeforeTaskAdmission.
// Related: internal/serverapi/transport.go, internal/serverapi/admission/input.go.
package serverapi

import "testing"

func TestInputBudgetRejectsBeforeTaskAdmission(t *testing.T) {
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := nativeTransport(t, f, "sse")
	release, ok := transport.inputs.Try(32 << 20)
	if !ok {
		t.Fatal("cannot occupy control budget")
	}
	defer release()
	reply := nativeRequest(t, transport, "POST", "/v1/command", `{"selectSession":{"create":true}}`)
	if reply.Code != 429 || f.app.State().Session.ID != "" {
		t.Fatal("control exceeded input admission", reply.Code, reply.Body.String())
	}
}
