// Purpose: Keep the server native-only and reject removed compatible endpoint routes on every transport.
// Pattern: Authenticated HTTP routing regression test.
// Usage: go test ./internal/serverapi -run TestRemovedCompatibleRoutes.
// Related: internal/serverapi/transport.go.
package serverapi

import "testing"

func TestRemovedCompatibleRoutes(t *testing.T) {
	for _, protocol := range []string{"sse", "socket", "grpc"} {
		t.Run(protocol, func(t *testing.T) {
			f := newFixture(t, generateFunc(idleGenerator), false)
			transport := nativeTransport(t, f, protocol)
			for _, route := range []struct{ method, path string }{{"GET", "/v1/models"}, {"GET", "/v1/models/meca"}, {"POST", "/v1/chat/completions"}, {"POST", "/v1/responses"}, {"GET", "/v1/responses/resp_missing"}, {"POST", "/v1/responses/resp_missing/cancel"}} {
				reply := nativeRequest(t, transport, route.method, route.path, "{}")
				if reply.Code != 404 {
					t.Fatal("removed endpoint still available", protocol, route.path, reply.Code)
				}
			}
			if f.app.State().Session.ID != "" || len(f.app.State().Tasks) != 0 {
				t.Fatal("removed route started engine work")
			}
		})
	}
}
