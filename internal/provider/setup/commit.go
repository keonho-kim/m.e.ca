// Purpose: Commit a selected connection and compensate newly created credentials on failure.
// Pattern: Connection persistence use case.
// Usage: The terminal adopts the prepared client only after Commit succeeds.
// Related: internal/provider/setup/setup.go, internal/provider/connection/models.go.
package setup

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/provider/connection"
)

type SelectionStore interface {
	SaveSelection(context.Context, string, connection.Selection) error
}
type CredentialWriter interface{ Set(string, string) error }

func Commit(ctx context.Context, store SelectionStore, vault CredentialWriter, session string, next connection.Selection, credential string) (connection.Selection, error) {
	createdRef := ""
	if credential != "" {
		createdRef = "credential_" + rand.Text()
		if err := vault.Set(createdRef, credential); err != nil {
			return next, err
		}
		next.Connection.CredentialRef = createdRef
	}
	if err := store.SaveSelection(ctx, session, next); err != nil {
		if createdRef != "" {
			err = errors.Join(err, vault.Set(createdRef, ""))
		}
		return next, fmt.Errorf("save model connection: %w", err)
	}
	return next, nil
}
