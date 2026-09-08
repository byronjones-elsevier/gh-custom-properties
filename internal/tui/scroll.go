package tui

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// rowContentWidth returns how many columns a list/table row's own text can
// use: the terminal width, minus outerFrameWidth (boxStyle's padding around
// the whole screen) and the row's 2-column cursor/indent prefix ("> " or
// "  "). Returns 0 (unbounded) when width is unknown.
func rowContentWidth(width int) int {
	if width <= 0 {
		return 0
	}
	const rowPrefixWidth = 2
	w := width - outerFrameWidth - rowPrefixWidth
	if w < 10 {
		return 0
	}
	return w
}

// truncateToWidth ANSI-aware-truncates s to at most width visible columns
// (preserving any embedded color codes), appending "…" if it had to cut
// anything. width <= 0 means unbounded (no truncation) — the same
// convention used elsewhere for an unknown terminal size.
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	return ansi.Truncate(s, width, "…")
}

// wrapToWidth wraps a message instead of dropping its tail. It is used for
// errors and other diagnostic text where every character is useful.
func wrapToWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	return ansi.Wrap(s, width, "")
}

// filterIndices returns the indices into items whose value contains filter
// as a case-insensitive substring, preserving order. Shared by optionPicker
// and any plain list/table that adds its own "?" filtering.
func filterIndices(items []string, filter string) []int {
	if filter == "" {
		out := make([]int, len(items))
		for i := range out {
			out[i] = i
		}
		return out
	}
	lf := strings.ToLower(filter)
	var out []int
	for i, item := range items {
		if strings.Contains(strings.ToLower(item), lf) {
			out = append(out, i)
		}
	}
	return out
}

// pinFooter appends footer to content, padding with blank lines first so
// footer lands on the terminal's last row when height is known. If footer
// is empty, or height is unknown, no padding is added — the content is
// simply returned (with footer appended, if any).
func pinFooter(content, footer string, height int) string {
	if footer == "" {
		return content
	}
	if height <= 0 {
		return content + "\n" + footer
	}
	contentLines := strings.Count(content, "\n") + 1
	footerLines := strings.Count(footer, "\n") + 1
	pad := height - contentLines - footerLines
	if pad < 0 {
		pad = 0
	}
	return content + strings.Repeat("\n", pad) + "\n" + footer
}

// visibleWindow returns the [start, end) slice bounds into a list of length
// n that keep index cursor visible within at most maxVisible rows.
// maxVisible <= 0 means "no limit" (the whole list is shown unclipped) —
// this is also the state before the first tea.WindowSizeMsg arrives, e.g.
// in tests that never send one.
func visibleWindow(n, cursor, maxVisible int) (start, end int) {
	if maxVisible <= 0 || n <= maxVisible {
		return 0, n
	}
	start = cursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end = start + maxVisible
	if end > n {
		end = n
		start = end - maxVisible
	}
	return start, end
}

// defaultPageSize is how far Page Up/Down move the cursor when the visible
// window size is unknown (maxVisible <= 0, e.g. before the first
// tea.WindowSizeMsg or in a test that never sends one).
const defaultPageSize = 10

// pageSize returns how far a single Page Up/Down press should move the
// cursor: a full visible window when known, otherwise defaultPageSize.
func pageSize(maxVisible int) int {
	if maxVisible > 0 {
		return maxVisible
	}
	return defaultPageSize
}

// isPageUpKey/isPageDownKey match a tea.KeyMsg.String() against the keys
// TUI.md specifies for paging through a long list: PgUp/PgDn, F7/F8, and
// Shift+Up/Shift+Down.
func isPageUpKey(s string) bool   { return s == "pgup" || s == "f7" || s == "shift+up" }
func isPageDownKey(s string) bool { return s == "pgdown" || s == "f8" || s == "shift+down" }

// availableRows estimates how many list rows fit in the terminal after
// leaving room for the header, a screen title, and a help footer, given the
// last known terminal height. Returns 0 ("unbounded") when height is
// unknown. The chrome estimate is deliberately generous — showing a couple
// fewer rows than would technically fit is harmless, but letting a list run
// past the bottom of the terminal cuts information off entirely.
func availableRows(height int) int {
	if height <= 0 {
		return 0
	}
	const chrome = 10
	n := height - chrome
	if n < 3 {
		n = 3
	}
	return n
}
