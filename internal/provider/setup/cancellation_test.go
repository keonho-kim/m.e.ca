// Purpose: Avoid credential I/O and persistence when model preparation or commit is already canceled.
// Pattern: Context-boundary contract tests.
// Usage: go test ./internal/provider/setup -run TestCanceled.
// Related: internal/provider/setup/setup.go, internal/commands/modelchange/commit.go.
package setup

import (
	"context"
	"errors"
	"testing"

	"github.com/keonho-kim/meca/internal/provider/connection"
)

type canceledCredentials struct{ called bool }

func (c *canceledCredentials) Get(context.Context, string, string) (string, error) {
	c.called = true
	return "", errors.New("unexpected credential read")
}

func TestCanceledSetupDoesNotReadOrWriteCredentials(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	conn, err := connection.New(connection.OpenAI, "")
	if err != nil {
		t.Fatal(err)
	}
	conn.CredentialRef = "saved"
	vault := &canceledCredentials{}
	if _, err := Prepare(ctx, conn, "", "alias", Defaults{}, connection.Selection{}, vault); !errors.Is(err, context.Canceled) || vault.called {
		t.Fatal("canceled setup touched credentials", err)
	}
}
