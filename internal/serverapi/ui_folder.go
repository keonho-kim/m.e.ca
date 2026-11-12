// Purpose: Admit one local, browser-authenticated desktop folder selection at a time.
// Pattern: UI-only HTTP adapter; workspace policy remains with addWorkspace.
// Usage: Local WebUI requests native selection; remote clients use the existing folder browser.
// Related: internal/serverapi/ui_auth.go, cmd/meca/server.go, internal/desktop/folderpicker.
package serverapi

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/diagnostics"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (t *Transport) pickFolder(w http.ResponseWriter, r *http.Request) {
	host := (&url.URL{Host: r.Host}).Hostname()
	peer, _, _ := net.SplitHostPort(r.RemoteAddr)
	localHost := strings.EqualFold(host, "localhost") || net.ParseIP(host).IsLoopback()
	if !t.browserAuthenticated(r) || !localHost || !net.ParseIP(peer).IsLoopback() {
		http.Error(w, diag.Text(diag.FolderPickerLocalOnly), http.StatusForbidden)
		return
	}
	if t.api.PickFolder == nil {
		http.Error(w, diag.Text(diag.FolderPickerUnavailable), http.StatusServiceUnavailable)
		return
	}
	if !t.pickerBusy.CompareAndSwap(false, true) {
		http.Error(w, diag.Text(diag.FolderPickerBusy), http.StatusConflict)
		return
	}
	defer t.pickerBusy.Store(false)
	// A disconnected browser or server shutdown also closes the owned desktop request.
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()
	path, err := t.api.PickFolder(ctx)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if fault.Code(err) == diag.Unavailable {
			http.Error(w, diag.Text(diag.FolderPickerUnavailable), http.StatusServiceUnavailable)
			return
		}
		diagnostics.Log(ctx, diag.FolderSelectionFailed, "error_code", fault.Code(err))
		if errors.Is(err, context.DeadlineExceeded) {
			http.Error(w, diag.Text(diag.FolderPickerTimedOut), http.StatusGatewayTimeout)
		} else {
			http.Error(w, diag.Text(diag.FolderPickerFailed), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Path string `json:"path"`
	}{Path: path})
}
