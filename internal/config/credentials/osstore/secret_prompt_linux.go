// Purpose: Cancel and dismiss Secret Service unlock or deletion prompts without leaking signal readers.
// Pattern: Cancellable IPC prompt state transition.
// Usage: Secret Service Set, Get and Clear wait only within the operation context.
// Related: internal/config/credentials/osstore/store_linux.go.
package osstore

import (
	"context"

	"time"

	dbus "github.com/godbus/dbus/v5"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (c *secretClient) unlock(paths []dbus.ObjectPath) error {
	var unlocked []dbus.ObjectPath
	var prompt dbus.ObjectPath
	if err := c.call(secretPath, "Service.Unlock", paths).Store(&unlocked, &prompt); err != nil {
		return err
	}
	return c.prompt(prompt)
}

func (c *secretClient) prompt(path dbus.ObjectPath) error {
	if path == "/" {
		return nil
	}
	if !path.IsValid() {
		return faults.Errorf(diagcatalog.OsstoreInvalidSecretServicePrompt)
	}
	options := []dbus.MatchOption{dbus.WithMatchSender(secretName), dbus.WithMatchObjectPath(path), dbus.WithMatchInterface(secretInterface + "Prompt"), dbus.WithMatchMember("Completed")}
	if err := c.conn.AddMatchSignalContext(c.ctx, options...); err != nil {
		return err
	}
	defer func() { _ = c.conn.RemoveMatchSignalContext(c.ctx, options...) }()
	signals := make(chan *dbus.Signal, 1)
	c.conn.Signal(signals)
	defer c.conn.RemoveSignal(signals)
	dismiss := func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(c.ctx), time.Second)
		defer cancel()
		_ = c.conn.Object(secretName, path).CallWithContext(ctx, secretInterface+"Prompt.Dismiss", 0).Err
	}
	if err := c.call(path, "Prompt.Prompt", "").Err; err != nil {
		if c.ctx.Err() != nil {
			dismiss()
			return c.ctx.Err()
		}
		return err
	}
	select {
	case <-c.ctx.Done():
		dismiss()
		return c.ctx.Err()
	case signal, ok := <-signals:
		if !ok || signal == nil || len(signal.Body) != 2 {
			return faults.Errorf(diagcatalog.OsstoreSecretServicePromptEndedWithoutAResult)
		}
		dismissed, ok := signal.Body[0].(bool)
		if !ok || dismissed {
			return faults.Errorf(diagcatalog.OsstoreSecretServicePromptWasDismissed)
		}
		return nil
	}
}
