# Handoff

Open work only. Completed work belongs in [CHANGELOG.md](CHANGELOG.md); the
specification is [ENGINEERING.md](ENGINEERING.md).

## 1. Status

- Last updated: 2026-11-23T17:00:00Z
- In flight: none.
- Verified 2026-11-23: CI is green on `main` (lint, test on ubuntu and macOS,
  govulncheck, build, generated docs) and `make vuln` passes locally. Not
  verified: markdownlint.

## 2. Resume here

Pick an item from section 4. Before starting, run `make all` and `make vuln`
(module commands behind the proxy need the certificate; see section 7).

## 3. In progress

None beyond the branches listed in section 1.

## 4. Outstanding work

| ID | Item | Why | Acceptance | Owner | Blocked by |
| --- | --- | --- | --- | --- | --- |
| H-3 | Verify terminal restore on SIGINT/SIGTERM | Required by `agents/tui.md`; the project installs no handler | `kill -TERM` the running TUI and confirm the terminal is usable | Byron Jones | none |
| H-6 | Test the real browser launcher on macOS, Linux, and Windows | Only the stubbed path is tested | `o` opens the page on each platform | Byron Jones | none |

## 5. Decisions needed

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

- Local branches already merged into `main` can be deleted:
  `docs/agent-standards`, `feat/open-in-browser`, `feat/blank-clears-value`,
  `fix/single-delete-clears`. `feat/Unload-Repo` predates this work and was not
  checked.
- Two untracked local files in the working tree (a `RemediationStatus` HTML
  report and a `remediate` shell script, both dated 2026-11-18) predate this
  work and were left alone; decide whether to commit, ignore, or delete them.
