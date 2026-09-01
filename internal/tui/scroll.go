package tui

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
