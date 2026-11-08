// Purpose: Declare server-owned session resources and transport-independent control values.
// Pattern: Consumer-owned I/O contracts.
// Usage: cmd/meca composes resources; serverapi invokes Service commands.
// Related: internal/commands/server/service.go, internal/commands/agentrequest/request.go.
package server

import (
	"context"

	"sync"

	conversationruntime "github.com/keonho-kim/meca/internal/commands/conversation"
	"github.com/keonho-kim/meca/internal/commands/modelchange"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/commands/teamrequest"
	"github.com/keonho-kim/meca/internal/config"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/storage/history"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type History interface {
	TimelineSummaries(context.Context, string, int64, int, history.ReadBounds) ([]history.Record, int64, error)
	ToolEvent(context.Context, record.EventReference, history.ReadBounds) (record.Event, error)
	SetTeamMode(context.Context, bool) error
}

// Conversation transfers its resources to Service only after Open succeeds.
type Conversation struct {
	Runtime     agentrequest.Runtime
	History     History
	Models      modelchange.SelectionStore
	Team        *teamrequest.Service
	TeamEnabled bool
	Close       func() error
	Model       *conversationruntime.Model
	readers     sync.WaitGroup
	Output      OutputHistory
	readContext context.Context
	stopReaders context.CancelFunc
}

type ModelSettings struct {
	Provider, Endpoint, Name, Credential string
	Thinking                             *bool
	Effort                               *string
	Remember                             bool
}

type PreparedModel struct {
	Selection connection.Selection
	Generator model.Generator
	Close     func()
}

// Ledger commits admission before any model or tool work; duplicates never run again.
type Ledger interface {
	Admit(context.Context, string, string, string) (Task, bool, error)
	Get(context.Context, string) (Task, error)
	Finish(context.Context, Task) error
}

type Options struct {
	Skills  *skills.Reader
	Project string
	Config  config.Config
	Open    func(context.Context, string, bool, config.Config) (*Conversation, error)
	List    func(context.Context, string, int) ([]sessionstate.Conversation, string, error)
	// PrepareModel reads metadata and returns an owned client without saving selections or credentials.
	PrepareModel func(context.Context, connection.Selection, string, setup.Defaults) (PreparedModel, error)
	UpdateConfig func(context.Context, func(*config.Config)) (config.Config, error)
	Ledger       Ledger
	Responses    ResponseStore
	ReadOutput   func(context.Context, string, int64, int64, int) (string, error)
}

type Task struct{ ID, Session, Status, Error, Code, Persistence string }

const (
	PersistencePending   = "pending"
	PersistenceConfirmed = "confirmed"
	PersistenceUncertain = "uncertain"
)

type Approval struct {
	ID        string
	Truncated bool
	Value     record.Event
}
type Draft struct {
	Agent          string
	Generation     uint64
	Text, Thinking string
	Truncated      bool
}
type ExecutionState struct {
	Session   sessionstate.Session
	Selection connection.Selection
	Team      bool
}

type Snapshot struct {
	Execution  *ExecutionState
	Cursor     string
	Session    sessionstate.Session
	Task       *Task
	Selection  connection.Selection
	Permission string
	Team       bool
	Approvals  []Approval
	Drafts     []Draft
}
type Event struct {
	Cursor, Session, Task string
	Status, Error         string
	Generation            uint64
	Reset, Truncated      bool
	ApprovalID            string
	Record                record.Event
}
type Batch struct {
	Snapshot *Snapshot
	Events   []Event
}

func (s *Service) requireConversation() error {
	if s.closed || s.ctx.Err() != nil {
		return faults.New(diagcatalog.Unavailable, diagcatalog.ServerServerIsShuttingDown)
	}
	if s.conversation == nil {
		return faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerOpenASessionFirst)
	}
	return nil
}
func (s *Service) requireIdle() error {
	if s.closed || s.ctx.Err() != nil {
		return faults.New(diagcatalog.Unavailable, diagcatalog.ServerServerIsShuttingDown)
	}
	if s.active != nil {
		return faults.New(diagcatalog.Conflict, diagcatalog.ServerARequestIsAlreadyRunning)
	}
	if s.completionErr != nil {
		return faults.New(diagcatalog.CompletionNotSaved, diagcatalog.CompletionUncertain)
	}
	if s.executionErr != nil {
		return faults.New(diagcatalog.Internal, diagcatalog.UnexpectedFailure)
	}
	if s.changing {
		return faults.New(diagcatalog.Conflict, diagcatalog.ServerSessionConfigurationIsChanging)
	}
	return nil
}
func checkText(value string, max int) error {
	if len(value) == 0 || len(value) > max {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerTextMustContainToBytes, max)
	}
	return nil
}
