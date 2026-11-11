// Purpose: Commit a session connection while preserving credentials after uncertain persistence.
// Pattern: Connection persistence use case.
// Usage: Terminal and server callers adopt the prepared client only after Commit succeeds.
// Related: internal/provider/setup/setup.go, internal/provider/connection/models.go.
package modelchange

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/json"
	"errors"

	"strings"
	"time"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type SelectionStore interface {
	SaveSelection(context.Context, string, connection.Selection) error
	Selection(context.Context, string) (connection.Selection, error)
}
type CredentialWriter interface {
	Set(context.Context, string, string, string) error
	Get(context.Context, string, string) (string, error)
}

type Committed struct {
	Selection    connection.Selection
	Uncertain    bool
	DefaultError error
}

func Commit(ctx context.Context, store SelectionStore, vault CredentialWriter, session string, next connection.Selection, credential string, remember func(connection.Selection) error) (Committed, error) {
	if err := ctx.Err(); err != nil {
		return Committed{}, err
	}
	credential = strings.Trim(credential, " \t")
	if credential != "" {
		current, err := store.Selection(ctx, session)
		if err != nil {
			current = connection.Selection{}
		}
		next, err = BindCredential(ctx, vault, current, next, credential)
		if err != nil {
			return Committed{}, err
		}
	}
	if err := store.SaveSelection(ctx, session, next); err != nil {
		if errors.Is(err, sessionstate.ErrForeignSession) {
			return Committed{}, err
		}
		verify, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		actual, readErr := store.Selection(verify, session)
		cancel()
		expectedBody, encodeErr := json.Marshal(next)
		actualBody, actualErr := json.Marshal(actual)
		if readErr != nil || encodeErr != nil || actualErr != nil || !bytes.Equal(expectedBody, actualBody) {
			return Committed{Uncertain: readErr != nil}, faults.Errorf(diagcatalog.ModelchangeSaveModelConnectionExistingCredentialsRetained, errors.Join(err, readErr, encodeErr, actualErr))
		}
	}

	result := Committed{Selection: next}
	if remember != nil {
		if err := remember(next); err != nil {
			// The session and credential are already durable. Keep them usable even
			// if the independent default cannot be written; never delete their key.
			result.DefaultError = faults.Errorf(diagcatalog.ModelchangeConnectionSavedForThisSessionButSaving, err)
		}
	}
	return result, nil
}

// BindCredential returns the immutable reference to commit with the selected model.
func BindCredential(ctx context.Context, vault CredentialWriter, current, next connection.Selection, credential string) (connection.Selection, error) {
	if err := ctx.Err(); err != nil {
		return connection.Selection{}, err
	}
	if credential == "" {
		return next, nil
	}
	if vault == nil {
		return connection.Selection{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.SetupCredentialStoreIsRequiredForSavedConnection)
	}
	scope := next.Connection.CredentialScope()
	if ref := current.Connection.CredentialRef; current.Connection.ID == next.Connection.ID && ref != "" {
		saved, err := vault.Get(ctx, ref, scope)
		if ctx.Err() != nil {
			return connection.Selection{}, ctx.Err()
		}
		if err == nil && subtle.ConstantTimeCompare([]byte(saved), []byte(credential)) == 1 {
			next.Connection.CredentialRef = ref
			return next, nil
		}
	}
	ref := "credential_" + rand.Text()
	if err := vault.Set(ctx, ref, scope, credential); err != nil {
		return connection.Selection{}, err
	}
	next.Connection.CredentialRef = ref
	return next, nil
}
