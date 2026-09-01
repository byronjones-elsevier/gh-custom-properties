// Package tui implements the Bubble Tea interface for viewing, adding,
// editing, and deleting GitHub custom repository properties, either for a
// single repo or across a batch of repos loaded from a file.
package tui

import (
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/repolist"
	tea "github.com/charmbracelet/bubbletea"
)

// innerModel is the subset of tea.Model that both singleRepoModel and
// batchModel satisfy, plus a way to tell the outer program to exit cleanly.
type innerModel interface {
	tea.Model
	Quitting() bool
}

// App is the root Bubble Tea model, delegating to either a single-repo or
// batch sub-model depending on how it was constructed.
type App struct {
	inner innerModel
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

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok && km.Type == tea.KeyCtrlC {
		return a, tea.Quit
	}
	next, cmd := a.inner.Update(msg)
	inner, ok := next.(innerModel)
	if !ok {
		return a, cmd
	}
	a.inner = inner
	return a, cmd
}

func (a *App) View() string {
	return a.inner.View()
}

var (
	_ tea.Model  = (*App)(nil)
	_ innerModel = (*singleRepoModel)(nil)
	_ innerModel = (*batchModel)(nil)
)
