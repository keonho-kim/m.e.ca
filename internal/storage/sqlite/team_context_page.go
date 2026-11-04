// Purpose: Resolve received public bodies in batches without copying them through full-message JSON.
// Pattern: Page-scoped authenticated content projection.
// Usage: ContextMessages and MemoryRecords load bounded metadata before restoring shared content.
// Related: internal/storage/sqlite/team_payloads.go, internal/storage/sqlite/context.go.
package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	teamchannel "github.com/keonho-kim/meca/internal/agent/team/channel"
	recordcontent "github.com/keonho-kim/meca/internal/storage/content"
	"strings"
)

func placeholders(count int) string { return strings.TrimSuffix(strings.Repeat("?,", count), ",") }

func (s *Store) sharedContext(ctx context.Context, page []storedRecord) (map[string]string, error) {
	if s.room == nil {
		return nil, nil
	}
	ids := map[string]bool{}
	args := make([]any, 0, len(page))
	for _, r := range page {
		var refs map[string]string
		if err := json.Unmarshal(r.refs, &refs); err != nil {
			return nil, err
		}
		if id := refs[`["content"]`]; id != "" && !ids[id] {
			ids[id] = true
			args = append(args, id)
		}
	}
	if len(args) == 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, "SELECT id,raw_bytes,body FROM payloads WHERE codec=3 AND id IN ("+placeholders(len(args))+")", args...)
	if err != nil {
		return nil, err
	}
	type ref struct {
		id       string
		sequence int64
		size     int
	}
	var refs []ref
	args = nil
	for rows.Next() {
		var id string
		var size int
		var body []byte
		var value roomPayload
		if err = rows.Scan(&id, &size, &body); err != nil {
			break
		}
		if err = json.Unmarshal(body, &value); err != nil || value.Sequence < 1 {
			err = fmt.Errorf("invalid public conversation reference")
			break
		}
		refs = append(refs, ref{id, value.Sequence, size})
		args = append(args, value.Sequence)
	}
	err = errors.Join(err, rows.Err(), rows.Close())
	if err != nil {
		return nil, err
	}
	if len(refs) == 0 {
		return nil, nil
	}
	bySequence, err := s.publicContexts(ctx, args)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(refs))
	for _, ref := range refs {
		content, ok := bySequence[ref.sequence]
		raw, err := json.Marshal(content)
		if err != nil {
			return nil, err
		}
		if !ok || len(raw) != ref.size || recordcontent.ID(raw) != ref.id {
			return nil, fmt.Errorf("referenced public message failed length/hash verification")
		}
		out[ref.id] = content
	}
	return out, nil
}
func (s *Store) publicMessage(ctx context.Context, record storedRecord) (teamchannel.Message, error) {
	body, err := s.unpackRecord(ctx, record, nil)
	if err != nil {
		return teamchannel.Message{}, err
	}
	message := teamchannel.Message{Sequence: record.sequence}
	if record.kind == "message" {
		var value modelapi.Message
		if err := json.Unmarshal(body, &value); err != nil {
			return message, err
		}
		if value.Role != "user" {
			return message, fmt.Errorf("public source is not a user instruction")
		}
		message.From, message.Text = "user", value.Content
	} else {
		var value runrecord.Event
		if err := json.Unmarshal(body, &value); err != nil {
			return message, err
		}
		if value.Kind != "team_message" && value.Kind != "team_progress" && value.Kind != "team_work" && value.Kind != "team_discussion" && value.Kind != "team_observation" && value.Kind != "team_status" {
			return message, fmt.Errorf("source is not public conversation")
		}
		if err := value.Team.Check(value.Kind); err != nil {
			return message, err
		}
		message.From, message.Text, message.Final = value.Agent, value.Text, value.Final
		message.Note = value.Team
		message.Discussion = value.Kind == "team_discussion"
		message.FromStream = value.Kind == "team_progress"
		message.Declaration = value.Kind == "team_work"
		message.Passive = value.Kind == "team_status"
		message.Observation = value.Kind == "team_observation" || message.Passive
	}
	return message, nil
}
