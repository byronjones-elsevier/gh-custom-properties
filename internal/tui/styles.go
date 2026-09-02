package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
	menubar "github.com/jejacks0n/bubbletea-menubar"
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
	keyHintStyle  lipgloss.Style
	topBarStyles  menubar.Styles

	boxStyle         = lipgloss.NewStyle().Padding(0, 1)
	footerPanelStyle lipgloss.Style
)

// outerFrameWidth is how many columns boxStyle's own horizontal padding
// adds around every screen's whole rendered output. Anything sized to the
// terminal's actual width — the top bar, the footer panel's frame — has to
// account for this too, or it'll overflow the real terminal by that much
// once boxStyle wraps around it.
const outerFrameWidth = 2 // Padding(0, 1): 1 column each side

// footerFrameWidth is how many columns the boxed footer panel's own
// border+padding, plus outerFrameWidth, take up — subtracted from the
// terminal width to get the panel's usable content width.
const footerFrameWidth = 4 + outerFrameWidth // 2 (footer border) + 2 (footer padding)

// footerContentWidth returns how many columns are available inside the
// boxed footer panel for a given terminal width, or 0 ("unbounded" — let
// it hug its content) when termWidth is unknown or too narrow to bother
// constraining.
func footerContentWidth(termWidth int) int {
	w := termWidth - footerFrameWidth
	if w < 10 {
		return 0
	}
	return w
}

// newHelpModel builds a bubbles/help Model styled to match the current
// palette (key hints highlighted via keyHintStyle, same as the top bar's
// hotkey underline serves for the top bar) for use in a boxed footer panel.
func newHelpModel() help.Model {
	h := help.New()
	h.Styles.ShortKey = keyHintStyle
	h.Styles.FullKey = keyHintStyle
	h.Styles.ShortDesc = helpStyle
	h.Styles.FullDesc = helpStyle
	h.Styles.ShortSeparator = helpStyle
	h.Styles.FullSeparator = helpStyle
	h.Styles.Ellipsis = helpStyle
	return h
}

// renderFooterPanel renders a screen's command reference as a bordered
// panel pinned to the terminal's last row (see pinFooter). It tries the
// compact single-line form first; if that wouldn't fit within width without
// truncating, it wraps onto as many additional lines as needed instead, so
// every command stays visible rather than any being elided.
//
// help.Model's own FullHelpView (fixed-height columns, ellipsis-truncated)
// was tried here first, but its width-fitting has a gap: if even a single
// column doesn't fit and there's no room left for an ellipsis marker, it
// falls through to rendering that oversized column anyway (see its
// shouldAddItem), so it doesn't reliably guarantee everything fits.
// wrapShortHelp guarantees it by packing plain "key desc" hints — using the
// same Styles as the short/full views, for a consistent look — onto
// multiple lines itself.
func renderFooterPanel(h help.Model, width int, keymap help.KeyMap) string {
	contentWidth := footerContentWidth(width)

	measure := h
	measure.Width = 0
	natural := measure.ShortHelpView(keymap.ShortHelp())

	body := natural
	if contentWidth > 0 && lipgloss.Width(natural) > contentWidth {
		body = wrapShortHelp(h, keymap.ShortHelp(), contentWidth)
	}
	return footerPanelStyle.Render(body)
}

// wrapShortHelp lays out bindings as "key desc" hints, greedily packing as
// many onto each line as fit within width before starting a new one, so
// every rendered line stays within width regardless of how many bindings
// there are — unlike help.Model.FullHelpView's column layout (see
// renderFooterPanel's comment).
func wrapShortHelp(h help.Model, bindings []key.Binding, width int) string {
	type segment struct{ plain, styled string }

	segs := make([]segment, 0, len(bindings))
	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		bh := b.Help()
		segs = append(segs, segment{
			plain:  bh.Key + " " + bh.Desc,
			styled: h.Styles.ShortKey.Render(bh.Key) + " " + h.Styles.ShortDesc.Render(bh.Desc),
		})
	}
	if len(segs) == 0 {
		return ""
	}

	sepWidth := lipgloss.Width(h.ShortSeparator)
	sepStyled := h.Styles.ShortSeparator.Render(h.ShortSeparator)

	var lines []string
	var lineParts []string
	lineWidth := 0
	flush := func() {
		if len(lineParts) > 0 {
			lines = append(lines, strings.Join(lineParts, sepStyled))
		}
	}
	for _, s := range segs {
		extra := lipgloss.Width(s.plain)
		if len(lineParts) > 0 {
			extra += sepWidth
		}
		if len(lineParts) > 0 && lineWidth+extra > width {
			flush()
			lineParts = nil
			lineWidth = 0
			extra = lipgloss.Width(s.plain)
		}
		lineParts = append(lineParts, s.styled)
		lineWidth += extra
	}
	flush()
	return strings.Join(lines, "\n")
}

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
	keyHintStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(p.Cursor))
	footerPanelStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color(p.Dim)).
		Padding(0, 1)

	bg := lipgloss.Color(p.Header)
	fg := lipgloss.Color("255")
	topBarStyles = menubar.Styles{
		Bar:       lipgloss.NewStyle().Background(bg).Foreground(fg),
		Item:      lipgloss.NewStyle().Padding(0, 1).Background(bg).Foreground(fg),
		Shortcut:  lipgloss.NewStyle().Background(bg).Foreground(fg),
		Hotkey:    lipgloss.NewStyle().Bold(true).Underline(true),
		Separator: lipgloss.NewStyle().Padding(0, 1).Background(bg).Foreground(fg),
		Disabled:  lipgloss.NewStyle().Padding(0, 1).Background(bg).Foreground(lipgloss.Color(p.Dim)),
		// SelectedItem/Dropdown*/ShortcutSelected are left at their zero
		// value: renderTopBar always keeps the menu inactive (see its
		// comment), so nothing ever selects an item or opens a dropdown.
	}
}

// renderTopBar renders the persistent top bar: relevant commands — as a
// bubbletea-menubar Model, used purely for its label-plus-underlined-hotkey
// rendering — with the owner/repo (or batch org/count) identity
// right-aligned. The menu is always constructed with Active left false:
// our own key handlers already dispatch every one of these commands (and
// have to, since e.g. "?" means something different depending on whether
// the property list is currently filtering), so wiring up the package's
// own navigation/activation would create a second, conflicting dispatch
// path for the same keys. This is display-only.
//
// ViewBarWithRightSide only uses width to decide how much spacer padding
// to add before the right-aligned text — it doesn't truncate or wrap the
// items themselves, so a wide item set (ours needs ~75-80 columns) still
// overflows a narrower-but-still-usable terminal (App's own minimum is
// 60). If the rendered bar would be wider than the terminal (accounting
// for outerFrameWidth, since the bar is rendered inside the same
// boxStyle-padded view as everything else), fall back to dropping the
// items and showing just the identity text — the footer panel
// (renderFooterPanel) is the guaranteed-to-fit place to find the commands
// regardless.
func renderTopBar(items []menubar.MenuItem, right string, width int) string {
	barWidth := width
	if barWidth > 0 {
		barWidth -= outerFrameWidth
	}

	mb := menubar.New(items)
	mb.Active = false
	mb.Styles = topBarStyles

	bar := mb.ViewBarWithRightSide(right, barWidth)
	if barWidth > 0 && lipgloss.Width(bar) > barWidth {
		mb.Items = nil
		bar = mb.ViewBarWithRightSide(right, barWidth)
	}
	return bar
}
