// Purpose: Bound simultaneous external browser sessions independently of task concurrency.
// Pattern: Process-wide nonblocking admission with session-owned release.
// Usage: Session.start acquires before provisioning and Close releases after confirmed shutdown.
// Related: internal/tools/browser/session.go, internal/diagnostics/catalog/browser.go.
package browser

import (
	"github.com/keonho-kim/meca/internal/processmemory"
	"os"
	"strconv"
	"sync"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

var browserCapacity = sync.OnceValues(func() (chan struct{}, error) {
	maximum := 3
	if value := os.Getenv("MECA_BROWSER_MAX_SESSIONS"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 {
			return nil, fault.New(diag.InvalidArgument, diag.BrowserCapacityInvalid)
		}
		maximum = parsed
	}
	return make(chan struct{}, maximum), nil
})

func acquireBrowser() (func(), error) {
	pressure, err := processmemory.Pressure()
	if err != nil {
		return nil, fault.Errorf(diag.BrowserMemoryObservation, err)
	}
	if pressure {
		return nil, fault.New(diag.ResourceExhausted, diag.BrowserMemoryPressure)
	}
	return reserveBrowser()
}

func reserveBrowser() (func(), error) {
	slots, err := browserCapacity()
	if err != nil {
		return nil, err
	}
	select {
	case slots <- struct{}{}:
		return sync.OnceFunc(func() { <-slots }), nil
	default:
		return nil, fault.New(diag.ResourceExhausted, diag.BrowserCapacityFull)
	}
}
