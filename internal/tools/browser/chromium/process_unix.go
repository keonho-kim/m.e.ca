//go:build darwin || linux

// Purpose: Own a disposable Chromium process with a private control pipe and a rejecting proxy.
// Pattern: Headless process boundary with joined cancellation.
// Usage: One document render owns one profile; the interactive browser keeps its own profile.
// Related: internal/tools/browser/chromium/pipe.go, internal/tools/browser/webfetch.go.
package chromium

import (
	"context"
	"errors"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func start(ctx context.Context, executable, base string) (_ *pipe, _ func() error, err error) {
	profile, err := os.MkdirTemp(base, "chromium-document-")
	if err != nil {
		return nil, nil, err
	}
	started := false
	defer func() {
		if !started {
			err = errors.Join(err, os.RemoveAll(profile))
		}
	}()
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}
	proxyDone := make(chan struct{})
	go rejectTraffic(listener, proxyDone)
	closeProxy := func() error {
		err := listener.Close()
		<-proxyDone
		return err
	}
	defer func() {
		if !started {
			err = errors.Join(err, closeProxy())
		}
	}()
	childInput, input, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	defer childInput.Close()
	output, childOutput, err := os.Pipe()
	if err != nil {
		return nil, nil, errors.Join(err, input.Close())
	}
	defer childOutput.Close()
	lifetime, cancel := context.WithCancel(ctx)
	args := []string{
		"--headless", "--remote-debugging-pipe", "--user-data-dir=" + filepath.Join(profile, "profile"),
		"--proxy-server=http://" + listener.Addr().String(), "--proxy-bypass-list=<-loopback>",
		"--host-resolver-rules=MAP * ~NOTFOUND", "--disable-quic",
		"--force-webrtc-ip-handling-policy=disable_non_proxied_udp",
		"--no-first-run", "--no-default-browser-check", "--disable-background-networking",
		"--password-store=basic", "--use-mock-keychain",
		"--disable-component-update", "--disable-default-apps", "--disable-extensions",
		"--disable-sync", "--disable-breakpad", "--disable-crash-reporter", "--disable-domain-reliability", "--disable-client-side-phishing-detection",
		"--disable-features=MediaRouter,OptimizationHints,AutofillServerCommunication",
		"about:blank",
	}
	command := exec.CommandContext(lifetime, executable, args...)
	command.ExtraFiles = []*os.File{childInput, childOutput}
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if err := syscall.Kill(-command.Process.Pid, syscall.SIGKILL); errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		} else {
			return err
		}
	}
	command.WaitDelay = 2 * time.Second
	command.Dir = profile
	command.Env = chromeEnvironment(profile)
	if err := command.Start(); err != nil {
		cancel()
		return nil, nil, errors.Join(fault.Errorf(diag.WebFetchBrowserStart), err, input.Close(), output.Close())
	}
	started = true
	p := newPipe(lifetime, input, output)
	closeBrowser := func() error {
		cancel()
		stopErr := command.Cancel()
		closeErr := errors.Join(input.Close(), output.Close())
		_ = command.Wait()
		<-p.readerDone
		if errors.Is(stopErr, os.ErrProcessDone) {
			stopErr = nil
		}
		return errors.Join(stopErr, closeErr, closeProxy(), os.RemoveAll(profile))
	}
	return p, closeBrowser, nil
}

func rejectTraffic(listener net.Listener, done chan<- struct{}) {
	defer close(done)
	for {
		connection, err := listener.Accept()
		if err != nil {
			return
		}
		connection.Close()
	}
}

func chromeEnvironment(profile string) []string {
	env := []string{"HOME=" + profile, "XDG_CONFIG_HOME=" + profile, "XDG_CACHE_HOME=" + profile, "TMPDIR=" + profile}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		switch key {
		case "PATH", "LANG", "LC_ALL":
			env = append(env, entry)
		}
	}
	return env
}
