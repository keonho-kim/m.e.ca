// Purpose: Verify the saved language controls the conversation picker independently of the host locale.
// Pattern: Virtual terminal startup tests.
// Usage: go test ./internal/tui/startup -run TestLanguage.
// Related: internal/tui/startup/resume.go, internal/tui/locale/preference.go.
package startup

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/keonho-kim/meca/internal/config"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

func TestLanguageConversationPicker(t *testing.T) {
	for _, test := range []struct {
		preference config.Language
		host, want string
	}{{config.LanguageKorean, "C", "대화 선택"}, {config.LanguageEnglish, "ko_KR.UTF-8", "Choose conversation"}} {
		t.Run(string(test.preference), func(t *testing.T) {
			t.Setenv("LC_ALL", test.host)
			screen, err := tcell.NewTerminfoScreenFromTty(vt.NewMockTerm(vt.MockOptSize{X: 80, Y: 12}))
			if err != nil {
				t.Fatal(err)
			}
			if err := screen.Init(); err != nil {
				t.Fatal(err)
			}
			defer screen.Fini()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			done := make(chan struct{})
			go func() {
				defer close(done)
				select {
				case screen.EventQ() <- tcell.NewEventKey(tcell.KeyEnter, "", 0):
				case <-ctx.Done():
				}
			}()
			load := func(context.Context, int, int) ([]sessionstate.Conversation, error) { return nil, nil }
			_, ok, err := Select(ctx, screen, load, sessionstate.Session{}, test.preference)
			cancel()
			<-done
			if err != nil || !ok {
				t.Fatalf("picker selection failed: %v", err)
			}
			var header strings.Builder
			for x := 0; x < 80; {
				text, _, width := screen.Get(x, 0)
				header.WriteString(text)
				x += max(1, width)
			}
			if !strings.Contains(header.String(), test.want) {
				t.Fatalf("configured language not used: %q", header.String())
			}
		})
	}
}
