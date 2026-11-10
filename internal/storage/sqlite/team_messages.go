// Purpose: Store public messages once and read complete room messages in committed timeline order.
// Pattern: Room log adapter over the existing conversation journal.
// Usage: Team serializes public appends separately; private sessions receive committed sequence cursors.
// Related: internal/storage/sqlite/team_members.go, internal/agent/team/types.go.
package sqlite

import (
	"context"

	teamchannel "github.com/keonho-kim/meca/internal/agent/team/channel"
	teamidentity "github.com/keonho-kim/meca/internal/agent/team/identity"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s *Store) AppendTeamMessage(ctx context.Context, message teamchannel.Message) (teamchannel.Message, error) {
	if message.Text == "" && len(message.Images) == 0 || len(message.Text) > 64*1024*1024 || len(message.Images) > 0 && message.From != "user" {
		return message, faults.Errorf(diagcatalog.SqliteInvalidPublicMessage)
	}
	if message.Passive && !message.Observation || message.Observation && (message.From == "user" || message.FromStream || message.Declaration || message.Discussion || message.Final) || message.Declaration && (message.From == "user" || message.FromStream || message.Discussion || message.Final) || message.Discussion && (message.From == "user" || message.FromStream || message.Final) || message.From == "user" && message.Note != nil {
		return message, faults.Errorf(diagcatalog.SqliteInconsistentPublicMessageProvenance)
	}
	message.Note = message.Note.Clone()
	var records []runrecord.Entry
	if message.From == "user" {
		records = []runrecord.Entry{{Message: &modelapi.Message{ID: "team-user", Role: "user", Content: message.Text, Images: message.Images}}}
	} else {
		if !teamidentity.ValidName(message.From) {
			return message, faults.Errorf(diagcatalog.SqliteInvalidPublicSender)
		}
		event := runrecord.Event{Kind: message.Kind(), Agent: message.From, Final: message.Final, Text: message.Text, Team: message.Note}
		records = []runrecord.Entry{{Event: &event}}
		if message.Final {
			records = append([]runrecord.Entry{{Message: &modelapi.Message{ID: "team-report", Role: "assistant", Content: message.Text}}}, records...)
		}
	}
	if message.Reference != nil {
		if !message.Observation {
			return message, faults.Errorf(diagcatalog.SqliteOnlyExecutionObservationsCanAttachAnOriginal)
		}
		reference := runrecord.Event{Kind: "team_reference", Agent: message.From, Reference: message.Reference}
		records = append([]runrecord.Entry{{Event: &reference}}, records...)
	}
	receipt := journalReceipt{public: &message}
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
		return nil, faults.Errorf(diagcatalog.SqliteInvalidRoomPage)
	}
	page, err := s.recordPage(ctx, "SELECT t.sequence,t.kind,COALESCE(m.body,e.body),COALESCE(m.refs,e.refs),COALESCE(m.raw_bytes,e.raw_bytes) FROM timeline t LEFT JOIN messages m ON t.kind='message' AND m.sequence=t.source_sequence LEFT JOIN tool_events e ON t.kind='event' AND e.sequence=t.source_sequence WHERE t.sequence>? AND t.sequence<=? AND (m.role='user' OR e.kind IN ('team_message','team_progress','team_work','team_discussion','team_observation','team_status')) ORDER BY t.sequence LIMIT ?", after, through, limit)
	if err != nil {
		return nil, err
	}
	// Admit the page before reading frozen prefixes or any large original body.
	bytes, count := 0, 0
	for _, record := range page {
		if count > 0 && record.size > teamchannel.PageBytes-bytes {
			break
		}
		bytes += record.size
		count++
	}
	page = page[:count]
	prefixes, err := s.publicPrefixes(ctx, page)
	if err != nil {
		return nil, err
	}
	out := make([]teamchannel.Message, 0, len(page))
	size := 0
	for _, record := range page {
		if size+record.size > teamchannel.PageBytes && len(out) > 0 {
			break
		}
		message, err := s.publicMessage(ctx, record, prefixes[record.sequence])
		if err != nil {
			return nil, err
		}

		cost := max(record.size, message.ContextBytes())
		if size+cost > teamchannel.PageBytes && len(out) > 0 {
			break
		}
		size += cost
		out = append(out, message)
	}
	return out, nil
}
