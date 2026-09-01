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
| `a` | add a property (from the org schema, or freeform if unavailable) |
| `enter`/`e` | edit the selected property's value |
| `d` | delete the selected property (confirm with `y`) |
| `s` | apply all changes (writes a backup first, then patches GitHub) |
| `q` | quit |

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

## Configuration

Settings load from `$HOME/.gh-custom-properties/config` (a plain
`key=value` file), then environment variables, then command-line flags,
each overriding the last:

```
backup_dir=/some/path
token=ghp_...
```

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
