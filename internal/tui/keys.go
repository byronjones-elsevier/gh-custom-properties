package tui

import "github.com/charmbracelet/bubbles/key"

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
// by a direct switch on msg.String()).
func keyBinding(keyStr, desc string) key.Binding {
	return key.NewBinding(key.WithHelp(keyStr, desc))
}

// helpLine renders a compact single-line help footer from binding help text.
func helpLine(bindings ...key.Binding) string {
	parts := make([]string, 0, len(bindings))
	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		h := b.Help()
		parts = append(parts, h.Key+" "+h.Desc)
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "  •  "
		}
		out += p
	}
	return helpStyle.Render(out)
}
