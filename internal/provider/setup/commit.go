// Purpose: Commit a session connection, optionally remember it for new sessions, and compensate failed credential writes.
// Pattern: Connection persistence use case.
// Usage: The terminal adopts the prepared client only after Commit succeeds.
// Related: internal/provider/setup/setup.go, internal/provider/connection/models.go.
package setup

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

type SelectionStore interface {
	SaveSelection(context.Context, string, connection.Selection) error
}
type CredentialWriter interface{ Set(string, string) error }

type Committed struct {
	Selection    connection.Selection
	DefaultError error
}

func Commit(ctx context.Context, store SelectionStore, vault CredentialWriter, session string, next connection.Selection, credential string, remember func(connection.Selection) error) (Committed, error) {
	createdRef := ""
	if credential != "" {
		createdRef = "credential_" + rand.Text()
		if err := vault.Set(createdRef, credential); err != nil {
			return Committed{}, err
		}
		next.Connection.CredentialRef = createdRef
	}
	if err := store.SaveSelection(ctx, session, next); err != nil {
		if createdRef != "" {
			err = errors.Join(err, vault.Set(createdRef, ""))
		}
		return Committed{}, fmt.Errorf("save model connection: %w", err)
	}
	result := Committed{Selection: next}
	if remember != nil {
		if err := remember(next); err != nil {
			// The session and credential are already durable. Keep them usable even
			// if the independent default cannot be written; never delete their key.
			result.DefaultError = fmt.Errorf("connection saved for this session, but saving the default for new sessions failed: %w", err)
		}
	}
	return result, nil
}
