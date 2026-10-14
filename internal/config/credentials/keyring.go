// Purpose: Store and retrieve provider credentials through the OS keyring.
// Pattern: Credential Adapter.
// Usage: Connection setup and model selection call Vault.Get and Set.
// Related: internal/config/credentials/redaction.go, internal/provider/setup/setup.go.
package credentials

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

const service = "v-code"

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
	if err := keyring.Set(service, ref, key); err != nil {
		return fmt.Errorf("OS credential store unavailable; key was not saved")
	}
	return nil
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
