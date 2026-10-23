// Purpose: Verify resolved internal moves run automatically while external, uncertain and protected targets retain guards.
// Pattern: Static shell policy regression tests.
// Usage: go test ./internal/policy/command -run TestMove.
// Related: internal/policy/command/effects.go, internal/policy/command/targets.go, internal/policy/safety.go.
package command

import (
	"context"
	"slices"
	"testing"
)

func TestMovePolicy(t *testing.T) {
	for _, test := range []struct{ text, want string }{
		{`mv old new`, "normal"},
		{`mv -- old new`, "normal"},
		{`mv -fnv old new`, "normal"},
		{`mv "*.txt" out`, "normal"},
		{`mv src/item .`, "normal"},
		{`mv one two out`, "normal"},
		{`mv -t out one two`, "normal"},
		{`mv -tout one two`, "normal"},
		{`mv -ftout one two`, "normal"},
		{`mv --target-directory=out one two`, "normal"},
		{`mv --target-directory out one two`, "normal"},
		{`cd src; mv old new`, "normal"},
		{`mv /work/project/src/item /work/project`, "normal"},
		{`mv old ../outside`, "ask"},
		{`mv ../outside new`, "ask"},
		{`mv *.txt out`, "ask"},
		{`mv /work/project/*.txt out`, "ask"},
		{`mv old /work/project/out*`, "ask"},
		{`mv *.txt`, "ask"},
		{`read -r operands; mv $operands`, "ask"},
		{`mv $(cat operands)`, "ask"},
		{`xargs mv`, "ask"},
		{`read -r source; mv "$source" out`, "ask"},
		{`mv old "$(cat target)"`, "ask"},
		{`mv --backup=numbered old new`, "ask"},
		{`mv --suffix=../outside old new`, "ask"},
		{`mv . backup`, "block"},
		{`mv .. backup`, "block"},
		{`mv /etc/config new`, "block"},
		{`mv /etc/*`, "block"},
		{`mv old /etc`, "block"},
		{`mv -t /etc old`, "block"},
		{`mv -T old /work/project`, "block"},
		{`git mv old new`, "ask"},
	} {
		t.Run(test.text, func(t *testing.T) {
			report, err := Analyze(context.Background(), test.text, testContext())
			got := "normal"
			if err != nil {
				got = "block"
			} else if len(report.Safety.Reasons) > 0 {
				got = "ask"
			}
			if got != test.want {
				t.Fatalf("want %s, got %s: %+v %v", test.want, got, report, err)
			}
		})
	}
}

func TestMoveDirectoryDestinationsUseSourceNames(t *testing.T) {
	for _, command := range []string{`mv src/one tests/two out`, `mv -t out src/one tests/two`} {
		report, err := Analyze(context.Background(), command, testContext())
		if err != nil || len(report.Safety.Reasons) > 0 {
			t.Fatalf("directory move required approval: %+v %v", report, err)
		}
		for _, path := range []string{"/work/project/src/one", "/work/project/tests/two", "/work/project/out/one", "/work/project/out/two"} {
			if !slices.Contains(report.Safety.Targets, path) {
				t.Fatalf("move endpoint missing: %q in %+v", path, report)
			}
		}
	}
}
