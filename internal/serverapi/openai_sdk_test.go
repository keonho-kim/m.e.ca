// Purpose: Exercise the official SDK without adding its dependencies to the product module.
// Pattern: Local-server fixture with a separate development-module client.
// Usage: go test ./internal/serverapi -run TestOpenAIOfficialSDK.
// Related: devtools/compatibility/openai_test.go, internal/serverapi/fixture_test.go.
package serverapi

import (
	"context"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenAIOfficialSDK(t *testing.T) {
	if testing.Short() {
		t.Skip("external SDK client integration")
	}
	f := newFixture(t, generateFunc(idleGenerator), false)
	transport := openAITransport(t, f, "sse")
	srv := httptest.NewUnstartedServer(transport)
	transport.hosts[srv.Listener.Addr().String()] = true
	srv.Start()
	defer srv.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	directory, err := filepath.Abs(filepath.Join("..", "..", "devtools"))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, "go", "-C", directory, "test", "./compatibility", "-run", "TestOfficialGoSDK", "-count=1")
	command.Env = append(os.Environ(), "MECA_COMPAT_BASE_URL="+srv.URL+"/v1/", "MECA_COMPAT_TOKEN="+testToken)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("official SDK: %v\n%s", err, output)
	}
}
