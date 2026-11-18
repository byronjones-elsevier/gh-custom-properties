// Package clidoc is the single source of truth for gh-custom-properties'
// command-line surface: main.go renders it as the -h/--help text, and
// docs/gendocs renders it as the man page and HTML help file, so the three
// never drift apart.
package clidoc

// Flag describes one command-line flag and all of its name aliases.
type Flag struct {
	Names            []string // e.g. []string{"f", "file"}
	ValuePlaceholder string   // e.g. "path"; empty for boolean flags
	Description      string
}

// Section is a block of free-form help text after the flag list, e.g.
// "AUTHENTICATION" or "BACKUPS".
type Section struct {
	Title string
	Body  string
}

// Spec is the full CLI specification for one command.
type Spec struct {
	Name    string
	Version string
	Date    string // man page date, e.g. "2026-09-01"; bump alongside Version
	Summary string
	Usage   []string
	Note    string // shown right after Usage, e.g. behavior when args are omitted
	Flags   []Flag
	Section []Section
}

// GHCustomProperties is the CLI spec for the gh-custom-properties command.
var GHCustomProperties = Spec{
	Name:    "gh-custom-properties",
	Version: "1.0",
	Date:    "2026-09-16",
	Summary: "view, add, edit, and delete GitHub custom repo properties",
	Usage: []string{
		"gh-custom-properties [flags] [owner/repo | repo-url]",
		"gh-custom-properties --file repos.txt [flags]",
		"gh-custom-properties --list [flags] owner/repo",
		"gh-custom-properties --set name=value [--set name=value ...] [flags] owner/repo",
	},
	Note: "If no repo argument or --file is given, the TUI prompts for a repo to load.",
	Flags: []Flag{
		{Names: []string{"f", "file"}, ValuePlaceholder: "path", Description: "batch mode: file listing one repo (owner/repo or URL) per line"},
		{Names: []string{"list"}, Description: "non-interactive: print the repo's custom properties as name=value lines and exit"},
		{Names: []string{"set"}, ValuePlaceholder: "name=value", Description: "non-interactive: set a property on the repo; repeat for multiple. Multi-select values are comma-separated; an empty value clears the property"},
		{Names: []string{"t", "token"}, ValuePlaceholder: "token", Description: "GitHub token, overriding GITHUB_TOKEN / `gh auth token`"},
		{Names: []string{"backup-dir"}, ValuePlaceholder: "path", Description: "directory to write timestamped backups to (default $HOME/.gh-custom-properties/backups)"},
		{Names: []string{"h", "H", "help", "HELP", "?"}, Description: "show help"},
		{Names: []string{"v", "version"}, Description: "show version"},
	},
	Section: []Section{
		{
			Title: "AUTHENTICATION",
			Body: "A GitHub token is required, resolved in this order:\n" +
				"  1. --token\n" +
				"  2. GITHUB_TOKEN environment variable\n" +
				"  3. `gh auth token` (if the gh CLI is installed and logged in)\n\n" +
				"The token needs permission to read and write custom properties on the\n" +
				"target repos, and to read the org's custom-property schema.",
		},
		{
			Title: "REPO LIST FILE FORMAT (--file)",
			Body: "One repo per line, as \"owner/repo\" or a github.com URL. Blank lines and\n" +
				"lines starting with '#' are ignored.",
		},
		{
			Title: "NON-INTERACTIVE MODE (--list, --set)",
			Body: "Both require a repo argument and cannot be combined with --file. With\n" +
				"both, --set is applied first and the updated properties are listed.\n" +
				"Values are validated against the org's property schema when readable\n" +
				"(unknown names, disallowed select values, non-boolean true_false values,\n" +
				"and clearing required properties are rejected before anything is written).\n" +
				"--list output goes to stdout; --set progress and the backup path go to stderr.\n\n" +
				"Example:\n" +
				"  gh-custom-properties --set team=platform --set tags=a,b --list my-org/my-repo",
		},
		{
			Title: "BACKUPS",
			Body: "Before any change is applied, a timestamped JSON snapshot of the prior\n" +
				"property values is written to the backup directory.",
		},
	},
}
