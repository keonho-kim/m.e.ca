// Purpose: Verify negotiated JSON compression and bounded response failure behavior.
// Pattern: HTTP adapter contract tests.
// Usage: go test ./internal/serverapi -run 'TestJSONCompression|TestReplyOutput'.
// Related: internal/serverapi/http_compression.go, internal/serverapi/http_reply.go.
package serverapi

import (
	"bytes"
	"compress/gzip"
	"github.com/keonho-kim/meca/internal/serverapi/admission"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONCompressionNegotiatesAndPreservesBody(t *testing.T) {
	body := []byte(strings.Repeat("bounded response ", 1000))
	for _, encoding := range []string{"gzip", "gzip;q=0", ""} {
		request := httptest.NewRequest("POST", "http://localhost/v1/command", nil)
		request.Header.Set("Accept-Encoding", encoding)
		response := httptest.NewRecorder()
		writeJSONBody(response, request, body, 200)
		actual := response.Body.Bytes()
		if encoding == "gzip" {
			if response.Header().Get("Content-Encoding") != "gzip" || len(actual) >= len(body) {
				t.Fatal("large JSON not compressed")
			}
			reader, err := gzip.NewReader(bytes.NewReader(actual))
			if err != nil {
				t.Fatal(err)
			}
			actual, err = io.ReadAll(reader)
			reader.Close()
			if err != nil {
				t.Fatal(err)
			}
		} else if response.Header().Get("Content-Encoding") != "" {
			t.Fatal("rejected encoding used")
		}
		if !bytes.Equal(actual, body) {
			t.Fatal("compression changed response")
		}
	}
}
func TestReplyOutputBudgetReturnsTypedFailure(t *testing.T) {
	transport := &Transport{api: &API{}, outputs: admission.New(1)}
	response := httptest.NewRecorder()
	transport.writeControlReply(response, httptest.NewRequest("POST", "http://localhost/v1/command", nil), &v1.Reply{Id: "same-request", Result: &v1.Reply_Ok{Ok: &v1.EmptyResult{}}})
	var reply v1.Reply
	if err := protojson.Unmarshal(response.Body.Bytes(), &reply); err != nil {
		t.Fatal(err)
	}
	if reply.Id != "same-request" || reply.GetError().GetCode() != "resource_exhausted" {
		t.Fatal(&reply)
	}
}

func TestReplyOutputBudgetDoesNotEchoOversizedCorrelation(t *testing.T) {
	transport := &Transport{api: &API{}, outputs: admission.New(1)}
	response := httptest.NewRecorder()
	transport.writeControlReply(response, httptest.NewRequest("POST", "http://localhost/v1/command", nil), &v1.Reply{Id: strings.Repeat("x", 1<<20)})
	if response.Body.Len() > 4096 {
		t.Fatal("rejection retained the oversized correlation")
	}
}
