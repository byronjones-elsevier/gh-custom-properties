package tui

import (
	"strings"
	"testing"
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
