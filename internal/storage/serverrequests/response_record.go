// Purpose: Encode compatible state without duplicating the row identity or project scope.
// Pattern: Storage-owned metadata format with strict reconstruction.
// Usage: Compatible reservation, updates and native task lookup share this record codec.
// Related: internal/storage/serverrequests/responses.go, internal/commands/server/response.go.
package serverrequests

import (
	"bytes"
	"encoding/hex"
	"encoding/json"

	"io"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/work"
	"github.com/keonho-kim/meca/internal/commands/server"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const maxResponseRecordBytes = 64 << 10

type responseRecord struct {
	Session    string            `json:"session,omitempty"`
	Task       string            `json:"task,omitempty"`
	Previous   string            `json:"previous,omitempty"`
	Next       string            `json:"next,omitempty"`
	Status     string            `json:"status"`
	Error      string            `json:"error,omitempty"`
	Code       string            `json:"code,omitempty"`
	TaskStatus string            `json:"taskStatus,omitempty"`
	Created    int64             `json:"created"`
	Completed  int64             `json:"completed,omitempty"`
	Begin      int64             `json:"begin,omitempty"`
	Through    int64             `json:"through,omitempty"`
	Stored     bool              `json:"stored,omitempty"`
	Background bool              `json:"background,omitempty"`
	Team       bool              `json:"team,omitempty"`
	End        work.End          `json:"end"`
	Usage      *model.TokenUsage `json:"usage,omitempty"`
}

func encodeResponse(v server.Response) ([]byte, error) {
	value := responseRecord{
		Session: v.Session, Task: v.Task, Previous: v.Previous, Next: v.Next,
		Status: v.Status, Error: v.Error, Code: v.Code, TaskStatus: v.TaskStatus,
		Created: v.Created, Completed: v.Completed, Begin: v.Begin, Through: v.Through,
		Stored: v.Stored, Background: v.Background, Team: v.Team, End: v.End, Usage: v.Usage,
	}
	body, err := json.Marshal(value)
	if err == nil && len(body) > maxResponseRecordBytes {
		err = faults.Errorf(diagcatalog.ServerrequestsResponseMetadataExceedsLimit)
	}
	return body, err
}

func decodeResponse(v stored, id, scope string) (server.Response, error) {
	if v.kind == "native" {
		return server.Response{}, faults.New(diagcatalog.NotFound, diagcatalog.ServerResponseWasNotFound)
	}
	var value responseRecord
	decoder := json.NewDecoder(bytes.NewReader(v.response))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return server.Response{}, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return server.Response{}, faults.Errorf(diagcatalog.ServerrequestsInvalidResponseMetadataOriginalPreserved)
	}
	result := server.Response{
		ID: id, Scope: scope, Digest: hex.EncodeToString(v.digest), Kind: v.kind,
		Session: value.Session, Task: value.Task, Previous: value.Previous, Next: value.Next,
		Status: value.Status, Error: value.Error, Code: value.Code, TaskStatus: value.TaskStatus,
		Created: value.Created, Completed: value.Completed, Begin: value.Begin, Through: value.Through,
		Stored: value.Stored, Background: value.Background, Team: value.Team, End: value.End, Usage: value.Usage,
	}
	return result, checkResponse(result)
}
