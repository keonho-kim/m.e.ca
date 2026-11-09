// Purpose: Drain diagnostic queues fairly and publish bounded loss summaries.
// Pattern: Single owned output worker.
// Usage: Sink starts and joins this worker; failures never replay partially written records.
// Related: internal/diagnostics/logger.go, internal/diagnostics/catalog/output.go.
package diagnostics

import (
	"encoding/json"
	"errors"
	"syscall"
	"time"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Sink) run() {
	defer close(s.done)
	ticker := time.NewTicker(catalog.LogSummaryInterval)
	defer ticker.Stop()
	burst := 0
	var dropped, suppressed, missing uint64
	outputFailure := s.outputFailure
	summary := func() {
		d, next, absent := s.dropped.Load(), s.suppressed.Load(), missingReporters.Load()
		if d == dropped && next == suppressed && absent == missing {
			return
		}
		dropped, suppressed, missing = d, next, absent
		body, _ := json.Marshal(map[string]any{"time": s.options.Now().UTC().Format(time.RFC3339Nano), "level": "WARN", "msg": string(catalog.ReporterSummary), "dropped": d, "suppressed": next, "missing_reporters": absent})
		s.write(append(body, '\n'))
	}
	for {
		select {
		case <-outputFailure:
			s.fail()
			outputFailure = nil
		case <-ticker.C:
			summary()
		default:
		}
		if burst < 8 {
			select {
			case body := <-s.high:
				s.write(body)
				burst++
				continue
			default:
			}
		} else {
			select {
			case body := <-s.low:
				s.write(body)
				burst = 0
				continue
			default:
			}
		}
		select {
		case <-outputFailure:
			s.fail()
			outputFailure = nil
		case body := <-s.high:
			s.write(body)
			burst++
		case body := <-s.low:
			s.write(body)
			burst = 0
		case <-ticker.C:
			summary()
			burst = 0
		case <-s.stop:
			for {
				select {
				case body := <-s.high:
					s.write(body)
				case body := <-s.low:
					s.write(body)
				default:
					summary()
					return
				}
			}
		}
	}
}

func (s *Sink) write(body []byte) {
	if s.failed.Load() {
		s.dropped.Add(1)
		return
	}
	var n int
	err := fault.Guard(func() error { var cause error; n, cause = s.options.Writer.Write(body); return cause })
	if n == 0 && (errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK)) {
		s.dropped.Add(1)
		return
	}
	if err != nil || n != len(body) {
		s.fail()
	}
}

func (s *Sink) fail() {
	if s.failed.CompareAndSwap(false, true) {
		s.writeErr = ErrWrite
		close(s.failure)
		if _, stderrFailed := s.options.Writer.(*Stderr); !stderrFailed {
			emergency(catalog.ReporterFailed)
		}
	}
}
