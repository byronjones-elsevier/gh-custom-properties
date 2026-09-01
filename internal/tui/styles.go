package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212"))

	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))

	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)

	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("203")).Bold(true)

	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)

	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	boxStyle = lipgloss.NewStyle().Padding(0, 1)
)

// renderHeader renders the persistent top-of-screen banner (the org/repo, or
// batch summary, being edited) shown above every screen's own content.
func renderHeader(text string) string {
	if text == "" {
		return ""
	}
	return headerStyle.Render(text) + "\n" + dimStyle.Render(strings.Repeat("─", len(text))) + "\n\n"
}
