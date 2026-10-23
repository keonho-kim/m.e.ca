// Purpose: Verify known directory branches remain usable without weakening external or root protections.
// Pattern: Static shell policy regression tests.
// Usage: go test ./internal/policy/command -run TestDirectory.
// Related: internal/policy/command/state.go, internal/policy/command/builtins.go, internal/policy/command/targets.go.
package command

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestDirectoryBranchesRemainInspectable(t *testing.T) {
	for _, text := range []string{
		`cd src; mkdir -p generated`,
		`cd src; printf text > result.txt`,
		`cd src; rm stale.txt`,
		`cd src; sed -i 's/old/new/g' source.txt`,
		`cd src; cd nested; touch result.txt`,
		`if test -d src; then cd src; else cd tests; fi; touch result.txt`,
		`cd src && printf 'done\n' 2>&1 | tail -n 40; touch result.txt`,
		`(cd src; touch result.txt); touch top.txt`,
		`env -C src sh -c 'cd nested; touch result.txt'`,
		`command cd src; touch result.txt`,
		`cd src; find nested -delete`,
		`cd src && cd /etc && cd - && touch result.txt`,
	} {
		t.Run(text, func(t *testing.T) {
			report, err := Analyze(context.Background(), text, testContext())
			if err != nil || len(report.Safety.Reasons) != 0 {
				t.Fatalf("known project-local branches required approval: %+v %v", report, err)
			}
		})
	}
}

func TestDirectoryResolutionPreservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	input := testContext()
	resolve := input.Resolve
	input.Resolve = func(directory, path string, removal bool) (string, error) {
		if path == "result.txt" {
			cancel()
		}
		return resolve(directory, path, removal)
	}
	if _, err := Analyze(ctx, `cd /etc; touch result.txt`, input); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation was replaced by a target decision: %v", err)
	}
}

func TestDirectoryCandidateBudgetKeepsUnknownChangesReviewed(t *testing.T) {
	var prefix strings.Builder
	for _, directory := range []string{"one", "two", "three", "four", "five", "six"} {
		prefix.WriteString("cd " + directory + "; ")
	}
	for _, test := range []struct{ text, want string }{
		{"touch relative.txt", "ask"},
		{"touch /work/project/absolute.txt", "normal"},
		{"touch /etc/config", "block"},
		{"cd /work/project && touch restored.txt", "normal"},
	} {
		report, err := Analyze(context.Background(), prefix.String()+test.text, testContext())
		got := "normal"
		if err != nil {
			got = "block"
		} else if len(report.Safety.Reasons) > 0 {
			got = "ask"
		}
		if got != test.want {
			t.Fatalf("%s: want %s, got %s: %+v %v", test.text, test.want, got, report, err)
		}
	}
}

func TestDirectoryBranchesCheckEveryCandidate(t *testing.T) {
	report, err := Analyze(context.Background(), `cd src; touch result.txt`, testContext())
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/work/project/src/result.txt", "/work/project/result.txt"} {
		if !slices.Contains(report.Safety.Targets, path) {
			t.Fatalf("possible working directory was discarded: %q in %+v", path, report)
		}
	}
	for _, test := range []struct{ text, want string }{
		{`cd /etc; touch result.txt`, "block"},
		{`cd /etc; rm .`, "block"},
		{`cd src; rm .`, "block"},
		{`cd src; rm ..`, "block"},
		{`cd /home/other; touch result.txt`, "ask"},
		{`if test -d src; then cd /etc; else cd src; fi; touch result.txt`, "block"},
		{`read -r destination; cd "$destination"; touch result.txt`, "ask"},
		{`cd "$(cat destination)"; touch result.txt`, "ask"},
		{`cd "$(cat destination)"; printf x > /etc/config`, "block"},
		{`cd "$(cat destination)"; touch /work/project/result.txt`, "normal"},
		{`find . -execdir sh -c 'touch "$PWD/result.txt"' \;`, "ask"},
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
