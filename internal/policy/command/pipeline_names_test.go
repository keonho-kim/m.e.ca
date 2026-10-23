// Purpose: Keep literal executables resolved across directory changes, pipelines and status reporting.
// Pattern: Static command policy regression tests.
// Usage: go test ./internal/policy/command -run TestPipelineLiteralNames.
// Related: internal/policy/command/flow.go, internal/policy/command/words.go, internal/policy/command/calls.go.
package command

import (
	"context"
	"testing"
)

func TestPipelineLiteralNames(t *testing.T) {
	for _, command := range []string{
		`cd /home/example/project`,
		`uv run pytest 2>&1`,
		`uv run pytest 2>&1 | tail -n 40`,
		`cd /home/example/project && uv run pytest 2>&1 | tail -n 40`,
		`cd /home/example/project; echo "EXIT: $?"`,
		`cd /home/example/project; "echo" "EXIT: $?"`,
		`cd /home/example/project; COMMAND=echo; "$COMMAND" "EXIT: $?"`,
		`cd /home/example/project; "${COMMAND:-echo}" "EXIT: $?"`,
		`echo "EXIT: $?"`,
		`cd /home/example/project && uv run pytest 2>&1 | tail -n 40; echo "EXIT: $?"`,
		`cd /home/example/project && uv run pytest 2>&1 | tail -n 40; echo “EXIT: $?”`,
		`cd /home/example/project && unregistered-command checks 2>&1 | tail -n 40; echo "EXIT: $?"`,
		`cd "$HOME/project" && unregistered-command checks 2>&1 | tail -n 40; echo "EXIT: $?"`,
		`set -eu; cd /home/example/project && unregistered-command checks 2>&1 | tail -n 40; printf 'EXIT: %s\n' "$?"`,
		`cd /home/example/project && unregistered-command checks 2>&1 | tail -n 40; result=$?; echo "EXIT: $result"`,
		`IFS=$(cat separator); cd /home/example/project && unregistered-command checks 2>&1 | tail -n 40; echo "EXIT: $?"`,
	} {
		t.Run(command, func(t *testing.T) {
			result, err := Analyze(context.Background(), command, testContext())
			if err != nil || len(result.Safety.Reasons) > 0 {
				t.Fatalf("literal executable required review: %+v %v", result, err)
			}
		})
	}
}

func TestPipelineNamesPreserveActualUncertainty(t *testing.T) {
	for _, test := range []struct{ command, want string }{
		{`cd src; "$PWD"`, "ask"},
		{`cd src; rm child`, "normal"},
		{`cd src && rm child`, "normal"},
		{`cd src; rm -rf /etc`, "block"},
		{`cd src; sudo true`, "block"},
		{`cd src; echo "$(rm -rf /etc)"`, "block"},
		{`cd src; "$(cat command)"`, "ask"},
		{`IFS=$(cat separator); COMMAND=echo; $COMMAND`, "ask"},
	} {
		t.Run(test.command, func(t *testing.T) {
			result, err := Analyze(context.Background(), test.command, testContext())
			got := "normal"
			if err != nil {
				got = "block"
			} else if len(result.Safety.Reasons) > 0 {
				got = "ask"
			}
			if got != test.want {
				t.Fatalf("want %s, got %s: %+v %v", test.want, got, result, err)
			}
		})
	}
}
