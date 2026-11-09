// Purpose: Serialize native HTTP replies once with the shared status and diagnostic contract.
// Pattern: Native HTTP response boundary.
// Usage: Command, task control and attachment error routes call writeControlReply.
// Related: internal/serverapi/http.go, internal/serverapi/task_control.go, internal/serverapi/attachments.go.
package serverapi

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
	"net/http"
)

func (t *Transport) writeControlReply(w http.ResponseWriter, reply *v1.Reply) {
	body, err := protojson.Marshal(reply)
	if err != nil {
		http.Error(w, diagcatalog.Text(diagcatalog.ServerapiCouldNotEncodeReply), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if reply.GetError() != nil {
		w.WriteHeader(errorStatus(reply.GetError().Code))
	}
	_, _ = w.Write(body)
}
