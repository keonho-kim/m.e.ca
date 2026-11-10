// Purpose: Compose feature-owned terminal state with event-loop dependencies and foreground routing.
// Pattern: Event Loop state.
// Usage: The terminal event loop alone changes app state.
// Related: internal/tui/session/app.go, internal/tui/session/storage.go.
package session

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/media"
	"sync"

	"github.com/gdamore/tcell/v3"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/teamrequest"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/skills"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"github.com/keonho-kim/meca/internal/tui/input"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/keonho-kim/meca/internal/tui/selection"
	"github.com/keonho-kim/meca/internal/tui/textlayout"
)

type Options struct {
	ReadImage    func(context.Context) (media.Image, error)
	Home, State  string
	Config       config.Config
	LoadPrompts  func() (prompt.Templates, error)
	Team         *teamrequest.Service
	LoadTeamMode func(context.Context) (bool, error)
	SaveTeamMode func(context.Context, bool) error
	UpdateConfig func(context.Context, func(*config.Config)) (config.Config, error)
	Store        Store
	Session      sessionstate.Session
	Tools        Tools
	Skills       *skills.Reader
	Files        fileview.Source
	Vault        *credentials.Vault
}
type app struct {
	approvalPreview             approvalPreviewState
	skills                      skillBrowserState
	team                        teamMode
	files                       fileBrowserState
	compose                     requestState
	chat                        chatState
	connection                  connectionState
	work                        workerActivity
	memory                      memoryUsage
	headerTitle                 string
	language                    locale.Language
	languageConfirm             bool
	permissionConfirm           bool
	pasted                      pasteInput
	scrollbar                   scrollbarState
	inputRegion, selectionInput inputRegion
	footerRows                  int
	selectionSources            map[int]textlayout.Fragment
	clipboardBusy               bool
	imagePasteBusy              bool
	selectionPage               string
	textSelection               selection.Selection
	stopNotice                  string
	opts                        Options
	screen                      tcell.Screen
	ctx                         context.Context
	cancel                      context.CancelFunc
	workers                     sync.WaitGroup
	events                      chan notice
	settings                    policy.Settings
	page                        string
	choices                     []string
	choice                      int
	menuInput                   input.Editor
	proposal                    *toolapi.Approval
	decision                    chan bool
	approvalReturn              string
	exit                        bool
	newSession                  bool
	paste                       bool
	status                      string
	previewOffset               int
}
