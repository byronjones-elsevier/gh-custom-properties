package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// listKeyMap is the key set for a property-list-style screen (single-repo
// property list, batch repo table).
type listKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Add    key.Binding
	Edit   key.Binding
	Delete key.Binding
	Save   key.Binding
	Cancel key.Binding
	Quit   key.Binding
}

func defaultListKeyMap() listKeyMap {
	return listKeyMap{
		Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Add:    key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Edit:   key.NewBinding(key.WithKeys("enter", "e"), key.WithHelp("enter/e", "edit")),
		Delete: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Save:   key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "apply changes")),
		Cancel: key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back")),
		Quit:   key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	}
}

// keyBinding builds a display-only key.Binding for use in helpLine when no
// key.Matches dispatch is needed (e.g. ad hoc confirm/quit prompts handled
// by a direct switch on msg.String()). WithKeys is set to the same display
// string purely so bubbles' Binding.Enabled() (which requires a non-nil key
// list) reports true and helpLine doesn't silently drop it — these bindings
// are never passed through key.Matches, so it has no effect on dispatch.
func keyBinding(keyStr, desc string) key.Binding {
	return key.NewBinding(key.WithKeys(keyStr), key.WithHelp(keyStr, desc))
}

// helpLine renders a compact single-line help footer from binding help text.
func helpLine(bindings ...key.Binding) string {
	return helpLineFlash("", bindings...)
}

// helpLineFlash is helpLine, but the segment whose help key matches
// flashLabel (if any) renders with flashStyle instead of helpStyle —
// briefly acknowledging a just-pressed command key before its action runs
// (see pendingFlash). Segments are styled individually, then joined
// unstyled, rather than wrapping the whole joined string in one Render
// call: nesting a styled flashStyle segment inside an outer helpStyle
// Render would have the inner segment's ANSI reset code kill the outer
// style for everything after it.
func helpLineFlash(flashLabel string, bindings ...key.Binding) string {
	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		h := b.Help()
		text := h.Key + " " + h.Desc
		if flashLabel != "" && h.Key == flashLabel {
			parts = append(parts, flashStyle.Render(text))
		} else {
			parts = append(parts, helpStyle.Render(text))
		}
	}
	sep := helpStyle.Render("  •  ")
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}

// flashDuration is how long a hub screen's just-pressed command key stays
// highlighted in the footer before its action actually runs.
const flashDuration = 100 * time.Millisecond

// flashElapsedMsg fires flashDuration after a command key sets pendingFlash.
type flashElapsedMsg struct{}

func flashTick() tea.Cmd {
	return tea.Tick(flashDuration, func(time.Time) tea.Msg { return flashElapsedMsg{} })
}

// pendingFlash defers a hub screen's command-key action until flashTick
// fires, so the footer can highlight the pressed key first instead of the
// screen changing instantly underneath it.
type pendingFlash struct {
	label  string
	action func() (tea.Model, tea.Cmd)
}
