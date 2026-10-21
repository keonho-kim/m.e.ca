// Purpose: Verify selective shell inspection applies one default to unregistered executables and preserves explicit risk gates.
// Pattern: Static policy regression tests.
// Usage: go test ./internal/policy/command -run TestSelectiveInspection.
// Related: internal/policy/command/calls.go, internal/policy/command/words.go, internal/policy/command/sed.go.
package command

import (
	"context"
	"testing"
)

func TestSelectiveInspection(t *testing.T) {
	for _, test := range []struct{ text, want string }{
		{`echo $((1 + 1)) && ls -la && unregistered-command sync`, "normal"},
		{`read -r VALUE; printf '%s' "$VALUE"; ls -la`, "normal"},
		{`read -r VALUE; rm -rf /etc`, "block"},
		{`rmdir -p /work/project/nested`, "block"},
		{`rmdir --parents /work/project/nested`, "block"},
		{`rmdir -p ./nested`, "block"},
		{`rmdir -p nested/deep`, "normal"},
		{`read -r VALUE; export RESULT="$VALUE"; ls`, "normal"},
		{`env RESULT_PATH="$(unregistered-command locate)" unregistered-command sync`, "normal"},
		{`env KEY=$(cat values) unregistered-command sync`, "ask"},
		{`NAME=$(cat name); env -u "$NAME" unregistered-command sync`, "normal"},
		{`command cd /etc && rm file`, "block"},
		{`nice -n $(cat priority) unregistered-command sync`, "ask"},
		{`echo $((COUNT += 1)); rm -rf /etc`, "block"},
		{`echo $((1 + $(rm -rf /etc)))`, "block"},
		{`TARGET=/etc; echo "${VALUE:=hello}"; rm -rf "$TARGET"`, "block"},
		{`printf '%s' "${TARGET:=/etc}"; rm -rf "$TARGET"`, "block"},
		{`while read -r file; do wc -l "$file"; done`, "normal"},
		{`while read -r file; do printf '%s' "$file"; done; ls`, "normal"},
		{`for file in $(find . -name '*.py'); do wc -l "$file"; done`, "normal"},
		{`for value in 1 2; do TARGET=../$TARGET; printf x > "$TARGET"; done`, "ask"},
		{`unregistered-one --directory /etc sync && unregistered-two install stable`, "normal"},
		{`unregistered-command run nested-command --target "$RUNTIME_VALUE"`, "normal"},
		{`unregistered-command --command 'rm -rf /etc'`, "normal"},
		{`unregistered-command run rm -rf /etc`, "normal"},
		{`unregistered-command sync && rm -rf /etc`, "block"},
		{`unregistered-command run task "$(rm -rf /etc)"`, "block"},
		{`unregistered-command > /etc/config`, "block"},
		{`find . -name '*.py' -print0 | xargs -0 wc -l`, "normal"},
		{`find . -exec cat {} \;`, "normal"},
		{`find . -type f -delete`, "ask"},
		{`find nested -delete`, "normal"},
		{`find -L nested -delete`, "ask"},
		{`find . -mindepth 1 -delete`, "ask"},
		{`find . -mindepth 1 -mindepth 0 -delete`, "block"},
		{`find . -mindepth 1 -mindepth "$(cat depth)" -delete`, "block"},
		{`find . -name '*.tmp' -delete`, "ask"},
		{`find . -maxdepth 1 -delete`, "block"},
		{`find . -name '*' -delete`, "block"},
		{`find . -type d -delete`, "block"},
		{`find . ! -type f -delete`, "block"},
		{`find . -type f -o -true -delete`, "block"},
		{`find . -type f , -delete`, "block"},
		{`find . -delete -type f`, "block"},
		{`find . -printf '-delete'`, "normal"},
		{`find . -fprintf names.txt '-delete'`, "normal"},
		{`find . -fprintf /etc/names '%f'`, "block"},
		{`find . -exec sed -i 's/a/b/g' {} \;`, "ask"},
		{`find . -print0 | xargs -0 -I{} wc -l {}`, "normal"},
		{`xargs -I{} sh -c '{}'`, "ask"},
		{`find . -execdir wc -l {} +`, "normal"},
		{`find . -exec unregistered-command run inspect {} \;`, "normal"},
		{`find . -print0 | xargs -0 unregistered-command run inspect`, "normal"},
		{`find . -print0 | xargs -0 rm -f`, "ask"},
		{`find /etc -exec rm {} \;`, "block"},
		{`xargs -I{} sed -i 's/a/b/g' {}`, "ask"},
		{`VALUE=1 cd /etc && rm file`, "block"},
		{`VALUE=1 printf '%s' "${TARGET:=/etc}"; rm -rf "$TARGET"`, "block"},
		{`cd src && sed -i 's/old/new/g' main.py && unregistered-command run checks`, "normal"},
		{`sed -i '' 's/old/new/g' main.py`, "normal"},
		{`sed -i/etc/* 's/a/b/g' main.py`, "block"},
		{`sed -i.bak -e 's/old/new/g' main.py`, "normal"},
		{`FILE=main.py; sed -i 's/old/new/g' "$FILE"`, "normal"},
		{`FILE=$(cat target); sed -i 's/old/new/g' "$FILE"`, "ask"},
		{`PRINT_PROGRAM=$(cat program); sed -n "$PRINT_PROGRAM" main.py`, "ask"},
		{`sed -i 's/old/w \/etc\/config/g' main.py`, "normal"},
		{`sed -i 's#old#new#gi' main.py`, "normal"},
		{`sed -i '1,3d; /old/s/old/new/g' main.py`, "normal"},
		{`sed -i 'a\
literal w /etc/config' main.py`, "normal"},
		{`sed -n '1,5p' /etc/config`, "normal"},
		{`sed -n 'w /etc/config' main.py`, "block"},
		{`sed -n 's/a/b/w /etc/config' main.py`, "block"},
		{`sed 'e rm -rf /etc' main.py`, "block"},
		{`sed 's/a/b/e' main.py`, "ask"},
		{`sed -f changes.sed main.py`, "ask"},
		{`git checkout -b feature && git status --short`, "normal"},
		{`git branch drop`, "normal"},
		{`FLAGS=$(cat options); git branch "$FLAGS" feature`, "ask"},
		{`git -c alias.cleanup='!rm -rf /etc' cleanup`, "block"},
		{`git -c color.ui=false status --short`, "normal"},
		{`git checkout -- main.py`, "ask"},
		{`git clean -nd`, "normal"},
		{`git clean -nd --no-dry-run`, "ask"},
		{`git rm tracked.py`, "ask"},
		{`git mv before.py after.py`, "ask"},
		{`git rm --dry-run tracked.py`, "normal"},
		{`git -c core.sshCommand='ssh custom' push`, "ask"},
		{`chmod u+x ./scripts/run`, "normal"},
		{`chmod -R 777 .`, "block"},
		{`COMMAND=$(cat executable); "$COMMAND"`, "ask"},
	} {
		t.Run(test.text, func(t *testing.T) {
			result, err := Analyze(context.Background(), test.text, testContext())
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

func TestSelectiveInspectionReportsRealTargetBounds(t *testing.T) {
	for _, text := range []string{"git reset --hard", `find . -exec rm {} \;`, `find . -type f -delete`} {
		report, err := Analyze(context.Background(), text, testContext())
		if err != nil {
			t.Fatal(err)
		}
		if len(report.Safety.Targets) != 1 || report.Safety.Targets[0] != "/work/project" {
			t.Fatalf("fictional target exposed: %+v", report)
		}
	}
}
