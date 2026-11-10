// Purpose: Exercise installed UI diagnostics through an isolated Playwright CLI session.
// Pattern: Opt-in browser contract test with a local HTTP fixture.
// Usage: MECA_UI_BROWSER_TEST=1 go test ./internal/config/bootstrap -run TestUIDesignBrowserDiagnostics.
// Related: internal/config/bootstrap/skills.go, internal/config/bootstrap/defaults/ui-design/scripts.
package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

type uiDiagnosticReport struct {
	Status      string   `json:"status"`
	Checked     int      `json:"checked"`
	Frames      int      `json:"frames"`
	Uninspected []string `json:"uninspected"`
	Findings    []struct {
		Kind   string `json:"kind"`
		Target string `json:"target"`
	} `json:"findings"`
}

func TestUIDesignBrowserDiagnostics(t *testing.T) {
	if os.Getenv("MECA_UI_BROWSER_TEST") != "1" {
		t.Skip("set MECA_UI_BROWSER_TEST=1 to exercise the installed scripts with Playwright CLI")
	}
	cli, err := exec.LookPath("playwright-cli")
	if err != nil {
		t.Fatal("Playwright CLI is required for this browser test:", err)
	}
	home, workdir := t.TempDir(), t.TempDir()
	if _, err := Initialize(home); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/empty":
			_, _ = w.Write([]byte("<!doctype html><title>Empty fixture</title><p>No controls</p>"))
			return
		case "/no-frames":
			_, _ = w.Write([]byte("<!doctype html><script>window.requestAnimationFrame = () => 0</script><button>Still</button>"))
			return
		case "/iframe":
			_, _ = w.Write([]byte("<!doctype html><button>Visible</button><iframe title='Uninspected' srcdoc='<p>Frame</p>'></iframe>"))
			return
		case "/sparse":
			_, _ = w.Write([]byte("<!doctype html><script>window.requestAnimationFrame = f => setTimeout(() => f(performance.now()), 80); window.cancelAnimationFrame = clearTimeout</script><button>Still</button>"))
			return
		case "/many-text":
			_, _ = w.Write([]byte("<!doctype html><button>" + strings.Repeat("<span>x</span>", 129) + "</button>"))
			return
		case "/long-text":
			_, _ = w.Write([]byte("<!doctype html><button>" + strings.Repeat("x", 16385) + "</button>"))
			return
		}
		_, _ = w.Write([]byte(uiDiagnosticFixture))
	}))
	t.Cleanup(server.Close)
	session := "meca-ui-" + strings.TrimPrefix(server.Listener.Addr().String(), "127.0.0.1:")
	invoke := func(args ...string) []byte {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		command := exec.CommandContext(ctx, cli, append([]string{"-s=" + session}, args...)...)
		command.Dir = workdir
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		if err := command.Run(); err != nil {
			t.Fatalf("Playwright CLI %v: %v\n%s\n%s", args, err, stdout.String(), stderr.String())
		}
		return stdout.Bytes()
	}
	t.Cleanup(func() { invoke("close") })
	browser := os.Getenv("MECA_UI_BROWSER")
	if browser == "" {
		browser = "chrome"
	}
	invoke("open", server.URL, "--browser="+browser)

	run := func(code string) uiDiagnosticReport {
		t.Helper()
		return decodeUIDiagnosticReport(t, invoke("--json", "run-code", code))
	}
	runFile := func(name string) uiDiagnosticReport {
		t.Helper()
		file := filepath.Join(home, ".meca", "skills", "ui-design", "scripts", name)
		return decodeUIDiagnosticReport(t, invoke("--json", "run-code", "--filename="+file))
	}

	layout := runFile("layout-check.js")
	if layout.Status != "issues" || layout.Checked < 3 || !hasUIFinding(layout, "control-fit", "button#clipped") || !hasUIFinding(layout, "horizontal-overflow", "button#overflow") {
		t.Fatalf("clipped control was not reported: %+v", layout)
	}
	if hasUIFinding(layout, "control-fit", "button#jump") {
		t.Fatalf("fitting control was reported as clipped: %+v", layout)
	}

	started := runFile("motion-check.js")
	if started.Status != "recording" {
		t.Fatalf("motion observation did not start: %+v", started)
	}
	run(`async page => {
		await page.waitForFunction(() => window.__mecaUIDesignMotion?.frames >= 4);
		const before = await page.evaluate(() => window.__mecaUIDesignMotion.frames);
		await page.locator('#jump').click();
		await page.waitForFunction(n => window.__mecaUIDesignMotion?.frames >= n + 8, before);
		return {status: 'triggered'};
	}`)
	result := run(`async page => page.evaluate(() => window.__mecaUIDesignMotion?.stop() ?? {status: 'inconclusive'})`)
	if result.Frames < 8 || result.Status == "no-issues-detected" || !hasUIFinding(result, "jump", "div#motion") {
		t.Fatalf("observable jump was not reported: %+v", result)
	}
	if started = runFile("motion-check.js"); started.Status != "recording" {
		t.Fatalf("motion observation was not released after stopping: %+v", started)
	}
	if still := run(`async page => page.evaluate(() => window.__mecaUIDesignMotion?.stop() ?? {status: 'inconclusive'})`); still.Status != "inconclusive" {
		t.Fatalf("observation without motion was presented as checked: %+v", still)
	}
	if started = runFile("motion-check.js"); started.Status != "recording" {
		t.Fatalf("motion observation did not restart: %+v", started)
	}
	run(`async page => {
		await page.waitForFunction(() => window.__mecaUIDesignMotion?.frames >= 4);
		await page.locator('#smooth').click();
		await page.locator('#motion').evaluate(async el => Promise.all(el.getAnimations().map(animation => animation.finished.catch(() => {}))));
		const after = await page.evaluate(() => window.__mecaUIDesignMotion.frames);
		await page.waitForFunction(n => window.__mecaUIDesignMotion?.frames >= n + 4, after);
		return {status: 'settled'};
	}`)
	smooth := run(`async page => page.evaluate(() => window.__mecaUIDesignMotion?.stop() ?? {status: 'inconclusive'})`)
	if hasUIFinding(smooth, "jump", "div#motion") {
		t.Fatalf("smooth transition was reported as an abrupt jump: %+v", smooth)
	}
	if started = runFile("motion-check.js"); started.Status != "recording" {
		t.Fatalf("motion observation did not restart before navigation: %+v", started)
	}
	invoke("goto", server.URL+"/empty")
	if missing := run(`async page => page.evaluate(() => window.__mecaUIDesignMotion?.stop() ?? {status: 'inconclusive'})`); missing.Status != "inconclusive" {
		t.Fatalf("navigation lost the recording without an inconclusive result: %+v", missing)
	}
	if empty := runFile("layout-check.js"); empty.Status != "inconclusive" || empty.Checked != 0 {
		t.Fatalf("empty page was presented as checked: %+v", empty)
	}
	invoke("goto", server.URL+"/iframe")
	if frame := runFile("layout-check.js"); frame.Status != "inconclusive" || frame.Checked == 0 {
		t.Fatalf("uninspected iframe was presented as checked: %+v", frame)
	}
	invoke("goto", server.URL+"/many-text")
	if bounded := runFile("layout-check.js"); bounded.Status != "inconclusive" || bounded.Checked != 0 {
		t.Fatalf("uninspected control beyond the text budget was presented as checked: %+v", bounded)
	}
	invoke("goto", server.URL+"/long-text")
	if bounded := runFile("layout-check.js"); bounded.Status != "inconclusive" || bounded.Checked != 0 {
		t.Fatalf("uninspected oversized text was presented as checked: %+v", bounded)
	}
	invoke("goto", server.URL+"/no-frames")
	if started = runFile("motion-check.js"); started.Status != "recording" {
		t.Fatalf("motion observation did not start without animation frames: %+v", started)
	}
	if noFrames := run(`async page => page.evaluate(() => window.__mecaUIDesignMotion?.stop() ?? {status: 'inconclusive'})`); noFrames.Status != "inconclusive" || noFrames.Frames != 0 {
		t.Fatalf("missing animation frames were presented as checked: %+v", noFrames)
	}
	invoke("goto", server.URL+"/sparse")
	if started = runFile("motion-check.js"); started.Status != "recording" {
		t.Fatalf("sparse motion observation did not start: %+v", started)
	}
	run(`async page => { await page.waitForFunction(() => window.__mecaUIDesignMotion?.frames >= 5); return {status: 'sampled'}; }`)
	sparse := run(`async page => page.evaluate(() => window.__mecaUIDesignMotion?.stop() ?? {status: 'inconclusive'})`)
	if sparse.Status != "inconclusive" || !slices.Contains(sparse.Uninspected, "animation frames were sampled too sparsely") {
		t.Fatalf("sparse animation frames were presented as checked: %+v", sparse)
	}
}

func decodeUIDiagnosticReport(t *testing.T, output []byte) uiDiagnosticReport {
	t.Helper()
	var envelope struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(output, &envelope); err != nil {
		t.Fatalf("decode Playwright CLI response %s: %v", output, err)
	}
	var report uiDiagnosticReport
	if err := json.Unmarshal([]byte(envelope.Result), &report); err != nil {
		t.Fatalf("decode UI diagnostic result %s: %v", envelope.Result, err)
	}
	return report
}

func hasUIFinding(report uiDiagnosticReport, kind, target string) bool {
	for _, finding := range report.Findings {
		if finding.Kind == kind && finding.Target == target {
			return true
		}
	}
	return false
}

const uiDiagnosticFixture = `<!doctype html>
<html lang="ko"><head><meta charset="utf-8"><title>UI diagnostic fixture</title><style>
body { margin: 24px; font: 16px/24px sans-serif; }
button { box-sizing: border-box; font: inherit; padding: 6px 10px; border: 1px solid; }
#clipped { height: 20px; }
#jump, #smooth { height: 40px; }
#overflow { height: 40px; width: 36px; white-space: nowrap; overflow: hidden; }
#motion { width: 80px; height: 36px; background: steelblue; margin-top: 24px; }
#motion.moved { transform: translateX(80px); }
#motion.smooth { transition: transform 350ms linear; }
</style></head><body>
<button id="clipped">잘리는 버튼</button>
<button id="overflow">너무 긴 버튼 이름</button>
<button id="jump">요소 이동</button>
<button id="smooth">부드럽게 복귀</button>
<div id="motion"></div>
<script>document.querySelector('#jump').addEventListener('click', () => document.querySelector('#motion').classList.toggle('moved'));</script>
<script>document.querySelector('#smooth').addEventListener('click', () => { const el = document.querySelector('#motion'); el.classList.add('smooth'); el.classList.remove('moved'); });</script>
</body></html>`
