//go:build !minimum

// Purpose: Verify the executable serves a real project without a TTY and shuts down on SIGTERM.
// Pattern: Executable integration test.
// Usage: go test ./cmd/meca -run TestServerExecutable.
// Related: cmd/meca/server.go, cmd/meca/executable_test.go.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/config"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestServerExecutable(t *testing.T) {
	binary := testExecutable(t)
	home, project := t.TempDir(), t.TempDir()
	var generations atomic.Int32
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.NotFound(w, r)
			return
		}
		generations.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"served"},"finish_reason":"stop"}]}`)
	}))
	defer provider.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, "--server", "--listen", "127.0.0.1:0")
	cmd.Dir = project
	cmd.Env = append(os.Environ(), "HOME="+home)
	pipe, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	waited := false
	defer func() {
		if !waited {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	}()
	address := make(chan string, 1)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			line := scanner.Text()
			var entry struct {
				Message string `json:"msg"`
				Listen  string `json:"listen"`
			}
			if json.Unmarshal([]byte(line), &entry) == nil && entry.Message == "server_listening" {
				address <- entry.Listen
			}
		}
	}()
	var url string
	select {
	case url = <-address:
	case <-ctx.Done():
		t.Fatal("server did not become ready", ctx.Err())
	}
	token, err := os.ReadFile(filepath.Join(home, ".meca", "server.token"))
	if err != nil {
		t.Fatal(err)
	}
	body, _ := protojson.Marshal(&v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/v1/command", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
	client := &http.Client{Timeout: 5 * time.Second}
	defer client.CloseIdleConnections()
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	var reply v1.Reply
	if err := protojson.Unmarshal(data, &reply); err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 200 || reply.GetState().GetSession().GetProjectRoot() == "" {
		t.Fatalf("status=%s reply=%s", res.Status, data)
	}
	session := reply.GetState().Session.Id
	control := func(command *v1.Command) *v1.Reply {
		t.Helper()
		body, err := protojson.Marshal(command)
		if err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url+"/v1/command", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		data, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		var reply v1.Reply
		if err := protojson.Unmarshal(data, &reply); err != nil {
			t.Fatal(err)
		}
		if reply.GetError() != nil || res.StatusCode != 200 {
			t.Fatal(string(data))
		}
		return &reply
	}
	control(&v1.Command{Operation: &v1.Command_SetModel{SetModel: &v1.ModelRequest{SessionId: session, Provider: "openrouter", Endpoint: provider.URL, Model: "unknown-serving-alias", Remember: true}}})
	permission := "all_mighty"
	control(&v1.Command{Operation: &v1.Command_SetSettings{SetSettings: &v1.SettingsRequest{SessionId: session, Permission: &permission}}})
	saved, err := config.Load(filepath.Join(home, ".meca"))
	if err != nil || saved.Model == nil || saved.Model.Name != "unknown-serving-alias" {
		t.Fatal(saved.Model, err)
	}
	control(&v1.Command{Operation: &v1.Command_CloseSession{CloseSession: &v1.SessionTarget{SessionId: session}}})
	control(&v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{SessionId: session}}})
	task := control(&v1.Command{Operation: &v1.Command_Start{Start: &v1.StartRequest{SessionId: session, Key: "executable-request", Text: "answer"}}}).GetTask()
	watch, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"/v1/events", nil)
	if err != nil {
		t.Fatal(err)
	}
	watch.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(token)))
	events, err := client.Do(watch)
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(events.Body)
	finished := false
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var n v1.Notification
		if err := protojson.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &n); err != nil {
			t.Fatal(err)
		}
		if n.GetSnapshot().GetTask().GetStatus() == "answered" || n.GetEvent().GetKind() == "request_finished" {
			finished = true
			break
		}
	}
	_ = events.Body.Close()
	if !finished {
		t.Fatal("request did not finish", scanner.Err())
	}
	status := control(&v1.Command{Operation: &v1.Command_TaskStatus{TaskStatus: &v1.TaskRequest{TaskId: task.Id}}}).GetTask()
	if status.Status != "answered" || generations.Load() != 1 {
		t.Fatal(status, generations.Load())
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	err = cmd.Wait()
	waited = true
	if err != nil {
		t.Fatal(err)
	}
	<-readDone
}
