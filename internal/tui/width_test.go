package tui

import (
	"strings"
	"testing"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func maxLineWidth(s string) int {
	max := 0
	for _, line := range strings.Split(s, "\n") {
		if lw := lipgloss.Width(line); lw > max {
			max = lw
		}
	}
	return max
}

// TestView_NeverOverflowsTerminalWidth renders through the full View()
// (top bar, content, footer panel, all inside the outer boxStyle wrap) at
// every width App actually allows through its own minimum-size gate (60),
// guarding against the top bar/footer/row-formatting each overflowing the
// real terminal — found during development: the third-party menu bar's
// ViewBarWithRightSide doesn't truncate its own items to fit, and the
// property list/repo table's fixed-width row formatting (%-30s/%-40s)
// didn't account for terminal width at all.
func TestView_NeverOverflowsTerminalWidth(t *testing.T) {
	widths := []int{60, 70, 80, 90, 120}

	t.Run("single-repo", func(t *testing.T) {
		m := newLoadedModel(t, manyProperties(3), nil, ghclient.ErrSchemaUnavailable)
		for _, w := range widths {
			next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: 22})
			m = next.(*singleRepoModel)
			if got := maxLineWidth(m.View()); got > w {
				t.Errorf("width=%d: rendered a line %d columns wide", w, got)
			}
		}
	})

	t.Run("batch", func(t *testing.T) {
		m := newTableModel(t, &fakeAPI{}, manyRows(3), nil)
		for _, w := range widths {
			next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: 22})
			m = next.(*batchModel)
			if got := maxLineWidth(m.View()); got > w {
				t.Errorf("width=%d: rendered a line %d columns wide", w, got)
			}
		}
	})
}
