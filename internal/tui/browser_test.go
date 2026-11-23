package tui

import (
	"testing"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/browser"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	tea "github.com/charmbracelet/bubbletea"
)

// stubBrowser replaces browser.Open for the test and returns a pointer to
// the URLs it was asked to open.
func stubBrowser(t *testing.T, err error) *[]string {
	t.Helper()
	var opened []string
	orig := browser.Open
	browser.Open = func(u string) error { opened = append(opened, u); return err }
	t.Cleanup(func() { browser.Open = orig })
	return &opened
}

// runCmd executes cmd and returns its message (nil when cmd is nil).
func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func TestSingleRepoModel_OKeyOpensCustomPropertiesPage(t *testing.T) {
	tests := []struct {
		name    string
		openErr error
		wantErr bool
	}{
		{name: "opens page"},
		{name: "surfaces launch failure", openErr: errString("no browser"), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opened := stubBrowser(t, tt.openErr)
			m := newLoadedModel(t, manyProperties(2), nil, ghclient.ErrSchemaUnavailable)

			next, _ := m.Update(runeKey('o'))
			_, cmd := next.(*singleRepoModel).Update(flashElapsedMsg{})
			msg := runCmd(cmd)
			next, _ = next.Update(msg)

			want := "https://github.com/octocat/hello-world/settings/custom-properties"
			if len(*opened) != 1 || (*opened)[0] != want {
				t.Fatalf("opened = %v, want [%s]", *opened, want)
			}
			if gotErr := next.(*singleRepoModel).err != nil; gotErr != tt.wantErr {
				t.Errorf("err set = %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestBatchModel_OKeyOpensSelectedRepo(t *testing.T) {
	opened := stubBrowser(t, nil)
	m := newTableModel(t, &fakeAPI{}, manyRows(3), nil)
	m.cursor = 1

	next, _ := m.Update(runeKey('o'))
	_, cmd := next.(*batchModel).Update(flashElapsedMsg{})
	runCmd(cmd)

	want := "https://github.com/acme/repo-01/settings/custom-properties"
	if len(*opened) != 1 || (*opened)[0] != want {
		t.Errorf("opened = %v, want [%s]", *opened, want)
	}
}

type errString string

func (e errString) Error() string { return string(e) }
