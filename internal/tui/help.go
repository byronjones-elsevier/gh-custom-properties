package tui

import "strings"

// renderHelpScreen renders the F1 in-app help overlay: a static keybinding
// reference, distinct from the CLI's -h/--help text (which documents
// command-line flags, not in-TUI usage).
func renderHelpScreen() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("gh-custom-properties — help") + "\n\n")

	b.WriteString(headerStyle.Render("Global") + "\n")
	b.WriteString("  F1                    show/close this help\n")
	b.WriteString("  F5                    refresh from GitHub\n")
	b.WriteString("  ctrl+c / ctrl+q / q   quit (asks to confirm)\n")
	b.WriteString("  pgup/pgdn, f7/f8,\n")
	b.WriteString("  shift+up/shift+down   page through long lists\n")
	b.WriteString("  ?                     filter the current list\n\n")

	b.WriteString(headerStyle.Render("Single-repo screen") + "\n")
	b.WriteString("  a / e / d / s         add / edit / delete / apply changes\n\n")

	b.WriteString(headerStyle.Render("Batch screen") + "\n")
	b.WriteString("  b                     start a bulk edit\n")
	b.WriteString("  space / a / n         toggle / select all / select none (target picker)\n\n")

	b.WriteString(headerStyle.Render("Editing a value") + "\n")
	b.WriteString("  tab                   confirm/advance (same as enter)\n")
	b.WriteString("  shift+tab             back up a step (when adding a property)\n")
	b.WriteString("  esc                   cancel\n\n")

	b.WriteString(helpLine(keyBinding("any key", "close")))
	return boxStyle.Render(b.String())
}
