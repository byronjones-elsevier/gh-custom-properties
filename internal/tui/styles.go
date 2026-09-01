package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Palette holds the color values used to build the package's styles, each a
// lipgloss-compatible color spec (an ANSI 256 color number as a string, or
// a hex code). Kept as a plain tui-local type (rather than importing
// internal/config) so this package doesn't need to know about config
// loading — main.go converts config.Palette into this shape.
type Palette struct {
	Title, Header, Cursor, Selected, Dim, Error, Success, Warn, Help string
}

// DefaultPalette matches the colors this app has always used.
func DefaultPalette() Palette {
	return Palette{
		Title:    "212",
		Header:   "39",
		Cursor:   "212",
		Selected: "212",
		Dim:      "240",
		Error:    "203",
		Success:  "42",
		Warn:     "214",
		Help:     "240",
	}
}

var (
	titleStyle    lipgloss.Style
	headerStyle   lipgloss.Style
	cursorStyle   lipgloss.Style
	selectedStyle lipgloss.Style
	dimStyle      lipgloss.Style
	errorStyle    lipgloss.Style
	successStyle  lipgloss.Style
	warnStyle     lipgloss.Style
	helpStyle     lipgloss.Style
	flashStyle    lipgloss.Style

	boxStyle = lipgloss.NewStyle().Padding(0, 1)
)

func init() {
	ApplyPalette(DefaultPalette())
}

// ApplyPalette (re)builds every color-dependent style from p. Call once at
// startup after loading config, before the program runs.
func ApplyPalette(p Palette) {
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(p.Title))
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(p.Header))
	cursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(p.Cursor)).Bold(true)
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(p.Selected))
	dimStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(p.Dim))
	errorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(p.Error)).Bold(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(p.Success)).Bold(true)
	warnStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(p.Warn))
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(p.Help))
	flashStyle = lipgloss.NewStyle().Bold(true).Reverse(true).Foreground(lipgloss.Color(p.Selected))
}

// renderHeader renders the persistent top-of-screen banner (the org/repo, or
// batch summary, being edited) shown above every screen's own content.
func renderHeader(text string) string {
	if text == "" {
		return ""
	}
	return headerStyle.Render(text) + "\n" + dimStyle.Render(strings.Repeat("─", len(text))) + "\n\n"
}
