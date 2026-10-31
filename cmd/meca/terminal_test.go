//go:build linux || darwin

// Purpose: Verify the real CLI restores a PTY and releases its session on keyboard exit.
// Pattern: Executable integration test with owned pseudo-terminal resources.
// Usage: go test ./cmd/meca -run TestTerminalExecutable.
// Related: cmd/meca/main.go, cmd/meca/conversation.go, internal/tui/session/app.go.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"golang.org/x/term"
)

func TestTerminalExecutable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("interactive application requires an unprivileged user")
	}
	binary := testExecutable(t)
	for _, exit := range []struct{ name, input string }{{"Ctrl-C", "\x03"}, {"quit command", "!quit\r"}} {
		t.Run(exit.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			home, project := t.TempDir(), t.TempDir()
			cfg := config.Defaults()
			cfg.Language = "en"
			state := filepath.Join(home, ".meca")
			if err := config.Save(state, cfg); err != nil {
				t.Fatal(err)
			}
			master, slave, err := pty.Open()
			if err != nil {
				t.Fatal(err)
			}
			defer master.Close()
			defer slave.Close()
			if err := pty.Setsize(master, &pty.Winsize{Rows: 24, Cols: 100}); err != nil {
				t.Fatal(err)
			}
			supervisor, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			command := exec.CommandContext(ctx, supervisor, "-test.run=^TestTerminalSupervisor$", "--", binary, project)
			command.Env = append(os.Environ(), "MECA_TEST_PTY_SUPERVISOR=1", "HOME="+home, "TERM=xterm-256color", "LC_ALL=C", "LANG=C")
			command.Stdin, command.Stdout, command.Stderr = slave, slave, slave
			command.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
			command.Cancel = func() error { return syscall.Kill(-command.Process.Pid, syscall.SIGKILL) }
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			exited := make(chan error, 1)
			go func() { exited <- command.Wait() }()
			output, readerDone := terminalOutput(ctx, master)
			finished := false
			defer func() {
				cancel()
				if !finished {
					<-exited
				}
				slave.Close()
				master.Close()
				<-readerDone
			}()
			waitTerminalText(t, ctx, output, "New conversation")
			if _, err := io.WriteString(master, "\r"); err != nil {
				t.Fatal(err)
			}
			waitTerminalText(t, ctx, output, "INPUT")
			if err := pty.Setsize(master, &pty.Winsize{Rows: 18, Cols: 72}); err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(master, exit.input); err != nil {
				t.Fatal(err)
			}
			waitTerminalText(t, ctx, output, "MECA_TERMINAL_RESTORED")
			select {
			case err := <-exited:
				finished = true
				if err != nil {
					t.Fatalf("terminal exit: %v", err)
				}
			case <-ctx.Done():
				t.Fatal("terminal did not exit")
			}
			paths, err := filepath.Glob(filepath.Join(state, "sessions", "*", "*.db"))
			if err != nil || len(paths) != 1 {
				t.Fatalf("sessions: %v %v", paths, err)
			}
			sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
			if err != nil {
				t.Fatal(err)
			}
			id := strings.TrimSuffix(filepath.Base(paths[0]), ".db")
			store, err := sessions.Open(ctx, project, id)
			if err != nil {
				t.Fatalf("terminal exit retained the database lease: %v", err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Keep the controlling terminal alive while comparing its state after MECA exits.
// macOS revokes the slave when its session leader exits, so the supervisor owns it.
func TestTerminalSupervisor(t *testing.T) {
	if os.Getenv("MECA_TEST_PTY_SUPERVISOR") != "1" {
		return
	}
	before, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Args[len(os.Args)-2], os.Args[len(os.Args)-1])
	command.Stdin, command.Stdout, command.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := command.Run(); err != nil {
		t.Fatal(err)
	}
	after, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("terminal state not restored: %v", err)
	}
	fmt.Fprintln(os.Stdout, "MECA_TERMINAL_RESTORED")
}

func terminalOutput(ctx context.Context, terminal *os.File) (<-chan string, <-chan struct{}) {
	output, done := make(chan string, 16), make(chan struct{})
	go func() {
		defer close(done)
		defer close(output)
		buffer := make([]byte, 4096)
		for {
			n, err := terminal.Read(buffer)
			if n > 0 {
				select {
				case output <- string(buffer[:n]):
				case <-ctx.Done():
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return output, done
}

func waitTerminalText(t *testing.T, ctx context.Context, output <-chan string, text string) {
	t.Helper()
	var received string
	for {
		select {
		case chunk, ok := <-output:
			if !ok {
				t.Fatalf("terminal ended before %q: %q", text, received)
			}
			received += chunk
			if strings.Contains(received, text) {
				return
			}
			if len(received) > 64*1024 {
				received = received[len(received)-64*1024:]
			}
		case <-ctx.Done():
			t.Fatalf("terminal did not render %q: %q", text, received)
		}
	}
}
