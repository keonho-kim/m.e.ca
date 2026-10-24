// Purpose: Own unsaved model settings, server-bound capabilities and prepared-client cleanup.
// Pattern: Event-loop draft state.
// Usage: !model opens this draft; keyboard, rendering and setup results use it until save or discard.
// Related: internal/tui/session/model_form_keys.go, internal/tui/session/model_form_setup.go.
package session

import (
	"fmt"
	"strings"

	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"github.com/keonho-kim/meca/internal/tui/input"
)

const (
	modelProvider = iota
	modelEndpoint
	modelCredential
	modelName
	modelThinking
	modelEffort
	modelDestination
	modelCatalog
	modelInformation
	modelDiagnostic
	modelRowCount
)

type modelForm struct {
	provider, row                                         int
	endpoint, credential, name                            input.Editor
	editing, confirm, global, keyChanged, controlsChanged bool
	saving                                                bool
	thinking                                              *bool
	effort                                                string
	credentialRef, credentialOwner                        string
	prepared                                              *connection.Selection
	client                                                *httpclient.Client
	models                                                []string
	hints                                                 []string
	hint                                                  int
	details                                               bool
	detailsOffset                                         int
	revision                                              int
	connectionRevision                                    int
	catalog                                               bool
}

func (a *app) openModel() {
	a.closeModelForm()
	a.stopNotice, a.status, a.page = "", "", "model"
	a.menuInput.Set("")
	a.initializeModelForm()
}

func (a *app) initializeModelForm() {
	f := &modelForm{global: true}
	a.modelForm = f
	providers := connection.Providers()
	for i, provider := range providers {
		if provider.Kind == a.selection.Connection.Kind {
			f.provider = i
			break
		}
	}
	f.endpoint.Set(providers[f.provider].Endpoint)
	if a.selection.Model != "" {
		f.endpoint.Set(a.selection.Connection.Endpoint)
		f.name.Set(a.selection.Model)
		f.credentialRef, f.credentialOwner = a.selection.Connection.CredentialRef, a.selection.Connection.ID
		selected := a.selection
		f.prepared = &selected
		f.thinking, f.effort = selected.Thinking, selected.Effort()
		f.applyDefaults()
	}
}

func (a *app) closeModelForm() {
	if f := a.modelForm; f != nil {
		if a.worker.active() && !a.worker.agent() && a.taskCancel != nil {
			a.worker.stop()
			a.taskCancel()
		}
		if f.client != nil && f.client != a.client {
			f.client.Close()
		}
		f.credential.Set("")
	}
	a.modelForm = nil
	a.menuInput.Set("")
}

func (f *modelForm) editor() *input.Editor {
	switch f.row {
	case modelEndpoint:
		return &f.endpoint
	case modelCredential:
		return &f.credential
	case modelName:
		return &f.name
	default:
		return nil
	}
}

func (f *modelForm) connection() (connection.Connection, error) {
	if strings.TrimSpace(f.endpoint.String()) == "" {
		return connection.Connection{}, fmt.Errorf("Endpoint cannot be empty")
	}
	c, err := connection.New(connection.Providers()[f.provider].Kind, f.endpoint.String())
	if err == nil && c.ID == f.credentialOwner && !f.keyChanged {
		c.CredentialRef = f.credentialRef
	}
	return c, err
}

func (f *modelForm) selection() connection.Selection {
	c, _ := f.connection()
	next := connection.Selection{Connection: c, Model: strings.TrimSpace(f.name.String())}
	if f.prepared != nil && f.prepared.Connection.ID == c.ID && f.prepared.Model == next.Model {
		next = *f.prepared
		next.Connection = c
	}
	next.Thinking, next.ReasoningEffort = f.thinking, nil
	if f.effort != "" {
		value := f.effort
		next.ReasoningEffort = &value
	}
	return next
}

func (f *modelForm) options() connection.Reasoning {
	next := f.selection()
	return connection.ReasoningOptions(next.Connection.Kind, next.Details)
}

func (f *modelForm) applyDefaults() {
	options := f.options()
	if f.thinking == nil {
		f.thinking = options.KnownDefault()
	}
	if f.effort == "" {
		f.effort = options.Default
	}
}

func (f *modelForm) invalidate(connectionChanged bool) {
	f.revision++
	f.detailsOffset = 0
	if f.client != nil {
		f.client.Close()
		f.client = nil
	}
	f.prepared, f.thinking, f.effort, f.controlsChanged = nil, nil, "", false
	if connectionChanged {
		f.connectionRevision++
		f.models, f.hints = nil, nil
	}
}
