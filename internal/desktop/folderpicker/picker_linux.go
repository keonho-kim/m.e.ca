// Purpose: Select a local desktop folder through XDG portals or the WSL Windows host.
// Pattern: Platform adapter with one owned D-Bus connection per user action.
// Usage: Pick uses the portal on Linux; WSL uses its built-in Windows folder chooser.
// Related: internal/desktop/folderpicker/picker.go, internal/host/wsl/discovery.go.
package folderpicker

import (
	"context"
	"crypto/rand"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/host/wsl"
)

func pick(ctx context.Context) (string, error) {
	if wsl.Available() {
		return pickWSL(ctx)
	}
	conn, err := dbus.ConnectSessionBus(dbus.WithContext(ctx))
	if err != nil {
		return "", fault.New(diag.Unavailable, diag.FolderPickerUnavailable)
	}
	defer conn.Close()
	const service = "org.freedesktop.portal.Desktop"
	const request = "org.freedesktop.portal.Request"
	token := "meca_" + rand.Text()
	sender := strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_")
	handle := dbus.ObjectPath("/org/freedesktop/portal/desktop/request/" + sender + "/" + token)
	options := []dbus.MatchOption{dbus.WithMatchSender(service), dbus.WithMatchObjectPath(handle), dbus.WithMatchInterface(request), dbus.WithMatchMember("Response")}
	// Subscribe before opening the dialog, so an immediate response cannot be lost.
	if err := conn.AddMatchSignalContext(ctx, options...); err != nil {
		return "", err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = conn.RemoveMatchSignalContext(cleanup, options...)
	}()
	signals := make(chan *dbus.Signal, 1)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)
	object := conn.Object(service, "/org/freedesktop/portal/desktop")
	var returned dbus.ObjectPath
	err = object.CallWithContext(ctx, "org.freedesktop.portal.FileChooser.OpenFile", 0, "", diag.Text(diag.FolderPickerTitle), map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token), "directory": dbus.MakeVariant(true), "multiple": dbus.MakeVariant(false),
	}).Store(&returned)
	if err != nil {
		var busError dbus.Error
		if errors.As(err, &busError) && (busError.Name == "org.freedesktop.DBus.Error.ServiceUnknown" || busError.Name == "org.freedesktop.DBus.Error.NameHasNoOwner" || busError.Name == "org.freedesktop.DBus.Error.UnknownMethod") {
			return "", fault.New(diag.Unavailable, diag.FolderPickerUnavailable)
		}
		return "", err
	}
	if returned != handle {
		return "", fault.New(diag.Internal, diag.FolderPickerFailed)
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = conn.Object(service, handle).CallWithContext(cleanup, request+".Close", 0).Err
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case signal := <-signals:
		if signal == nil || len(signal.Body) != 2 {
			return "", fault.New(diag.Internal, diag.FolderPickerFailed)
		}
		var response uint32
		var values map[string]dbus.Variant
		if err := dbus.Store(signal.Body, &response, &values); err != nil {
			return "", err
		}
		if response == 1 {
			return "", nil
		}
		var uris []string
		if response != 0 || values["uris"].Store(&uris) != nil || len(uris) != 1 {
			return "", fault.New(diag.Internal, diag.FolderPickerFailed)
		}
		return localURI(uris[0])
	}
}

func localURI(raw string) (string, error) {
	uri, err := url.Parse(raw)
	if err != nil || uri.Scheme != "file" || uri.Host != "" && uri.Host != "localhost" || uri.User != nil || uri.RawQuery != "" || uri.Fragment != "" || uri.Opaque != "" {
		return "", fault.New(diag.Internal, diag.FolderPickerFailed)
	}
	return uri.Path, nil
}

func pickWSL(ctx context.Context) (string, error) {
	path, err := wsl.PowerShell()
	if err != nil {
		return "", err
	}
	// EOF from the Linux-owned input pipe terminates the Windows process even
	// if a WSL proxy is killed while the WinForms dialog is blocking.
	const script = `$ErrorActionPreference='Stop'; [Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false);
Add-Type -TypeDefinition 'using System; using System.Threading; public class MecaLifetime { public static void Watch() { new Thread(() => { try { var s=Console.OpenStandardInput(); while(s.ReadByte()!=-1){} } finally { Environment.Exit(0); } }) { IsBackground=true }.Start(); } }';
[MecaLifetime]::Watch(); Add-Type -AssemblyName System.Windows.Forms;
$d=New-Object System.Windows.Forms.FolderBrowserDialog;
try { if($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { [Console]::Write($d.SelectedPath) } } finally { $d.Dispose() }`
	selected, err := run(ctx, path, "-NoProfile", "-NonInteractive", "-STA", "-Command", script)
	if err != nil || selected == "" {
		return "", err
	}
	value, err := run(ctx, "wslpath", "-a", "-u", selected)
	return strings.TrimSuffix(value, "\n"), err
}
