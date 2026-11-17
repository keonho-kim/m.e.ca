// Purpose: Run one bounded helper invocation without inheriting user browser automation configuration.
// Pattern: Native process boundary with composed capture buffers, joined cancellation and output ceilings.
// Usage: Session serializes calls and owns the helper's daemon socket directory.
// Related: internal/tools/browser/session.go, internal/tools/browser/runtime/install.go.
package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/processmemory"
)

const maxCapture = 4 << 20

type boundedBuffer struct {
	buffer    bytes.Buffer
	remaining int
	limited   bool
}

func (b *boundedBuffer) Bytes() []byte { return b.buffer.Bytes() }

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.remaining {
		b.limited = true
		return 0, fault.Errorf(diag.BrowserLimit)
	}
	n, err := b.buffer.Write(p)
	b.remaining -= n
	return n, err
}

type captured struct {
	data   json.RawMessage
	memory *processmemory.PayloadLease
}

func (c *captured) Close() {
	if c.memory != nil {
		c.memory.Close()
		c.memory = nil
	}
	c.data = nil
}

type response struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

func (s *Session) invoke(ctx context.Context, args ...string) (out captured, err error) {
	memory := processmemory.NewPayloadLease()
	defer func() {
		if err != nil {
			memory.Close()
		}
	}()
	captureLimit := 32 << 10
	if len(args) > 0 && (args[0] == "snapshot" || args[0] == "errors" || len(args) > 1 && args[0] == "get" && args[1] == "text") {
		captureLimit = maxCapture
	}
	// Shutdown must remain available even when observation admission is exhausted.
	if len(args) == 0 || args[0] != "close" {
		if err := memory.Resize(captureLimit * 3); err != nil {
			return captured{}, err
		}
	}
	ctx, cancel := context.WithTimeout(ctx, 40*time.Second)
	defer cancel()
	all := []string{"--json", "--no-webmcp", "--config", s.config, "--session", s.name, "--executable-path", s.paths.Browser, "--idle-timeout", "5m"}
	all = append(all, args...)
	command := exec.CommandContext(ctx, s.paths.Agent, all...)
	command.Dir = s.work
	command.Env = environment(s.sockets)
	command.WaitDelay = 2 * time.Second
	output := &boundedBuffer{remaining: captureLimit}
	stderr := &boundedBuffer{remaining: 8192}
	command.Stdout, command.Stderr = output, stderr
	err = command.Run()
	if output.limited || stderr.limited {
		return captured{}, fault.Errorf(diag.BrowserLimit)
	}
	if ctx.Err() != nil {
		return captured{}, ctx.Err()
	}
	result, parseErr := parseResponse(output.Bytes())
	if parseErr != nil {
		return captured{}, parseErr
	}
	if err != nil || !result.Success {
		message := result.Error
		if message == "" {
			message = diag.Text(diag.BrowserHelperExited)
		}
		return captured{}, fault.Errorf(diag.BrowserFailed, boundedError(s.redact(message)))
	}
	return captured{data: result.Data, memory: memory}, nil
}
func environment(sockets string) []string {
	allowed := map[string]bool{"HOME": true, "PATH": true, "LANG": true, "LC_ALL": true, "TMPDIR": true, "DISPLAY": true, "WAYLAND_DISPLAY": true, "XDG_RUNTIME_DIR": true, "SSL_CERT_FILE": true, "SSL_CERT_DIR": true}
	var env []string
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if allowed[key] {
			env = append(env, entry)
		}
	}
	return append(env, "AGENT_BROWSER_SOCKET_DIR="+sockets, "AGENT_BROWSER_DEFAULT_TIMEOUT=10000")
}

func boundedError(message string) string {
	if len(message) <= 2048 {
		return message
	}
	end := 2048
	for end > 0 && !utf8.RuneStart(message[end]) {
		end--
	}
	return message[:end]
}
