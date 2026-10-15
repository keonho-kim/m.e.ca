// Purpose: Prepare an identified and diagnosed model connection and transfer client ownership.
// Pattern: Connection setup use case.
// Usage: Session workers call Prepare, then adopt or close the returned client.
// Related: internal/provider/setup/identity.go, internal/provider/discovery/client.go, internal/tui/session/model.go.
package setup

import (
	"context"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/diagnostic"
	"github.com/keonho-kim/v-code/internal/provider/discovery"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	"github.com/keonho-kim/v-code/internal/provider/transport"
)

type Credentials interface {
	Get(string) (string, error)
}

type Defaults struct {
	ContextLength            int
	Inference                *inference.Settings
	Profile, ReasoningEffort string
	Thinking                 bool
}

type Prepared struct {
	Client    *httpclient.Client
	Models    []connection.ModelInfo
	Selection *connection.Selection
}

func Prepare(ctx context.Context, connectionValue connection.Connection, key, model string, defaults Defaults, previous connection.Selection, credentials Credentials) (prepared Prepared, err error) {
	if key == "" && connectionValue.CredentialRef != "" {
		if credentials == nil {
			return prepared, fmt.Errorf("credential store is required for saved connection")
		}
		key, err = credentials.Get(connectionValue.CredentialRef)
		if err != nil {
			return prepared, err
		}
	}
	client, err := httpclient.NewClient(connectionValue, key, defaults.Profile)
	if err != nil {
		return prepared, err
	}
	defer func() {
		if prepared.Client == nil && client != nil {
			client.Close()
		}
	}()
	metadataHTTP := transport.New(key)
	defer metadataHTTP.Close()
	catalog := discovery.Client{Connection: connectionValue, HTTP: metadataHTTP}
	selected := connection.Selection{Connection: connectionValue, Model: model}
	if model == "" {
		prepared.Models, err = catalog.ListModels(ctx)
		prepared.Selection = &selected
		return prepared, err
	}
	report, err := catalog.Identify(ctx, model)
	if err != nil {
		return prepared, err
	}
	resolved, err := resolveIdentity(model, defaults.Profile, &report)
	if err != nil {
		return prepared, err
	}
	selected.Identity = &report
	selected.Profile = resolved.ID
	// The generation adapter owns one immutable effective profile. The served alias is unchanged.
	client.Close()
	client, err = httpclient.NewClient(connectionValue, key, resolved.ID)
	if err != nil {
		return prepared, err
	}
	info, discoveryErr := catalog.PrepareModel(ctx, model)
	if discoveryErr != nil {
		if ctx.Err() != nil {
			return prepared, ctx.Err()
		}
		if connectionValue.Kind == connection.Ollama || connectionValue.Kind == connection.LMStudio {
			return prepared, fmt.Errorf("model capability discovery: %w", discoveryErr)
		}
		info = connection.ModelInfo{ID: model, Notice: "Serving limits unavailable: " + discoveryErr.Error()}
	}
	if defaults.ContextLength != 0 {
		if err := client.SetContextLength(defaults.ContextLength); err != nil {
			return prepared, err
		}
	}
	if err := client.BindModel(info); err != nil {
		return prepared, err
	}
	selected.Details = &info
	if defaults.ReasoningEffort != "" {
		value := defaults.ReasoningEffort
		selected.ReasoningEffort = &value
	}
	if previous.Connection.ID == connectionValue.ID && previous.Model == model && previous.Profile == resolved.ID && previous.ReasoningEffort != nil {
		value := *previous.ReasoningEffort
		selected.ReasoningEffort = &value
	}
	if info.Reasoning != nil {
		err = info.Reasoning.CheckEffort(selected.Effort())
	} else {
		err = resolved.CheckEffort(connectionValue.Kind, selected.Effort())
	}
	if err != nil {
		return prepared, err
	}
	supported, err := diagnostic.ThinkingControl(connectionValue.Kind, model, resolved.ID, &info)
	if err != nil {
		return prepared, err
	}
	nativeOn := info.Reasoning != nil && info.Reasoning.Enabled()
	caps, err := diagnostic.Check(ctx, client, model, supported, nativeOn, defaults.Inference)
	if err != nil {
		return prepared, err
	}
	if caps.Thinking {
		selected.ThinkingSupported = true
		value := defaults.Thinking
		if previous.Connection.ID == connectionValue.ID && previous.Model == model && previous.Thinking != nil {
			value = *previous.Thinking
		}
		selected.Thinking = &value
	}
	if caps.Reasoning && !caps.Thinking {
		value := true
		selected.Thinking = &value
	}
	if selected.Effort() != "" {
		if !caps.Reasoning {
			return prepared, fmt.Errorf("Thinking effort requires verified separate reasoning")
		}
		if err = diagnostic.Effort(ctx, client, model, selected.Effort(), defaults.Inference); err != nil {
			return prepared, fmt.Errorf("Thinking effort diagnostic failed: %w", err)
		}
	}
	prepared.Client, prepared.Selection = client, &selected
	return prepared, nil
}
