// Purpose: Verify dedicated-tool routing parses command structure without classifying arguments as commands.
// Pattern: Policy parser tests.
// Usage: go test ./internal/policy/shell.
// Related: internal/policy/shell/check.go.
package shell

import "testing"

func TestDedicatedCommandRouting(t *testing.T) {
	for _, text := range []string{`touch a`, `mkdir x`, `mv a b`, `echo hi > output`, `find . -fprint output`, `sed --in-place=.bak x a`} {
		if err := Check(text); err == nil {
			t.Errorf("accepted dedicated command: %s", text)
		}
	}
	for _, text := range []string{`find "$PWD" -type f`, `find . -exec rm {} \;`, `rg --pre=cat text`, `rm a`, `/bin/rm a`, `env X=1 rm a`, `true && rm a`, `sh -c 'rm a'`, `find . -delete`, `git rm a`, `go test ./...`, `printf 'rm is text'`, `git status --short`, `make test`, `find . -type f -name "*.go"`, `ls -la`, `rg -n TODO .`, `grep -rn TODO .`, `find . -type f | head -n 20`} {
		if err := Check(text); err != nil {
			t.Errorf("%s: %v", text, err)
		}
	}
}
