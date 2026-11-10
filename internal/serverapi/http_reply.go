// Purpose: Serialize bounded native replies and hold output memory until network delivery finishes.
// Pattern: HTTP response boundary with byte admission and pooled compression.
// Usage: Command, task control and attachment errors share this writer.
// Related: internal/serverapi/transport.go, internal/serverapi/http_compression.go.
package serverapi

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/serverapi/admission"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"net/http"
)

func (t *Transport) writeControlReply(w http.ResponseWriter, r *http.Request, reply *v1.Reply) {
	t.outputOnce.Do(func() {
		if t.outputs == nil {
			t.outputs = admission.New(32 << 20)
		}
	})
	// Six bytes cover JSON escaping per UTF-8 byte; fixed slack covers this schema's field names.
	release, ok := t.outputs.Try(int64(proto.Size(reply))*6 + 64<<10)
	if !ok {
		reply = t.api.errorReply(reply.GetId(), faults.New(diagcatalog.ResourceExhausted, diagcatalog.PayloadMemoryLimit))
	} else {
		defer release()
	}
	body, err := protojson.Marshal(reply)
	if err != nil {
		http.Error(w, diagcatalog.Text(diagcatalog.ServerapiCouldNotEncodeReply), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	status := 0
	if reply.GetError() != nil {
		status = errorStatus(reply.GetError().Code)
	}
	reply = nil
	writeJSONBody(w, r, body, status)
}
