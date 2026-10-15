// Purpose: Own active terminal state and injected application dependencies.
// Pattern: Event Loop state.
// Usage: The terminal event loop alone changes app state.
// Related: internal/tui/session/app.go, internal/tui/session/storage.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/commands"
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/config/credentials"
	"github.com/keonho-kim/v-code/internal/policy"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	sessionstate "github.com/keonho-kim/v-code/internal/storage/session"
	"github.com/keonho-kim/v-code/internal/tools/local"
	"github.com/keonho-kim/v-code/internal/tui/browser"
	"github.com/keonho-kim/v-code/internal/tui/input"
	"github.com/keonho-kim/v-code/internal/tui/viewport"
	"sync"
	"time"
)

type Options struct {
	Home, State string
	Config      config.Config
	LoadPrompts func() (prompt.Templates, error)
	SaveConfig  func(config.Config) error
	Store       Store
	Session     sessionstate.Session
	Tools       *local.Executor
	Vault       *credentials.Vault
}
type app struct {
	filePreview                *browser.Preview
	previewCancel              context.CancelFunc
	previewRevision            int
	elapsedStarted             time.Time
	elapsed                    time.Duration
	elapsedTimer               *time.Ticker
	elapsedTicks               <-chan time.Time
	stopNotice                 string
	tokensPerSecond            *float64
	modelNames, modelHints     []string
	modelHintsOpen             bool
	modelHintIndex             int
	pending                    string
	agentActive, stopRequested bool
	compressionTimer           *time.Ticker
	compressionTicks           <-chan time.Time
	compressionDots            int
	tokenTotal, tokenCurrent   modelapi.TokenUsage
	opts                       Options
	screen                     tcell.Screen
	ctx                        context.Context
	cancel                     context.CancelFunc
	workers                    sync.WaitGroup
	events                     chan notice
	editor                     input.Editor
	output                     viewport.View
	inputs                     []string
	historyIndex               int
	draft                      string
	settings                   policy.Settings
	selection                  modelconn.Selection
	client                     *httpclient.Client
	plan                       *planning.Plan
	planMode                   bool
	planStopping               bool
	thinkingChoice             bool
	page                       string
	choices                    []string
	choice                     int
	menuInput                  input.Editor
	connectionKind             int
	endpoint, key              string
	browser                    browser.Browser
	proposal                   *toolapi.Approval
	decision                   chan bool
	busy                       bool
	taskCancel                 context.CancelFunc
	exit                       bool
	newSession                 bool
	paste                      bool
	status, stream, thinking   string
	commandHints               []commands.Suggestion
	commandOpen                bool
	commandIndex               int
	changeField, changeValue   string
	historyCursor              int64
	previewOffset              int
	pendingConnection          modelconn.Connection
}
