// Package tui implements the Bubble Tea interface for viewing, adding,
// editing, and deleting GitHub custom repository properties, either for a
// single repo or across a batch of repos loaded from a file.
package tui

import (
	"fmt"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/repolist"
	tea "github.com/charmbracelet/bubbletea"
)

// minWidth/minHeight are the smallest terminal dimensions this app's
// screens are designed to render sensibly in; below this, App.View shows
// a warning instead of clipped/garbled content.
const (
	minWidth  = 60
	minHeight = 12
)

// innerModel is the subset of tea.Model that both singleRepoModel and
// batchModel satisfy.
type innerModel interface {
	tea.Model
}

// overlayKind identifies an App-level modal that takes over the whole
// screen regardless of which inner screen is active underneath it.
type overlayKind int

const (
	overlayNone overlayKind = iota
	overlayQuitConfirm
	overlayHelp
)

// quitRequestedMsg is emitted (via a returned tea.Cmd) by an inner model
// when the user presses a quit-intent key that can't be safely intercepted
// globally — bare "q", which must stay scoped to screens with no active
// text input so it doesn't swallow a literal "q" being typed. Cmd-produced
// messages route back through App.Update before ever reaching the inner
// model again (the same mechanism loadedMsg/appliedMsg already rely on),
// so App can still gate it behind the same confirmation as Ctrl-C/Ctrl-Q.
type quitRequestedMsg struct{}

func quitRequestedCmd() tea.Msg { return quitRequestedMsg{} }

// App is the root Bubble Tea model, delegating to either a single-repo or
// batch sub-model depending on how it was constructed, and layering
// global concerns (quit confirmation, help, minimum terminal size) on top.
type App struct {
	inner         innerModel
	overlay       overlayKind
	width, height int
}

// NewSingleRepo builds an App for the single-repo add/edit/delete flow.
// ownerRepo may be empty, in which case the TUI prompts for a repo.
func NewSingleRepo(api ghclient.PropertiesAPI, backupDir, owner, repo string) *App {
	return &App{inner: newSingleRepoModel(api, backupDir, owner, repo)}
}

// NewBatch builds an App for the batch bulk-edit flow over the given
// already-parsed repo list.
func NewBatch(api ghclient.PropertiesAPI, backupDir string, entries []repolist.Entry, skipped []repolist.Skipped) *App {
	return &App{inner: newBatchModel(api, backupDir, entries, skipped)}
}

func (a *App) Init() tea.Cmd {
	return a.inner.Init()
}

func isQuitIntentKey(km tea.KeyMsg) bool {
	return km.Type == tea.KeyCtrlC || km.String() == "ctrl+q"
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		a.width, a.height = wsm.Width, wsm.Height
	}

	if km, ok := msg.(tea.KeyMsg); ok {
		if a.overlay != overlayNone {
			return a, a.handleOverlayKey(km)
		}
		if isQuitIntentKey(km) {
			a.overlay = overlayQuitConfirm
			return a, nil
		}
		if km.Type == tea.KeyF1 {
			a.overlay = overlayHelp
			return a, nil
		}
	}

	if _, ok := msg.(quitRequestedMsg); ok {
		a.overlay = overlayQuitConfirm
		return a, nil
	}

	next, cmd := a.inner.Update(msg)
	inner, ok := next.(innerModel)
	if !ok {
		return a, cmd
	}
	a.inner = inner
	return a, cmd
}

// handleOverlayKey handles a keypress while an overlay is showing, instead
// of forwarding it to the inner model.
func (a *App) handleOverlayKey(km tea.KeyMsg) tea.Cmd {
	switch a.overlay {
	case overlayQuitConfirm:
		switch km.String() {
		case "y", "enter", "ctrl+c", "ctrl+q":
			return tea.Quit
		default:
			a.overlay = overlayNone
		}
	case overlayHelp:
		a.overlay = overlayNone
	}
	return nil
}

func (a *App) View() string {
	if a.width > 0 && a.height > 0 && (a.width < minWidth || a.height < minHeight) {
		return fmt.Sprintf(
			"Terminal too small (%dx%d).\nPlease resize to at least %dx%d.",
			a.width, a.height, minWidth, minHeight,
		)
	}
	switch a.overlay {
	case overlayQuitConfirm:
		return renderQuitConfirm()
	case overlayHelp:
		return renderHelpScreen()
	}
	return a.inner.View()
}

func renderQuitConfirm() string {
	return boxStyle.Render(
		warnStyle.Render("Quit gh-custom-properties?") + "\n\n" +
			helpLine(keyBinding("y/enter", "confirm"), keyBinding("any other key", "cancel")),
	)
}

var (
	_ tea.Model  = (*App)(nil)
	_ innerModel = (*singleRepoModel)(nil)
	_ innerModel = (*batchModel)(nil)
)
