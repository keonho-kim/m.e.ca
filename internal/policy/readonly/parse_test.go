// Purpose: Verify useful query forms and reject shell, command and option bypasses.
// Pattern: Query language boundary tests.
// Usage: go test ./internal/policy/readonly.
// Related: internal/policy/readonly/parse.go, internal/policy/readonly/queries.go.
package readonly

import (
	"strings"
	"testing"
)

func TestReadQueries(t *testing.T) {
	for _, command := range []string{
		"pwd; ls -lah src", "cat README.md | head -n 20", "grep -rn -e TODO src && wc -l README.md",
		"rg --files -g '*.go'", "rg --hidden -n 'func main' .", "find . -type f -name '*.go' -print",
		"find src -maxdepth 2 '(' -name '*.go' -o -name '*.md' ')' -print0",
		"sed -n '1,80p' README.md", "sed -n '1,$p' file", "uniq -c file", "cut -d : -f 1 file",
		"printf '%s\\n' 'hello world' | tr a-z A-Z", "du -sh .", "stat -f '%N %z' README.md",
		"readlink -f link", "git status --short", "git diff --stat HEAD", "git log -n 3 --format='%h %s'",
		"git show HEAD:README.md", "git ls-files -- '*.go'", "git grep -n TODO", "git rev-parse --show-toplevel",
		"cat 'name with spaces'", "printf '%s' 'a'\"'\"'b'", "cat -- --output",
	} {
		t.Run(command, func(t *testing.T) {
			query, err := Parse(command)
			if err != nil {
				t.Fatal(err)
			}
			rendered, err := query.Render(func(name string) (string, error) { return "/trusted/" + name, nil })
			if err != nil || !strings.Contains(rendered, "'/trusted/") {
				t.Fatalf("render: %q %v", rendered, err)
			}
		})
	}
}

func TestReadQueriesRejectBypasses(t *testing.T) {
	for _, command := range []string{
		"cat file > changed", "cat file >> changed", "cat file 2> changed", "cat < input", "cat <<EOF\nx\nEOF",
		"cat $(touch changed)", "cat `touch changed`", "cat ${X:=changed}", "cat \"$HOME\"", "cat $((1+2))",
		"X=1 cat file", "export X=1", "cat *.go", "cat ~/file", "cat file &", "! cat file", "cat file || ls",
		"(cat file)", "{ cat file; }", "for x in file; do cat $x; done", "f() { cat file; }; f",
		"sh -c 'cat file'", "env cat file", "command cat file", "timeout 1 cat file", "xargs cat", "awk '{print}' file",
		"python -c 'print(1)'", "curl localhost", "wget localhost", "go test ./...", "npm test", "make", "rm file",
		"sed -i '1p' file", "sed -n '1w changed' file", "sed -n '1e touch changed' file", "sed -e 'p' file",
		"sed -n 'p;w changed' file", "sed -n '1p' -i file", "sed -n '1p' --in-place file",
		"find . -delete", "find . -exec touch changed ';'", "find . -execdir sh -c true ';'", "find . -fprint changed",
		"find . -fprintf changed '%p'", "find . -ok sh ';'", "rg --pre 'touch changed' value", "rg -z value",
		"rg --pre=sh value", "rg --hostname-bin sh", "rg --search-zip value", "rg --no-config --pre sh value",
		"uniq input changed", "sort -o changed input", "git -c core.fsmonitor=sh status", "git --exec-path=evil status",
		"git config key value", "git checkout file", "git clean -fd", "git fetch", "git diff --output=changed",
		"git diff --ext-diff", "git show --textconv HEAD:file", "git log --show-signature", "git grep --open-files-in-pager=sh x",
		"git diff -U --output=changed", "git diff --unified --output=changed", "git log -U --output=changed", "rg -E --pre=sh value",
		"/bin/cat file", "./cat file", "cat file | tee changed", "git status; touch changed",
	} {
		t.Run(command, func(t *testing.T) {
			if _, err := Parse(command); err == nil {
				t.Fatal("accepted bypass")
			}
		})
	}
}

func TestRenderDisablesExecutableHelpers(t *testing.T) {
	for command, expected := range map[string][]string{
		"rg value":      {"'--no-config'"},
		"git diff HEAD": {"'--no-pager'", "'--no-optional-locks'", "'core.fsmonitor=false'", "'--no-ext-diff'", "'--no-textconv'"},
	} {
		query, err := Parse(command)
		if err != nil {
			t.Fatal(err)
		}
		rendered, err := query.Render(func(name string) (string, error) { return "/trusted/" + name, nil })
		if err != nil {
			t.Fatal(err)
		}
		for _, flag := range expected {
			if !strings.Contains(rendered, flag) {
				t.Fatalf("missing %s: %s", flag, rendered)
			}
		}
	}
}
