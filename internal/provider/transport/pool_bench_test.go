// Purpose: Measure HTTP/1.1 connection churn across repeated concurrent provider rounds.
// Pattern: Local transport benchmark with counted connection ownership.
// Usage: go test ./internal/provider/transport -run '^$' -bench BenchmarkIdlePool -benchmem.
// Related: internal/provider/transport/client.go.
package transport

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type poolRound struct {
	arrived atomic.Int32
	gate    chan struct{}
}

func BenchmarkIdlePool(b *testing.B) {
	for _, capacity := range []int{2, 8} {
		b.Run(fmt.Sprint(capacity), func(b *testing.B) {
			var connections atomic.Int64
			var active atomic.Pointer[poolRound]
			active.Store(&poolRound{gate: make(chan struct{})})
			endpoint := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				state := active.Load()
				if state.arrived.Add(1) == 8 {
					close(state.gate)
				}
				select {
				case <-state.gate:
				case <-r.Context().Done():
					return
				}
				_, _ = io.WriteString(w, "ok")
			}))
			endpoint.Config.ConnState = func(_ net.Conn, state http.ConnState) {
				if state == http.StateNew {
					connections.Add(1)
				}
			}
			endpoint.Start()
			defer endpoint.Close()
			client := New("", "")
			defer client.Close()
			pool := client.HTTP.Transport.(*http.Transport)
			pool.MaxIdleConns, pool.MaxIdleConnsPerHost = capacity, capacity
			round := func() {
				ctx, cancel := context.WithTimeout(b.Context(), 5*time.Second)
				defer cancel()
				var workers sync.WaitGroup
				for range 8 {
					workers.Go(func() {
						request, err := client.NewRequest(ctx, "GET", endpoint.URL, nil)
						if err != nil {
							b.Error(err)
							return
						}
						response, err := client.HTTP.Do(request)
						if err != nil {
							b.Error(err)
							return
						}
						_, err = io.Copy(io.Discard, response.Body)
						if err != nil {
							b.Error(err)
						}
						if err := response.Body.Close(); err != nil {
							b.Error(err)
						}
					})
				}
				workers.Wait()
				active.Store(&poolRound{gate: make(chan struct{})})
			}
			round()
			connections.Store(0)
			b.ReportAllocs()
			for b.Loop() {
				round()
			}
			b.ReportMetric(float64(connections.Load())/float64(b.N), "connections/round")
		})
	}
}
