# AGENTS.md

Notes for an AI coding agent picking this repository back up. See
[Engineering.md](Engineering.md) for design details and package layout,
[README.md](README.md) for user-facing usage.

## Quick orientation

- Module: `github.com/ByronJones-Elsevier/gh-custom-properties`, Go 1.23+.
- No third-party GitHub SDK — `internal/ghclient` talks to the REST API
  directly via `net/http`. External dependencies are UI
  (`charmbracelet/bubbletea`, `bubbles`, `lipgloss`, `jejacks0n/bubbletea-menubar`
  for the top bar) plus `golang.org/x/term` (used only by
  `internal/termkeys`'s Alt-key detection probe).
- `internal/clidoc` is the single source of truth for the CLI's flags/help
  text. If you change a flag in `main.go`, update
  `internal/clidoc/spec.go` first and run `go generate ./...` to
  regenerate `docs/gh-custom-properties.1` and `docs/gh-custom-properties.html` —
  don't hand-edit those generated files.
- This project follows the house TUI standard at `~/.claude/agents/TUI.md`
  (if you have access to it), with one deliberate, documented deviation:
  custom commands stay bare letters rather than being *exclusively*
  Alt-A..Alt-Z — see Engineering.md's "Alt-key commands" section for why
  (Cmd is unreachable from a terminal program; Option/Alt needs a setting
  most terminals don't have by default) and how Alt shortcuts are still
  wired up additively via `internal/termkeys`.

## Before committing

```sh
make all   # fmt/check, vet, lint, test, build — same checks CI runs
```
`golangci-lint` needs v2.x, built against a Go toolchain at least as new as
whatever's installed locally — an older binary can panic outright rather
than report findings; `brew upgrade golangci-lint` (or equivalent) fixes
that. If you touched `internal/clidoc`, also run `make docs/gen` and
commit the regenerated `docs/*.1`/`docs/*.html` alongside — CI's `docs`
job fails the build if they're out of sync.

## Local environment note

This machine sits behind a Zscaler TLS-intercepting proxy. `go get`/`go mod
tidy` need `SSL_CERT_FILE` pointed at the actual (tilde-expanded) cert path,
since the shell-set env var contains a literal, unexpanded `~`:

```sh
SSL_CERT_FILE="$HOME/.ssh/zscaler/zscaler.pem" go get ...
```

## Testing conventions in this repo

- Table-driven tests, `t.Run` subtests, `t.TempDir()`/`t.Setenv()` over
  manual cleanup.
- No network calls in tests: `internal/ghclient` tests use
  `httptest.Server`; `internal/tui` tests use the hand-rolled
  `fakeAPI` in `internal/tui/fake_api_test.go` (implements
  `ghclient.PropertiesAPI`).
- TUI tests drive `Update` directly with synthetic `tea.KeyMsg`/message
  values rather than running the full Bubble Tea program loop — see
  `internal/tui/singlerepo_test.go` and `batch_test.go` for the pattern.
- Hub-screen command keys (`a`/`e`/`d`/`s`/`q`/`b`) defer via a 100ms flash
  before acting (see Engineering.md's "Flash-on-keypress") — use the
  `settle`/`settleBatch` test helpers (feed a `flashElapsedMsg` directly)
  right after pressing one, instead of waiting out the real delay.

## Git workflow for this repo

Private repo at `github.com/byronjones-elsevier/gh-custom-properties`
(matches the module path case-insensitively — GitHub usernames aren't
case-sensitive). `main` is the default branch; work happens on feature
branches with a PR back into `main` — see `~/.claude/agents/CODING.md` if
you have access to it for the broader git-hygiene convention. CI
(`.github/workflows/ci.yml`) runs on every push and PR to `main`.
