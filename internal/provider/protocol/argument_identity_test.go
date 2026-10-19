// Purpose: Verify Responses may reformat JSON but cannot change streamed argument values.
// Pattern: Stream boundary contract tests.
// Usage: go test ./internal/provider/protocol -run TestArgumentIdentity.
// Related: internal/provider/protocol/lmstudio_calls.go.
package protocol

import "testing"

func TestArgumentIdentityPreservesValuesAcrossJSONFormatting(t *testing.T) {
	for _, tc := range []struct {
		streamed, final string
		ok              bool
	}{
		{`{"steps":["생성", "확인"],"path":"src"}`, `{"path":"src","steps":["생성","확인"]}`, true},
		{`{"path":"가"}`, `{"path":"\uac00"}`, true},
		{`{"content":" line\n"}`, `{"content":"line\n"}`, false},
		{`{"steps":["one","two"]}`, `{"steps":["two","one"]}`, false},
		{`{"path":"a"}`, `{"path":"b","path":"a"}`, false},
		{`{"path":"a"}`, `{"path":"a"} {}`, false},
		{`{"offset":9007199254740992}`, `{"offset":9007199254740993}`, false},
	} {
		var draft callDraft
		if err := draft.appendArguments(tc.streamed); err != nil {
			t.Fatal(err)
		}
		err := draft.finalArguments(tc.final)
		if (err == nil) != tc.ok {
			t.Fatalf("%s -> %s: %v", tc.streamed, tc.final, err)
		}
		if draft.arguments.String() != tc.streamed {
			t.Fatal("literal streamed content changed")
		}
	}
}
