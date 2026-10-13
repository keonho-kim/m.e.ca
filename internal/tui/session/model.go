// Purpose: Own provider setup, model discovery and diagnostics with cancellable connection workers.
// Pattern: Use Case with explicit worker lifecycle.
// Usage: Started by model menus; transfers completed clients through ordered session notices.
// Related: internal/provider/connection/connection.go, internal/provider/httpclient/client.go, internal/tui/session/events.go.
package session

import (
	"context"
	"fmt"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"strings"
	"time"
)

func providerKinds() []modelconn.ProviderKind {
	return []modelconn.ProviderKind{modelconn.Ollama, modelconn.VLLM, modelconn.SGLang, modelconn.LMStudio, modelconn.OpenAICompatible}
}
func (a *app) connect(connection modelconn.Connection, key, model string) {
	if a.busy {
		return
	}
	a.busy = true
	a.status = "Connecting"
	ctx, cancel := context.WithTimeout(a.ctx, 3*time.Minute)
	a.taskCancel = cancel
	previous := a.selection
	defaults := a.opts.Config.Thinking
	a.workers.Add(1)
	go func() {
		defer a.workers.Done()
		defer cancel()
		finish := func(n notice) {
			n.Done = true
			if err := a.post(a.ctx, n); err != nil && n.Client != nil {
				n.Client.Close()
			}
		}
		if key == "" && connection.CredentialRef != "" {
			var err error
			key, err = a.opts.Vault.Get(connection.CredentialRef)
			if err != nil {
				finish(notice{Err: err})
				return
			}
		}
		client, err := httpclient.NewClient(connection, key)
		if err != nil {
			finish(notice{Err: err})
			return
		}
		transferred := false
		defer func() {
			if !transferred {
				client.Close()
			}
		}()
		selected := sqlite.Selection{Connection: connection, Model: model}
		if model == "" {
			listing, err := client.ListModels(ctx)
			var ids []string
			for _, m := range listing {
				ids = append(ids, m.ID)
			}
			ids = append(ids, "Enter Model ID manually")
			selected.Model = ""
			finish(notice{Models: ids, Selection: &selected, Err: err})
			return
		}
		caps, err := client.Diagnose(ctx, model)
		if err != nil {
			finish(notice{Err: err})
			return
		}
		if !caps.Chat || !caps.ToolDialogue {
			finish(notice{Err: fmt.Errorf("model did not follow the staged tool dialogue diagnostic")})
			return
		}
		supported, thinkingErr := client.ThinkingControl(ctx, model)
		if thinkingErr == nil && supported {
			selected.ThinkingSupported = true
			value := defaults
			if previous.Connection.ID == connection.ID && previous.Model == model && previous.Thinking != nil {
				value = *previous.Thinking
			}
			selected.Thinking = &value
		}
		transferred = true
		finish(notice{Client: client, Selection: &selected})
	}()
}

func (a *app) confirmField() {
	switch a.page {
	case "endpoint":
		endpoint := strings.TrimSpace(a.menuInput.String())
		if endpoint == "" {
			a.status = "Endpoint cannot be empty"
			return
		}
		connection, err := modelconn.New(providerKinds()[a.connectionKind], endpoint)
		if err != nil {
			a.fail(err)
			return
		}
		a.endpoint = connection.Endpoint
		a.status = ""
		a.page = "key"
		a.menuInput.Set("")
	case "key":
		a.key = a.menuInput.String()
		a.opts.Vault.Remember(a.key)
		kinds := providerKinds()
		connection, err := modelconn.New(kinds[a.connectionKind], a.endpoint)
		if err != nil {
			a.fail(err)
			return
		}
		a.menuInput.Set("")
		if a.key != "" {
			connection.CredentialRef = connection.ID
			if err = a.opts.Vault.Set(connection.ID, a.key); err != nil {
				a.fail(err)
				a.page = "chat"
				a.key = ""
				return
			}
		}
		a.connect(connection, a.key, "")
		a.key = ""
		a.page = "chat"
	case "manual":
		model := strings.TrimSpace(a.menuInput.String())
		if model == "" {
			return
		}
		a.connect(a.pendingConnection, "", model)
		a.menuInput.Set("")
		a.page = "chat"
	}
}
