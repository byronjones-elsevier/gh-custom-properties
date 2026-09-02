package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func isQuitCmd(t *testing.T, cmd tea.Cmd) bool {
	t.Helper()
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestApp_CtrlCShowsConfirmThenQuits(t *testing.T) {
	a := NewSingleRepo(&fakeAPI{}, t.TempDir(), "octocat", "hello-world")

	next, cmd := a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	a = next.(*App)
	if isQuitCmd(t, cmd) {
		t.Fatal("first ctrl+c should show a confirmation, not quit immediately")
	}
	if !strings.Contains(a.View(), "Quit gh-custom-properties?") {
		t.Errorf("expected quit-confirm overlay, got:\n%s", a.View())
	}

	_, cmd = a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !isQuitCmd(t, cmd) {
		t.Error("second ctrl+c while confirming should quit")
	}
}

func TestApp_QuitConfirm_AnyOtherKeyCancels(t *testing.T) {
	a := NewSingleRepo(&fakeAPI{}, t.TempDir(), "octocat", "hello-world")

	next, _ := a.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	a = next.(*App)

	next, cmd := a.Update(runeKey('x'))
	a = next.(*App)
	if isQuitCmd(t, cmd) {
		t.Fatal("an unrelated key should cancel, not quit")
	}
	if strings.Contains(a.View(), "Quit gh-custom-properties?") {
		t.Errorf("overlay should be dismissed:\n%s", a.View())
	}
}

func TestApp_QuitRequestedMsgFromInnerShowsConfirm(t *testing.T) {
	a := NewSingleRepo(&fakeAPI{}, t.TempDir(), "octocat", "hello-world")

	next, _ := a.Update(quitRequestedMsg{})
	a = next.(*App)
	if !strings.Contains(a.View(), "Quit gh-custom-properties?") {
		t.Errorf("expected quit-confirm overlay from quitRequestedMsg, got:\n%s", a.View())
	}
}

func TestApp_F1TogglesHelp(t *testing.T) {
	a := NewSingleRepo(&fakeAPI{}, t.TempDir(), "octocat", "hello-world")

	next, _ := a.Update(tea.KeyMsg{Type: tea.KeyF1})
	a = next.(*App)
	if !strings.Contains(a.View(), "help") {
		t.Errorf("expected help overlay, got:\n%s", a.View())
	}

	next, _ = a.Update(runeKey('x'))
	a = next.(*App)
	if strings.Contains(a.View(), "gh-custom-properties — help") {
		t.Errorf("help overlay should have closed:\n%s", a.View())
	}
}

func TestApp_MinimumSizeWarning(t *testing.T) {
	a := NewSingleRepo(&fakeAPI{}, t.TempDir(), "octocat", "hello-world")

	next, _ := a.Update(tea.WindowSizeMsg{Width: 20, Height: 5})
	a = next.(*App)
	if !strings.Contains(a.View(), "too small") {
		t.Errorf("expected a too-small warning, got:\n%s", a.View())
	}

	next, _ = a.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	a = next.(*App)
	if strings.Contains(a.View(), "too small") {
		t.Errorf("did not expect a too-small warning at 80x24:\n%s", a.View())
	}
}
