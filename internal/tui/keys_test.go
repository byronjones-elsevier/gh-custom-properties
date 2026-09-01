package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// TestKeyBinding_IsEnabled guards against a regression where keyBinding
// built a key.Binding without WithKeys, which bubbles' Binding.Enabled()
// treats as disabled (it requires a non-nil key list) — silently dropping
// the hint from every helpLine that used it.
func TestKeyBinding_IsEnabled(t *testing.T) {
	b := keyBinding("y", "confirm")
	if !b.Enabled() {
		t.Fatal("keyBinding-produced binding is not Enabled(), so helpLine will drop it")
	}
}

func TestHelpLine_IncludesKeyBindingEntries(t *testing.T) {
	got := helpLine(keyBinding("y", "confirm"), keyBinding("n", "cancel"))
	if got == "" {
		t.Fatal("helpLine of two keyBinding entries should not be empty")
	}
	for _, want := range []string{"y", "confirm", "n", "cancel"} {
		if !strings.Contains(got, want) {
			t.Errorf("helpLine output missing %q: %q", want, got)
		}
	}
}

// TestDefaultListKeyMap_AltVariants guards TUI.md's Alt-A..Alt-Z convention:
// every custom command must also match alt+<letter>, registered
// unconditionally alongside the bare letter (see defaultListKeyMap's
// comment for why that's safe on terminals that never send it).
func TestDefaultListKeyMap_AltVariants(t *testing.T) {
	keys := defaultListKeyMap()
	tests := []struct {
		name    string
		binding key.Binding
		alt     string
	}{
		{"Add", keys.Add, "alt+a"},
		{"Edit", keys.Edit, "alt+e"},
		{"Delete", keys.Delete, "alt+d"},
		{"Save", keys.Save, "alt+s"},
		{"Quit", keys.Quit, "alt+q"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tt.alt[len(tt.alt)-1:]), Alt: true}
			if !key.Matches(msg, tt.binding) {
				t.Errorf("%s binding does not match %q", tt.name, tt.alt)
			}
		})
	}
}
