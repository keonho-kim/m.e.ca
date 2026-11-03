// Purpose: Store public messages once and read complete room messages in committed timeline order.
// Pattern: Room log adapter over the existing conversation journal.
// Usage: Team serializes public appends separately; private sessions receive committed sequence cursors.
// Related: internal/storage/sqlite/team_members.go, internal/agent/team/types.go.
package sqlite

import (
	"context"
	"fmt"
	teamchannel "github.com/keonho-kim/meca/internal/agent/team/channel"
	teamidentity "github.com/keonho-kim/meca/internal/agent/team/identity"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func (s *Store) AppendTeamMessage(ctx context.Context, message teamchannel.Message) (teamchannel.Message, error) {
	if message.Text == "" || len(message.Text) > 64*1024*1024 {
		return message, fmt.Errorf("invalid public message")
	}
	var records []runrecord.Entry
	if message.From == "user" {
		records = []runrecord.Entry{{Message: &modelapi.Message{ID: "team-user", Role: "user", Content: message.Text}}}
	} else {
		if !teamidentity.ValidName(message.From) {
			return message, fmt.Errorf("invalid public sender")
		}
		event := runrecord.Event{Kind: "team_message", Agent: message.From, Final: message.Final, Text: message.Text}
		if message.FromStream {
			event.Kind = "team_progress"
		}
		records = []runrecord.Entry{{Event: &event}}
		if message.Final {
			records = append([]runrecord.Entry{{Message: &modelapi.Message{ID: "team-report", Role: "assistant", Content: message.Text}}}, records...)
		}
	}
	receipt := journalReceipt{}
	if err := s.commit(ctx, s.session.ID, records, 0, &receipt); err != nil {
		return message, err
	}
	message.Sequence = receipt.timeline
	return message, nil
}

func (s *Store) TeamMessages(ctx context.Context, after, through int64, limit int) ([]teamchannel.Message, error) {
	if through == 0 {
		through = 1 << 62
	}
	if after < 0 || through < after || limit < 1 || limit > 200 {
		return nil, fmt.Errorf("invalid room page")
	}
	page, err := s.recordPage(ctx, `SELECT t.sequence,t.kind,COALESCE(m.body,e.body),COALESCE(m.refs,e.refs),COALESCE(m.raw_bytes,e.raw_bytes) FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence WHERE t.session_id=? AND t.sequence>? AND t.sequence<=? AND (m.role='user' OR e.kind IN ('team_message','team_progress')) ORDER BY t.sequence LIMIT ?`, s.session.ID, after, through, limit)
	if err != nil {
		return nil, err
	}
	out := make([]teamchannel.Message, 0, len(page))
	size := 0
	for _, record := range page {
		if size+record.size > 16*1024*1024 && len(out) > 0 {
			break
		}
		message, err := s.publicMessage(ctx, record)
		if err != nil {
			return nil, err
		}

		if size+len(message.Text) > 16*1024*1024 && len(out) > 0 {
			break
		}
		size += len(message.Text)
		out = append(out, message)
	}
	return out, nil
}

// TeamPending checks unread work without hydrating bodies or counting own native responses twice.
func (s *Store) TeamPending(ctx context.Context, after, through int64, own string) (bool, error) {
	var pending bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM timeline t
 LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence
 LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence
 WHERE t.session_id=? AND t.sequence>? AND t.sequence<=? AND
 (m.role='user' OR e.kind='team_message' OR e.kind='team_progress' AND json_extract(e.body,'$.agent')<>?))`, s.session.ID, after, through, own).Scan(&pending)
	return pending, err
}
