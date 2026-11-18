// Purpose: Parse browser requests into a bounded allowlist of native CLI arguments.
// Pattern: Pure browser tool trust boundary.
// Usage: Local preparation validates once; sessions validate again before invoking the helper.
// Related: internal/tools/local/catalog/browser.go, internal/tools/browser/session.go.
package browser

import (
	"net"
	"net/url"
	"strconv"
	"strings"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type Request struct {
	Action, URL, Target, Value, Direction       string
	View, Selector, ResultID, Revision          string
	Width, Height, Pixels, Depth, Offset, Limit int
}

func (r Request) Validate() error {
	invalid := func(reason string) error { return fault.Errorf(diag.BrowserInput, reason) }
	for _, value := range []string{r.Target, r.Value, r.Selector} {
		if strings.ContainsRune(value, 0) || len(value) > 8192 {
			return invalid(diag.Text(diag.BrowserInvalidText))
		}
	}
	for _, value := range []string{r.Target, r.Selector} {
		if strings.HasPrefix(value, "-") {
			return invalid(diag.Text(diag.BrowserInvalidSelector))
		}
	}
	if r.Action != "fill" && strings.HasPrefix(r.Value, "-") {
		return invalid(diag.Text(diag.BrowserInvalidValue))
	}
	if r.Offset < 0 || r.Limit < 0 || r.Limit > 16384 || r.Limit > 0 && r.Limit < 4 {
		return invalid(diag.Text(diag.BrowserInvalidBounds))
	}
	if r.ResultID != "" {
		if r.Offset == 0 || r.Revision == "" {
			return invalid(diag.Text(diag.BrowserInvalidCursor))
		}
		if r.Action != "" || r.View != "" || r.URL != "" || r.Target != "" || r.Value != "" || r.Selector != "" || r.Depth != 0 || r.Width != 0 || r.Height != 0 || r.Pixels != 0 || r.Direction != "" || len(r.ResultID) > 64 || len(r.Revision) > 64 {
			return invalid(diag.Text(diag.BrowserInvalidResultRead))
		}
		return nil
	}
	if r.Offset != 0 || r.Revision != "" {
		return invalid(diag.Text(diag.BrowserMissingResult))
	}
	if r.View != "" {
		if r.Action != "" {
			return invalid(diag.Text(diag.BrowserMixedInput))
		}
		if r.URL != "" || r.Target != "" || r.Value != "" || r.Direction != "" || r.Width != 0 || r.Height != 0 || r.Pixels != 0 || r.View != "snapshot" && r.Depth != 0 || r.View != "snapshot" && r.View != "text" && r.Selector != "" {
			return invalid(diag.Text(diag.BrowserUnexpectedFields))
		}
		switch r.View {
		case "snapshot", "text", "screenshot", "errors":
		default:
			return invalid(diag.Text(diag.BrowserUnknownView))
		}
		if r.Depth < 0 || r.Depth > 12 {
			return invalid(diag.Text(diag.BrowserInvalidDepth))
		}
		return nil
	}
	if r.Selector != "" || r.Depth != 0 || r.Limit != 0 || r.Action != "viewport" && (r.Width != 0 || r.Height != 0) || r.Action != "scroll" && (r.Direction != "" || r.Pixels != 0) || r.Action != "fill" && r.Action != "select" && r.Action != "press" && r.Value != "" {
		return invalid(diag.Text(diag.BrowserUnexpectedFields))
	}
	switch r.Action {
	case "open":
		u, err := url.Parse(r.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || len(r.URL) > 8192 {
			return invalid(diag.Text(diag.BrowserInvalidURL))
		}
	case "click", "fill", "hover", "select", "check", "uncheck", "wait":
		if r.Target == "" || strings.HasPrefix(r.Target, "-") {
			return invalid(diag.Text(diag.BrowserMissingTarget))
		}
	case "press":
		if r.Value == "" || strings.HasPrefix(r.Value, "-") {
			return invalid(diag.Text(diag.BrowserMissingKey))
		}
	case "scroll":
		if r.Direction != "up" && r.Direction != "down" && r.Direction != "left" && r.Direction != "right" {
			return invalid(diag.Text(diag.BrowserInvalidDirection))
		}
		if r.Pixels < 0 || r.Pixels > 10000 {
			return invalid(diag.Text(diag.BrowserInvalidDistance))
		}
	case "viewport":
		if r.Width < 320 || r.Width > 2560 || r.Height < 240 || r.Height > 1600 {
			return invalid(diag.Text(diag.BrowserInvalidViewport))
		}
	case "back", "forward", "reload", "close":
	default:
		return invalid(diag.Text(diag.BrowserUnknownAction))
	}
	if r.Action != "open" && r.URL != "" {
		return invalid(diag.Text(diag.BrowserUnexpectedURL))
	}
	return nil
}

func (r Request) args() []string {
	if r.View == "snapshot" {
		depth := r.Depth
		if depth == 0 {
			depth = 5
		}
		args := []string{"snapshot", "-c", "-d", strconv.Itoa(depth)}
		if r.Selector != "" {
			args = append(args, "-s", r.Selector)
		}
		return args
	}
	if r.View == "text" {
		selector := r.Selector
		if selector == "" {
			selector = "body"
		}
		return []string{"get", "text", selector}
	}
	if r.View == "errors" {
		return []string{"errors"}
	}
	switch r.Action {
	case "open":
		return []string{"open", r.URL}
	case "fill":
		return []string{"fill", r.Target, "--", r.Value}
	case "select":
		return []string{"select", r.Target, r.Value}
	case "click", "hover", "check", "uncheck", "wait":
		return []string{r.Action, r.Target}
	case "press":
		return []string{"press", r.Value}
	case "scroll":
		pixels := r.Pixels
		if pixels == 0 {
			pixels = 500
		}
		return []string{"scroll", r.Direction, strconv.Itoa(pixels)}
	case "viewport":
		return []string{"set", "viewport", strconv.Itoa(r.Width), strconv.Itoa(r.Height)}
	default:
		return []string{r.Action}
	}
}

func (r Request) Interaction() bool {
	switch r.Action {
	case "click", "fill", "press", "select", "check", "uncheck", "hover", "back", "forward", "reload":
		return true
	}
	return false
}
func External(address string) bool {
	u, err := url.Parse(address)
	if err != nil {
		return true
	}
	host := u.Hostname()
	ip := net.ParseIP(host)
	return host != "localhost" && (ip == nil || !ip.IsLoopback())
}
