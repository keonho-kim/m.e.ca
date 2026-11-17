// Purpose: Observe native browser process memory and verify process/artifact release after repeated inspection.
// Pattern: Opt-in process-tree sampling against an isolated local HTTP fixture.
// Usage: Set MECA_BROWSER_MEMORY_PROBE=1 and the two native binary paths, then run TestNativeBrowserMemory.
// Related: internal/tools/browser/native_test.go, internal/tools/browser/session.go.
package browser

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/media/imagefile"
	browserruntime "github.com/keonho-kim/meca/internal/tools/browser/runtime"
)

func TestNativeBrowserMemory(t *testing.T) {
	agent, chrome := os.Getenv("MECA_BROWSER_NATIVE_AGENT"), os.Getenv("MECA_BROWSER_EXECUTABLE")
	if os.Getenv("MECA_BROWSER_MEMORY_PROBE") != "1" || agent == "" || chrome == "" {
		t.Skip("opt-in native process memory observation with explicit binary paths")
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("process sampling requires Unix ps")
	}
	fixture := `<html><body><main>` + strings.Repeat(`<p>Browser memory fixture paragraph</p>`, 1500) + `</main><canvas width="2560" height="1600"></canvas><script>
const canvas = document.querySelector('canvas'), ctx = canvas.getContext('2d');
const pixels = ctx.createImageData(2560, 1600); let value = 12345;
for (let i = 0; i < pixels.data.length; i += 4) {
  value ^= value << 13; value ^= value >>> 17; value ^= value << 5;
  pixels.data[i] = value & 255; pixels.data[i + 1] = (value >>> 8) & 255;
  pixels.data[i + 2] = (value >>> 16) & 255; pixels.data[i + 3] = 255;
}
ctx.putImageData(pixels, 0, 0); canvas.style.position = 'fixed'; canvas.style.inset = '0';
</script></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, fixture)
	}))
	defer server.Close()
	var sessions []*Session
	t.Cleanup(func() {
		for _, s := range sessions {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			err := s.Close(ctx)
			cancel()
			if err != nil {
				t.Error(err)
			}
		}
	})
	var roots []int
	var ownedPaths []string
	for i := range 3 {
		s := New(t.TempDir(), t.TempDir(), func(text string) string { return text }, nil)
		s.prepare = func(context.Context, string) (browserruntime.Paths, error) {
			return browserruntime.Paths{Agent: agent, Browser: chrome}, nil
		}
		sessions = append(sessions, s)
		if _, err := s.Run(t.Context(), Request{Action: "open", URL: server.URL}, ""); err != nil {
			t.Fatal(err)
		}
		ownedPaths = append(ownedPaths, s.work, s.sockets)
		pid, err := os.ReadFile(filepath.Join(s.sockets, s.name+".pid"))
		if err != nil {
			t.Fatal(err)
		}
		root, err := strconv.Atoi(strings.TrimSpace(string(pid)))
		if err != nil {
			t.Fatal(err)
		}
		roots = append(roots, root)
		processes, rss := nativeBrowserProcesses(t, roots)
		t.Logf("open_sessions=%d; owned_processes=%d; summed_RSS=%d bytes", i+1, len(processes), rss)
	}
	for _, s := range sessions {
		for range 8 {
			if _, err := s.Run(t.Context(), Request{View: "snapshot"}, s.Current()); err != nil {
				t.Fatal(err)
			}
		}
	}
	processes, rss := nativeBrowserProcesses(t, roots)
	t.Logf("after_24_snapshots; owned_processes=%d; summed_RSS=%d bytes", len(processes), rss)
	s := sessions[0]
	if _, err := s.Run(t.Context(), Request{Action: "viewport", Width: 2560, Height: 1600}, s.Current()); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	page, err := s.Run(t.Context(), Request{View: "screenshot"}, s.Current())
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()
	image, err := imagefile.Decode(t.Context(), "viewport.png", page.Image)
	if err != nil || image.Width != 2560 || image.Height != 1600 {
		t.Fatal("invalid viewport image", image.Width, image.Height, err)
	}
	runtime.ReadMemStats(&after)
	t.Logf("screenshot_bytes=%d; Go_total_alloc=%d bytes; heap_delta_before_GC=%d bytes", len(page.Image), after.TotalAlloc-before.TotalAlloc, int64(after.HeapAlloc)-int64(before.HeapAlloc))
	runtime.KeepAlive(image)
	runtime.KeepAlive(page)
	for _, s := range sessions {
		if err := s.Close(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	// RSS is a process-tree sum, not PSS: shared pages may be counted repeatedly.
	// Keep all observed PIDs to catch children reparented while the daemon exits.
	deadline := time.Now().Add(5 * time.Second)
	for {
		left, rss := nativeBrowserProcesses(t, processes)
		if len(left) == 0 {
			t.Log("all sampled browser/daemon PIDs exited")
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("owned browser processes survived close", left, rss)
		}
		time.Sleep(50 * time.Millisecond)
	}
	for _, path := range ownedPaths {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("retained native artifacts", path, err)
		}
	}
}

func nativeBrowserProcesses(t *testing.T, roots []int) ([]int, int64) {
	t.Helper()
	output, err := exec.CommandContext(t.Context(), "ps", "-A", "-o", "pid=,ppid=,rss=").Output()
	if err != nil {
		t.Fatal(err)
	}
	type process struct{ pid, parent, rss int }
	var rows []process
	for _, row := range strings.Split(string(output), "\n") {
		fields := strings.Fields(row)
		if len(fields) != 3 {
			continue
		}
		values := [3]int{}
		for i, field := range fields {
			values[i], err = strconv.Atoi(field)
			if err != nil {
				t.Fatal(err)
			}
		}
		rows = append(rows, process{values[0], values[1], values[2]})
	}
	owned := make(map[int]bool, len(roots))
	for _, root := range roots {
		owned[root] = true
	}
	for changed := true; changed; {
		changed = false
		for _, row := range rows {
			if owned[row.parent] && !owned[row.pid] {
				owned[row.pid], changed = true, true
			}
		}
	}
	var pids []int
	var rss int64
	for _, row := range rows {
		if owned[row.pid] {
			pids = append(pids, row.pid)
			rss += int64(row.rss) * 1024
		}
	}
	return pids, rss
}
