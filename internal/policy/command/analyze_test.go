// Purpose: Verify shell grammar, expansion and wrapper decisions without running any commands.
// Pattern: Table-driven policy tests.
// Usage: go test ./internal/policy/command.
// Related: internal/policy/command/analyze.go, internal/policy/command/words.go.
package command

import (
	"context"
	"path/filepath"
	"testing"
)

func testContext() Context {
	return Context{Project: "/work/project", Home: "/home/user", Environment: []string{"HOME=/home/user", "PATH=/usr/bin:/bin"}, Resolve: func(dir, path string, _ bool) (ResolvedPath, error) {
		if !filepath.IsAbs(path) {
			path = filepath.Join(dir, path)
		}
		path = filepath.Clean(path)
		directory := path == "/work" || path == "/work/project" || path == "/work/project/out" || path == "/etc"
		return ResolvedPath{Path: path, Directory: directory}, nil
	}}
}

func TestAnalyzeDecisions(t *testing.T) {
	for _, test := range []struct{ text, want string }{
		{"ls -la && find . -type f && rg TODO .", "normal"},
		{"set -eu; unregistered-command run checks -q", "normal"},
		{"set -- /etc; rm -rf \"$1\"", "block"},
		{"export TARGET=/etc; rm -rf \"$TARGET\"", "block"},
		{"command -v unregistered-command; env; git status --short; git diff; unregistered-command init app", "normal"},
		{"echo 'rm -rf /etc'; printf '%s' 'sudo true' # rm /\ncat file", "normal"},
		{"printf hello > file; cat file 2>/dev/null", "normal"},
		{"cat <<'EOF' > source\nrm -rf /etc\nEOF", "normal"},
		{"cat <<EOF > source\n$(rm -rf /etc)\nEOF", "block"},
		{"cat <<'EOF' > source\n$(rm -rf /etc)\nEOF", "normal"},
		{"rm -rf /etc", "block"},
		{"/bin/rm -rf /usr/lib", "block"},
		{"env command rm -rf \"$HOME/bin\"", "block"},
		{"TARGET=/etc; rm -rf \"$TARGET\"", "block"},
		{"TARGET='/etc /tmp'; rm -rf $TARGET", "block"},
		{"TARGET='/etc /tmp'; rm -rf \"$TARGET\"", "ask"},
		{"rm -rf ${TARGET:-/etc}", "block"},
		{"cd /etc && rm -rf .", "block"},
		{"cd /etc; rm -rf .", "block"},
		{"echo \"$(rm -rf /etc)\"", "block"},
		{"bash -c 'cat <(rm -rf /etc)'", "block"},
		{"bash -c 'cat <(ls)'", "normal"},
		{"echo `rm -rf /etc`", "block"},
		{"sh -c 'rm -rf /etc'", "block"},
		{"bash -lc 'rm -rf /etc'", "block"},
		{"sh -c 'rm -rf \"$1\"' name /etc", "block"},
		{"sh <<'EOF'\nrm -rf /etc\nEOF", "block"},
		{"eval 'rm -rf /etc'", "block"},
		{"nice -n 10 timeout -s KILL 5 command rm /etc/a", "block"},
		{"busybox rm /etc/a", "block"},
		{"find /etc -delete", "block"},
		{"find /etc -type f -delete", "block"},
		{"find /etc -exec rm '{}' ';'", "block"},
		{"find . -exec rm -rf /etc ';'", "block"},
		{"printf /etc | xargs rm -rf", "ask"},
		{"xargs -0 rm -rf /etc", "block"},
		{"rm -rf /etc/*", "block"},
		{"rm -rf *.pyc", "ask"},
		{"rm -rf '*'", "normal"},
		{"rm -rf .", "block"},
		{"rm -rf ..", "block"},
		{"rm -- file", "normal"},
		{"mv old new", "normal"},
		{"printf hi > /etc/config", "block"},
		{"cp file /etc/file", "block"},
		{"mkdir -p /etc/new", "block"},
		{"tee /etc/file < source", "block"},
		{"printf hi > /home/user/new-file", "ask"},
		{"git reset --hard HEAD; git clean -fd", "ask"},
		{"git push origin main", "ask"},
		{"sudo true", "block"},
		{"env -i /usr/bin/sudo true", "block"},
		{"for p in /etc /tmp; do rm -rf \"$p\"; done", "block"},
		{"f() { rm -rf \"$1\"; }; f /etc", "block"},
		{"f() { \"$@\"; }; f sudo true", "block"},
		{"sh -c '\"$@\"' name sudo true", "block"},
		{"env -C /etc rm -rf .", "block"},
		{"stdbuf -o L sudo true", "block"},
		{"sed -i 's/a/b/g' /etc/config", "block"},
		{"git -C /etc reset --hard", "block"},
		{"git diff -- reset push", "normal"},
		{"echo {1..1000000000}", "normal"},
		{"f() { command f; }; f", "normal"},
		{"f() { f; }; f", "ask"},
		{"TARGET=$(cat target); rm -rf \"$TARGET\"", "ask"},
		{". ./settings; rm -rf \"$TARGET\"", "ask"},
		{"sh ./script.sh", "ask"},
		{"echo 'rm -rf /etc' | sh", "ask"},
		{"PROGRAM=$(cat program); sh -c \"$PROGRAM\"", "ask"},
		{"trap 'rm -rf /etc' EXIT", "block"},
		{"rm -rf ~someone/cache", "ask"},
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

func TestInvalidSyntaxAndCancellation(t *testing.T) {
	if _, err := Analyze(context.Background(), "if then", testContext()); err == nil {
		t.Fatal("invalid shell accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Analyze(ctx, "true", testContext()); err == nil {
		t.Fatal("cancellation ignored")
	}
}

func FuzzAnalyze(f *testing.F) {
	for _, seed := range []string{"rm \"$TARGET\"", "sh -c 'ls'", "cat <<'EOF'\ntext\nEOF", "f() { f; }; f", "${X:=value}", "echo $(ls)"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 65536 {
			t.Skip()
		}
		_, _ = Analyze(context.Background(), text, testContext())
	})
}

func BenchmarkAnalyze(b *testing.B) {
	for b.Loop() {
		_, err := Analyze(context.Background(), "cd app && unregistered-command run checks -q && git diff --stat", testContext())
		if err != nil {
			b.Fatal(err)
		}
	}
}
