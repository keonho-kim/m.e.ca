// Purpose: Own model-setting drafts, capability selections and prepared-client cleanup.
// Pattern: Event-loop-owned settings component.
// Usage: Session lifecycle and input handlers use Form until save or discard.
// Related: internal/tui/settings/model/controls.go, internal/tui/settings/model/display.go, internal/tui/session/model_form_setup.go.
package model

import (
	"fmt"
	"strings"

	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"github.com/keonho-kim/meca/internal/tui/input"
)

const (
	Provider = iota
	Endpoint
	Credential
	Name
	Thinking
	Effort
	Destination
	Information
	Diagnostic
	RowCount
)

type Form struct {
	Provider, Row                                         int
	Endpoint, Credential, Name                            input.Editor
	Editing, Confirm, Global, KeyChanged, ControlsChanged bool
	Saving                                                bool
	Thinking                                              *bool
	Effort                                                string
	credentialRef, credentialOwner                        string
	Prepared                                              *connection.Selection
	Client                                                *httpclient.Client
	Models                                                []string
	Hints                                                 []string
	Hint                                                  int
	Details                                               bool
	DetailsOffset                                         int
	Revision                                              int
	ConnectionRevision                                    int
	Catalog                                               bool
}

func New(selected connection.Selection) *Form {
	f := &Form{Global: true}
	providers := connection.Providers()
	for i, provider := range providers {
		if provider.Kind == selected.Connection.Kind {
			f.Provider = i
			break
		}
	}
	f.Endpoint.Set(providers[f.Provider].Endpoint)
	if selected.Model != "" {
		f.Endpoint.Set(selected.Connection.Endpoint)
		f.Name.Set(selected.Model)
		f.credentialRef, f.credentialOwner = selected.Connection.CredentialRef, selected.Connection.ID
		f.Prepared = &selected
		f.Thinking, f.Effort = selected.Thinking, selected.Effort()
		f.ApplyDefaults()
	}
	return f
}

func (f *Form) Close(active *httpclient.Client) {
	if f.Client != nil && f.Client != active {
		f.Client.Close()
	}
	f.Client = nil
	f.Credential.Set("")
}

func (f *Form) Editor() *input.Editor {
	switch f.Row {
	case Endpoint:
		return &f.Endpoint
	case Credential:
		return &f.Credential
	case Name:
		return &f.Name
	default:
		return nil
	}
}

func (f *Form) Connection() (connection.Connection, error) {
	if strings.TrimSpace(f.Endpoint.String()) == "" {
		return connection.Connection{}, fmt.Errorf("endpoint cannot be empty")
	}
	c, err := connection.New(connection.Providers()[f.Provider].Kind, f.Endpoint.String())
	if err == nil && c.ID == f.credentialOwner && !f.KeyChanged {
		c.CredentialRef = f.credentialRef
	}
	return c, err
}

func (f *Form) Selection() connection.Selection {
	c, _ := f.Connection()
	next := connection.Selection{Connection: c, Model: strings.TrimSpace(f.Name.String())}
	if f.Prepared != nil && f.Prepared.Connection.ID == c.ID && f.Prepared.Model == next.Model {
		next = *f.Prepared
		next.Connection = c
	}
	next.Thinking, next.ReasoningEffort = f.Thinking, nil
	if f.Effort != "" {
		value := f.Effort
		next.ReasoningEffort = &value
	}
	return next
}

func (f *Form) Options() connection.Reasoning {
	next := f.Selection()
	return connection.ReasoningOptions(next.Connection.Kind, next.Details)
}

func (f *Form) ApplyDefaults() {
	options := f.Options()
	if f.Thinking == nil {
		f.Thinking = options.KnownDefault()
	}
	if f.Effort == "" {
		f.Effort = options.Default
	}
}

func (f *Form) Invalidate(connectionChanged bool) {
	f.Revision++
	f.DetailsOffset = 0
	if f.Client != nil {
		f.Client.Close()
		f.Client = nil
	}
	f.Prepared, f.Thinking, f.Effort, f.ControlsChanged = nil, nil, "", false
	if connectionChanged {
		f.ConnectionRevision++
		f.Models, f.Hints = nil, nil
	}
}
func (f *Form) RefreshHints() {
	if f.Editing && f.Row == Name {
		f.Hints = suggestions(f.Models, f.Name.String())
		f.Hint = 0
	}
}
