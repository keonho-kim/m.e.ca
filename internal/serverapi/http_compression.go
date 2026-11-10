// Purpose: Compress substantial JSON replies with reusable buffers while preserving negotiation.
// Pattern: HTTP encoding adapter.
// Usage: writeControlReply retains output admission until this writer finishes.
// Related: internal/serverapi/http_reply.go, internal/httpencoding/accept.go.
package serverapi

import (
	"compress/gzip"
	"github.com/keonho-kim/meca/internal/httpencoding"
	"io"
	"net/http"
	"strconv"
	"sync"
)

var jsonCompressors sync.Pool

func writeJSONBody(w http.ResponseWriter, r *http.Request, body []byte, status int) {
	accepted, identity := httpencoding.Accept(r.Header.Get("Accept-Encoding"))
	if !accepted && !identity {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	var compressor *gzip.Writer
	if accepted && (len(body) >= 4096 || !identity) {
		if cached := jsonCompressors.Get(); cached != nil {
			compressor = cached.(*gzip.Writer)
		} else {
			compressor, _ = gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
		}
	}
	w.Header().Add("Vary", "Accept-Encoding")
	if compressor != nil {
		w.Header().Set("Content-Encoding", "gzip")
	} else {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	}
	if status != 0 {
		w.WriteHeader(status)
	}
	if compressor == nil {
		_, _ = w.Write(body)
		return
	}
	compressor.Reset(w)
	_, _ = compressor.Write(body)
	_ = compressor.Close()
	// A pooled writer must not retain the response or its request context.
	compressor.Reset(io.Discard)
	jsonCompressors.Put(compressor)
}
