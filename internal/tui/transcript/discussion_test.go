// Purpose: Show public feedback with natural bilingual attribution and preserve literal work details.
// Pattern: Pure transcript contract tests.
// Usage: go test ./internal/tui/transcript -run TestPublicDiscussion.
// Related: internal/tui/transcript/team.go, internal/tui/locale/messages.go.
package transcript

import (
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/storage/history"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestPublicDiscussionUsesNaturalAttribution(t *testing.T) {
	event := runrecord.Event{Kind: "team_discussion", Agent: "Ren", Text: "기존 API는 423을 사용합니다.", Team: &runrecord.TeamNote{ReplyTo: 184, ReplyAuthor: "Hana", BasedOn: 190}}
	for _, tc := range []struct {
		language locale.Language
		header   string
	}{
		{locale.Korean, "**Ren** · Hana에게"}, {locale.English, "**Ren** · to Hana"},
	} {
		f := Formatter{Language: tc.language, Redact: func(text string) string { return text }}
		blocks := f.Event(event)
		if len(blocks) != 2 || blocks[0].Text != tc.header || blocks[1].Text != event.Text {
			t.Fatalf("unexpected public feedback: %+v", blocks)
		}
		restored := f.History([]history.Record{{Kind: "event", Event: event}})
		if len(restored) != 2 || restored[0].Text != blocks[0].Text || restored[1].Text != blocks[1].Text {
			t.Fatal("saved discussion and live attribution differ", restored)
		}
		if strings.Contains(blocks[0].Text, "184") || strings.Contains(blocks[0].Text, "업무 선언") {
			t.Fatal("internal identifiers or unnatural labels appeared in the header")
		}
	}
}

func TestDepartureNoticeDoesNotClaimSuccessfulCompletion(t *testing.T) {
	for _, tc := range []struct {
		language locale.Language
		want     string
	}{{locale.Korean, "Hana · 퇴장했습니다"}, {locale.English, "Hana · Left"}} {
		f := Formatter{Language: tc.language, Redact: func(s string) string { return s }}
		blocks := f.Event(runrecord.Event{Kind: "team_notice", Agent: "Hana", Text: "left"})
		if len(blocks) != 1 || blocks[0].Text != tc.want {
			t.Fatal("departure claimed completed work", blocks)
		}
	}
}

func TestPublicDiscussionWorkDetailsRemainVisible(t *testing.T) {
	f := Formatter{Redact: func(text string) string { return text }}
	event := runrecord.Event{Kind: "team_work", Agent: "Hana", Text: "로그인 오류를 구현합니다.", Team: &runrecord.TeamNote{Work: &runrecord.TeamWork{Revision: 1, Paths: []string{"src/auth"}, Contract: "잠긴 계정은 423"}}}
	blocks := f.Event(event)
	if len(blocks) != 2 || !strings.Contains(blocks[1].Text, "src/auth") || !strings.Contains(blocks[1].Text, "잠긴 계정은 423") {
		t.Fatal("public work announcement hid required details", blocks)
	}
}

func TestPublicDiscussionShowsReplyAndOtherMentionedPeers(t *testing.T) {
	f := Formatter{Language: locale.Korean, Redact: func(v string) string { return v }}
	e := runrecord.Event{Kind: "team_discussion", Agent: "Ren", Text: "함께 확인해주세요.", Team: &runrecord.TeamNote{ReplyTo: 3, ReplyAuthor: "Hana", Mentions: []string{"Hana", "Arjun"}}}
	blocks := f.Event(e)
	if blocks[0].Text != "**Ren** · Hana, Arjun에게" {
		t.Fatal(blocks[0].Text)
	}
	e.Team.ReplyAuthor = "user"
	e.Team.Mentions = []string{"Hana"}
	if header := f.Event(e)[0].Text; header != "**Ren** · 사용자, Hana에게" {
		t.Fatal(header)
	}
}

func TestPublicDiscussionOmitsItsAuthorFromRecipients(t *testing.T) {
	for _, tc := range []struct {
		name string
		note *runrecord.TeamNote
		ko   string
		en   string
	}{
		{"own reference", &runrecord.TeamNote{ReplyTo: 3, ReplyAuthor: "Ren"}, "**Ren**", "**Ren**"},
		{"own mention", &runrecord.TeamNote{Mentions: []string{"Ren"}}, "**Ren**", "**Ren**"},
		{"followup for a peer", &runrecord.TeamNote{ReplyTo: 3, ReplyAuthor: "Ren", Mentions: []string{"Ren", "Hana"}}, "**Ren** · Hana에게", "**Ren** · to Hana"},
		{"peer reply with own mention", &runrecord.TeamNote{ReplyTo: 3, ReplyAuthor: "Hana", Mentions: []string{"Ren", "Hana", "Arjun"}}, "**Ren** · Hana, Arjun에게", "**Ren** · to Hana, Arjun"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, language := range []locale.Language{locale.Korean, locale.English} {
				want := tc.ko
				if language == locale.English {
					want = tc.en
				}
				f := Formatter{Language: language, Redact: func(s string) string { return s }}
				event := runrecord.Event{Kind: "team_discussion", Agent: "Ren", Text: "A public finding.", Team: tc.note}
				for _, blocks := range [][]viewport.Block{f.Event(event), f.History([]history.Record{{Kind: "event", Event: event}})} {
					if len(blocks) != 2 {
						t.Fatalf("expected header and body, got %d blocks", len(blocks))
					}
					if blocks[0].Text != want || blocks[1].Text != event.Text {
						t.Fatalf("unexpected live or restored recipients: header=%q body=%q", blocks[0].Text, blocks[1].Text)
					}
				}
			}
		})
	}
}
