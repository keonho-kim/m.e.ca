// Purpose: Verify dedicated-tool routing parses command structure without classifying arguments as commands.
// Pattern: Policy parser tests.
// Usage: go test ./internal/policy/shell.
// Related: internal/policy/shell/check.go.
package shell

import "testing"

func TestDedicatedCommandRouting(t *testing.T) {
	for _, text := range []string{"rm a", "/bin/rm a", `'rm' a`, `env X=1 rm a`, `true && rm a`, `sh -c 'rm a'`, `echo $(rm a)`, `find . -delete`, `git rm a`} {
		if err := Check(text); err == nil {
			t.Errorf("accepted dedicated command: %s", text)
		}
	}
	for _, text := range []string{`go test ./...`, `printf 'rm is text'`, `git status --short`, `make test`, `printf done > build.log`} {
		if err := Check(text); err != nil {
			t.Errorf("%s: %v", text, err)
		}
	}
}
