# gh-custom-properties

A terminal UI for viewing, adding, editing, and deleting [GitHub custom
properties](https://docs.github.com/en/organizations/managing-organization-settings/managing-custom-properties-for-repositories-in-your-organization)
on a repository — either one repo at a time, or in bulk across a list of
repos loaded from a file. Every change is preceded by a timestamped backup
of the prior values.

## Install

```sh
go install github.com/ByronJones-Elsevier/gh-custom-properties@latest
```

Or build from a checkout:

```sh
go build -o gh-custom-properties .
```

## Authentication

A GitHub token is required, resolved in this order:

1. `--token`
2. `GITHUB_TOKEN` environment variable
3. `gh auth token` (if the [gh CLI](https://cli.github.com/) is installed and logged in)

The token needs permission to read and write custom properties on the
target repos (`repo` scope, or the equivalent fine-grained "Custom
properties" permission), and organization membership/read access to fetch
the org's custom-property schema. If the schema can't be read, the tool
still works — it just falls back to freeform text entry for property
values instead of type-aware pickers.

## Usage

Every screen has a top bar (commands on the left, the repo/org context
right-aligned) and, on the property list and repo table, a boxed command
reference pinned to the bottom — it shows one line when everything fits,
wrapping onto more lines rather than eliding any command if the terminal's
too narrow for one line.

Single repo, interactively:

```sh
gh-custom-properties
```

Single repo, given up front:

```sh
gh-custom-properties octocat/hello-world
gh-custom-properties https://github.com/octocat/hello-world
```

Within the single-repo screen:

| Key | Action |
|---|---|
| `↑`/`↓` or `j`/`k` | move selection |
| `pgup`/`pgdn`, `f7`/`f8`, `shift+↑`/`shift+↓` | page through a long list |
| `a` | add a property (from the org schema, or freeform if unavailable) |
| `enter`/`e` | edit the selected property's value |
| `d` | delete the selected property (confirm with `y`) |
| `s` | apply all changes (writes a backup first, then patches GitHub) |
| `?` | filter the list by name (typing narrows it live; `esc` clears) |
| `q` | quit (confirm with `y`/`enter`; any other key cancels) |

See [Global keybindings](#global-keybindings) below for `F1`/`F5`/Ctrl-C/Ctrl-Q, which work on every screen.

Batch mode, across a list of repos:

```sh
gh-custom-properties --file repos.txt
```

`repos.txt` lists one repo per line, as `owner/repo` or a `github.com` URL.
Blank lines and lines starting with `#` are ignored:

```
# platform team repos
octocat/hello-world
octocat/other-repo
https://github.com/octocat/a-third-repo
```

The batch screen loads and summarizes every repo's current properties, then
`b` opens the bulk-edit flow: choose to set or delete a property, pick its
value (again type-aware when the org schema is available), choose which
loaded repos to target (defaults to all), and apply. You can run several
bulk edits in one session before quitting; each apply writes its own
backup.

Batch mode assumes the repo list is drawn from a single GitHub org — the
bulk-edit property picker is built from the first successfully loaded
repo's org schema. If a repo in a different org doesn't recognize the
chosen property, that repo's apply result will show the API error rather
than being silently skipped.

On the repo table: `↑`/`↓`/`j`/`k` and paging move the selection, `?`
filters the table by `owner/repo`, `b` starts a bulk edit, `F5` re-fetches
every repo from GitHub (discarding nothing — batch mode never stages
unapplied edits), and `q` quits.

## Global keybindings

These work on every screen, in both modes:

| Key | Action |
|---|---|
| `F1` | show/close a keybinding reference |
| `F5` | refresh from GitHub (property list / repo table only) |
| `ctrl+c` / `ctrl+q` / `q`\* | quit — asks to confirm; a second press (or `y`/`enter`) confirms, any other key cancels |
| `tab` | in the add/edit editor, same as `enter` (confirm/advance) |
| `shift+tab` | in the add editor's value step, go back and pick a different property |

\* `q` only means quit on the property list / repo table screens — elsewhere (e.g. typing a repo name or a property value) it's just the letter q. `ctrl+q` is intercepted by some terminals for flow control; `ctrl+c` is the more universally reliable path.

Every custom command above (`a`/`e`/`d`/`s`/`q`/`b`) also matches
`alt+<letter>`, honoring the equivalent of the common "Alt-key shortcut"
terminal convention. This only does something on terminals that support
the [Kitty keyboard protocol](https://sw.kovidgoyal.net/kitty/keyboard-protocol/)
(kitty, WezTerm, Ghostty, newer iTerm2 builds) or that have "Use Option as
Meta Key" enabled — notably **not** macOS's default Terminal.app — so the
bare letter is always there as the reliable fallback.

## Backups

Before any change is applied — single-repo or batch — a timestamped JSON
snapshot of the properties' prior values is written to
`$HOME/.gh-custom-properties/backups/` (override with `--backup-dir`, the
`GH_CUSTOM_PROPERTIES_BACKUP_DIR` environment variable, or `backup_dir=` in
the config file). If the pre-change fetch fails, no backup is written and
the change is not applied.

- Single-repo apply: `owner-repo-20260901-153000.json`
- Batch apply (one file per bulk-edit run, however many repos it touches):
  `batch-20260901-153000.json`

```json
{
  "timestamp": "2026-09-01T15:30:00Z",
  "mode": "single",
  "repos": [
    {"owner": "octocat", "repo": "hello-world",
     "properties_before": [{"name": "team", "value": "platform"}]}
  ]
}
```

## Elsevier standard properties

Dropdown-style properties (`MigrationReady`, `ReviewComplete`, `SystemType`,
`TargetOrg`, `TechOrg`, `TechOrgGroup`, ...) always get their allowed values
live from the org's schema (`GET /orgs/{org}/properties/schema`) — nothing
is hardcoded, so the dropdown list can't go stale.

GitHub's schema only knows a property is `string`, though, not that it
should look like an email address or a date. For Elsevier's standard
string-typed properties, gh-custom-properties adds that format validation
client-side (`internal/knownprops`), rejecting the value (with an inline
error) until it's fixed:

| Property | Required format |
|---|---|
| `CostCode` | alphanumeric |
| `SystemID` | alphanumeric |
| `SystemName` | alphanumeric |
| `owner` | email address |
| `MigrationReadyDate` | `YYYY-MM-DD` |

## Configuration

Settings load from `$HOME/.gh-custom-properties/config` (a plain
`key=value` file), then environment variables, then command-line flags,
each overriding the last:

```
backup_dir=/some/path
token=ghp_...
```

The color theme is config-file-only (no env/flag override), and falls back
to the built-in defaults for anything unset:

```
color_title=212      # screen titles
color_header=39      # the persistent owner/repo (or org — N repo(s)) banner
color_cursor=212      # the selected row's cursor
color_selected=212    # the selected row's text
color_dim=240         # secondary text (scroll indicators, "(unset)", filter box)
color_error=203       # error messages
color_success=42      # success messages
color_warn=214        # warnings (e.g. org schema unavailable)
color_help=240        # footer key hints
```

Values are lipgloss color specs: an ANSI 256 color number (`"212"`) or a
hex code (`"#ff69b4"`).

## Development

See [Engineering.md](Engineering.md) for the package layout and design
notes, and [AGENTS.md](AGENTS.md) for notes aimed at an AI coding agent
picking this repo back up.

```sh
go build ./...
go vet ./...
go test ./...
gofmt -l .          # should print nothing
go generate ./...   # regenerate docs/*.1 and docs/*.html after editing internal/clidoc
```
