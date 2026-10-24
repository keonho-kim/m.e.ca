// Purpose: Own active terminal state and injected application dependencies.
// Pattern: Event Loop state.
// Usage: The terminal event loop alone changes app state.
// Related: internal/tui/session/app.go, internal/tui/session/storage.go.
package session

import (
	"context"
	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/policy"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"github.com/keonho-kim/meca/internal/tui/browser"
	"github.com/keonho-kim/meca/internal/tui/input"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/selection"
	"github.com/keonho-kim/meca/internal/tui/textlayout"
	"github.com/keonho-kim/meca/internal/tui/viewport"
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
	Tools       Tools
	Vault       *credentials.Vault
}
type app struct {
	language                            locale.Language
	languageConfirm                     bool
	steering                            chan string
	steeringDrafts                      []string
	chatVisible, selectedTranscript     []viewport.Line
	historyLoading, historyEnd          bool
	directoryReads                      readSlot[directoryPage]
	directoryRevision                   int
	directoryLoading, fileSelectionBusy bool
	fileSelectionQueue                  []string
	fileSubmission                      *fileSubmission
	fileSendConfirm                     bool
	fileReserve                         bool
	referenceRevision                   uint64
	references                          referenceProjection
	pasted                              pasteInput
	setupRevision                       int
	chatProjection                      chatProjection
	scrollbar                           scrollbarState
	inputRegion, selectionInput         inputRegion
	footerRows                          int
	selectionSources                    map[int]textlayout.Fragment
	clipboardBusy                       bool
	selectionPage                       string
	textSelection                       selection.Selection
	filePreview                         *browser.Preview
	previewReads                        readSlot[previewRequest]
	previewRevision                     int
	elapsedStarted                      time.Time
	elapsed                             time.Duration
	elapsedTimer                        *time.Ticker
	elapsedTicks                        <-chan time.Time
	stopNotice                          string
	tokensPerSecond                     *float64
	pending                             string
	worker                              workerState
	compressionTimer                    *time.Ticker
	compressionTicks                    <-chan time.Time
	compressionDots                     int
	tokenTotal                          modelapi.TokenUsage
	tokenCurrent                        modelapi.TokenUsage
	modelForm                           *modelForm
	runSummary                          *viewport.Block
	opts                                Options
	screen                              tcell.Screen
	ctx                                 context.Context
	cancel                              context.CancelFunc
	workers                             sync.WaitGroup
	events                              chan notice
	editor                              input.Editor
	output                              viewport.View
	inputs                              []string
	historyIndex                        int
	draft                               string
	settings                            policy.Settings
	selection                           modelconn.Selection
	client                              *httpclient.Client
	page                                string
	choices                             []string
	choice                              int
	menuInput                           input.Editor
	browser                             browser.Browser
	proposal                            *toolapi.Approval
	decision                            chan bool
	taskCancel                          context.CancelFunc
	exit                                bool
	newSession                          bool
	paste                               bool
	status, stream, thinking            string
	commandHints                        []commands.Suggestion
	commandOpen                         bool
	commandIndex                        int
	historyCursor                       int64
	historyRevision                     int
	previewOffset                       int
}
