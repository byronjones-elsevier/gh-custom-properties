# AGENTS.md

Notes for an AI coding agent picking this repository back up. See
[Engineering.md](Engineering.md) for design details and package layout,
[README.md](README.md) for user-facing usage.

## Quick orientation

- Module: `github.com/ByronJones-Elsevier/gh-custom-properties`, Go 1.23+.
- No third-party GitHub SDK — `internal/ghclient` talks to the REST API
  directly via `net/http`. Only UI dependencies
  (`charmbracelet/bubbletea`, `bubbles`, `lipgloss`) are external.
- `internal/clidoc` is the single source of truth for the CLI's flags/help
  text. If you change a flag in `main.go`, update
  `internal/clidoc/spec.go` first and run `go generate ./...` to
  regenerate `docs/gh-custom-properties.1` and `docs/gh-custom-properties.html` —
  don't hand-edit those generated files.

## Before committing

```sh
go build ./... && go vet ./... && gofmt -l . && go test ./...
```
`gofmt -l .` must print nothing. If you touched `internal/clidoc`, also run
`go generate ./...` and commit the regenerated `docs/` output alongside.

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

## Git workflow for this repo

Single `main` branch, no PRs (no remote configured yet). Commit
incrementally as logical pieces land, per the user's global git-hygiene
preference — see `~/.claude/agents/CODING.md` if you have access to it.
