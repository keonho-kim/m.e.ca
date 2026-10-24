// Purpose: Store, retrieve and remove MECA-owned keyring credentials with a non-secret cleanup marker.
// Pattern: Credential Adapter.
// Usage: Connection setup and model selection call Vault.Get and Set.
// Related: internal/config/credentials/redaction.go, internal/provider/setup/setup.go, internal/commands/install/uninstall.go.
package credentials

import (
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/config/statefs"
	"strings"

	"github.com/zalando/go-keyring"
)

const service = "meca"

func (v *Vault) Set(ref, key string) error {
	if ref == "" || len(key) > 2048 || strings.ContainsAny(key, "\r\n\x00") {
		return fmt.Errorf("invalid credential")
	}
	v.Remember(key)
	if key == "" {
		err := keyring.Delete(service, ref)
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return err
	}
	if v.State == "" {
		return fmt.Errorf("credential state directory is required")
	}
	if err := statefs.Write(v.State, "credentials.used", []byte(service+"\n")); err != nil {
		return err
	}
	if err := keyring.Set(service, ref, key); err != nil {
		return fmt.Errorf("OS credential store unavailable; key was not saved")
	}
	return nil
}

// Clear removes only credentials belonging to MECA's fixed keyring service.
func Clear() error {
	err := keyring.DeleteAll(service)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
func (v *Vault) Get(ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	key, err := keyring.Get(service, ref)
	if err != nil {
		return "", fmt.Errorf("credential unavailable; reconnect using !model")
	}
	v.Remember(key)
	return key, nil
}
