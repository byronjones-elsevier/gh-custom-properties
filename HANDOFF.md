# Handoff

Open work only. Completed work belongs in [CHANGELOG.md](CHANGELOG.md); the
specification is [ENGINEERING.md](ENGINEERING.md).

## 1. Status

- Last updated: 2026-11-23T14:00:00Z
- In flight: `docs/agent-standards` (this branch), plus the local branches
  `feat/open-in-browser`, `feat/blank-clears-value`, and `fix/single-delete-clears`.
- Verified 2026-11-23: `go vet`, `go test ./...`, and `golangci-lint run` pass on
  each branch. `make vuln` passed locally on 2026-11-23 and `go mod tidy` was run. Not verified: `make all` end to end, markdownlint,
  CI.

## 2. Resume here

1. The three feature and fix branches must land on `main` first. `ENGINEERING.md`
   already describes their behavior, so this branch must not merge before them.
2. Then rebase `docs/agent-standards` onto `main`, resolving any
   `Engineering.md` to `ENGINEERING.md` rename conflict by keeping this branch's
   file.
3. Run `make all`, `make vuln`, and
   `npx --yes markdownlint-cli2 '**/*.md' '#.git/**'`.

## 3. In progress

None beyond the branches listed in section 1.

## 4. Outstanding work

| ID | Item | Why | Acceptance | Owner | Blocked by |
| --- | --- | --- | --- | --- | --- |
| H-1 | Add a `LICENSE` file | `agents/git.md` requires one (MIT by default) | `LICENSE` present; `README.md` states it | Byron Jones | Decision D-A |
| H-2 | Verify `NO_COLOR` is honored | `agents/tui.md` requires it; nothing in the code reads it, lipgloss may handle it | Run the TUI with `NO_COLOR=1` in a terminal and confirm no color; add a test if feasible (REQ-TUI-1) | Byron Jones | none |
| H-3 | Verify terminal restore on SIGINT/SIGTERM | Required by `agents/tui.md`; the project installs no handler | `kill -TERM` the running TUI and confirm the terminal is usable | Byron Jones | none |
| H-5 | Confirm the CI `vuln` job passes | Locally `make vuln` reports no vulnerabilities (2026-11-23, after upgrading Go). The CI job first failed on the `go.mod` Go version (1.25.0), then, on the newest stable Go, govulncheck v1.1.4 panicked (`unexpected expr: *ast.KeyValueExpr`); the job now uses the newest stable Go and `GOVULNCHECK_VERSION` v1.10.0, unverified locally | CI `vuln` job green on the updated workflow; if it still fails, read its log first | Byron Jones | none |
| H-6 | Test the real browser launcher on macOS, Linux, and Windows | Only the stubbed path is tested | `o` opens the page on each platform | Byron Jones | none |

## 5. Decisions needed

- **D-A: license holder.** Options: MIT with "Elsevier" as holder; MIT with a
  personal name; no license (private work repo). Recommendation: ask Elsevier
  legal or the repo owner; do not guess a holder.
- **D-B: GitHub Enterprise hosts.** The tool is hard-coded to `github.com` and
  `api.github.com`. If you use another host, the browser URL and API base need a
  setting. Recommendation: defer until needed.

## 6. Known issues, risks, and unverified areas

- A blank `SystemID`/`CostCode`/`SystemName` unsets the property; this was not
  exercised against live GitHub.
- If an org marks one of those properties as required, GitHub will reject the
  clear. The interactive path shows the API error; `--set` pre-checks it.
- The `agents/` files are snapshots of `~/.claude/agents` taken 2026-11-23 and
  were not modified. Refresh them when the source changes.
- Test counts are deliberately not recorded here.

## 7. Environment and access notes

- The machine sits behind a Zscaler TLS proxy; module commands need
  `SSL_CERT_FILE="$HOME/.ssh/zscaler/zscaler.pem"`.
- Inside the Claude Code sandbox, set `GOCACHE` to a writable directory; module
  downloads and `make vuln` do not work there.
- The repository owner pushes and merges; agents do not push.
- Credentials: tokens come from `GITHUB_TOKEN` or `gh auth token`; never write
  them to files.

## 8. Cleanup debt

- Two untracked local files in the working tree (a `RemediationStatus` HTML
  report and a `remediate` shell script, both dated 2026-11-18) predate this
  work and were left alone; decide whether to commit, ignore, or delete them.
