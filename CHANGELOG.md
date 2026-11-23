# Changelog

History of all work on `gh-custom-properties`, newest first, one entry per
branch. Entries hold no pull request numbers, merge times, merged status, or
commit ids: run `make changelog/status` to print those from git, in UTC.

People:

- Byron Jones (`ByronJones-Elsevier`; owner, author, reviewer, merger). Git
  aliases: "Byron Scott Jones @ Elsevier" (commits made through the GitHub web
  UI, including merge commits).

---
### docs/agent-standards: adopt the house agent standards

- Branch: `docs/agent-standards`, base `main`
- Worked on by: Byron Jones (owner, requester); Claude Sonnet 5.5 (Claude Code, author)
- Written: 2026-11-23T14:00:00Z
- What it did:
  - Added `ENGINEERING.md` (renamed from `Engineering.md`) as the primary
    specification: requirement IDs, interfaces, data, behavior, security,
    dependencies, verification plan, decisions.
  - Added `HANDOFF.md` and this `CHANGELOG.md` (history backfilled).
  - Added `agents/` topic-file snapshots and a Topic Routing table in `AGENTS.md`.
  - Removed the `ctrl+q` quit binding (reserved for flow control).
  - Added `make vuln` (govulncheck), a CI `vuln` job, and `make changelog/status`.
  - Added tests that keep the spec and the living documents honest.
- Spec and docs: all of `ENGINEERING.md`; REQ-TUI-2, REQ-BUILD-1, D-1 to D-6
- Verification: `go vet`, `go test ./...`, and `golangci-lint run` pass. Not
  exercised: `make vuln` (the sandbox blocked the module cache), `make all` as
  a whole, markdownlint.
- Notes: the spec describes the behavior of the three feature branches below as
  merged; see HANDOFF "Resume here".
  Update 2026-11-23T15:00:00Z: the owner ran `make vuln`; govulncheck v1.1.4
  works and reported one standard-library finding fixed in a newer Go patch
  release (tracked as H-5). It found nothing in module dependencies.

---
### fix/single-delete-clears: single-repo delete now clears on GitHub

- Branch: `fix/single-delete-clears`, base `main`
- Worked on by: Byron Jones (owner); Claude Sonnet 5.5 (Claude Code, author)
- Written: 2026-11-23T13:45:00Z
- What it did: deleting a property in single-repo mode only removed it from the
  in-memory list, so apply PATCHed the remaining properties and the deleted one
  stayed set. Apply now adds an explicit `null` for each loaded property that
  is no longer listed.
- Spec and docs: ENGINEERING.md REQ-EDIT-1, section 5
- Verification: new `internal/tui/delete_apply_test.go`; full test suite, vet,
  and lint pass.
- Notes: found while implementing blank-clears-value; the bug predates it.

---
### feat/blank-clears-value: blank CostCode/SystemID/SystemName unsets the property

- Branch: `feat/blank-clears-value`, base `main`
- Worked on by: Byron Jones (owner, requester); Claude Sonnet 5.5 (Claude Code, author)
- Written: 2026-11-23T13:35:00Z
- What it did: the three properties accept a blank value, which the value
  editor turns into `nil`, so apply unsets the property. The editor shows a
  hint for these properties.
- Spec and docs: REQ-VAL-1, D-5; README validation table
- Verification: `internal/knownprops` and `internal/tui/clearvalue_test.go`;
  full suite, vet, and lint pass. Not exercised against live GitHub.

---
### feat/open-in-browser: open the repo's Custom properties page (o key)

- Branch: `feat/open-in-browser`, base `main`
- Worked on by: Byron Jones (owner, requester); Claude Sonnet 5.5 (Claude Code, author)
- Written: 2026-11-23T13:25:00Z
- What it did: new `internal/browser` package and an `o` / `alt+o` command on
  the single-repo list and the batch table that opens
  `https://github.com/<owner>/<repo>/settings/custom-properties`.
- Spec and docs: REQ-BROWSER-1; README keybinding tables; F1 help
- Verification: `internal/browser` and `internal/tui/browser_test.go` (browser
  launcher stubbed); full suite, vet, and lint pass. The real launcher was not
  exercised.

---
### chore/bump-v1.1: bump version to 1.1 (backfilled history)

- Branch: `chore/bump-v1.1`, base `main`
- Worked on by: Byron Jones (author, merger)
- Written: 2026-11-23T14:00:00Z (backfilled from git history)
- What it did: version bump to 1.1.

---
### feat/headless-list-set: non-interactive --list and --set (backfilled history)

- Branch: `feat/headless-list-set`, base `main`
- Worked on by: Byron Jones (author, merger)
- Written: 2026-11-23T14:00:00Z (backfilled from git history)
- What it did: added `--list` and `--set` modes (`internal/headless`).

---
### main (direct): security policy and gitignore (backfilled history)

- Branch: `main`, direct commits
- Worked on by: Byron Jones (author)
- Written: 2026-11-23T14:00:00Z (backfilled from git history)
- What it did: revised `SECURITY.md` contact information; ignored local repo
  lists and CSV exports.

---
### release/v1.0: v1.0 release packaging (backfilled history)

- Branch: `release/v1.0`, base `main`
- Worked on by: Byron Jones (author, merger)
- Written: 2026-11-23T14:00:00Z (backfilled from git history)
- What it did: prepared v1.0 packages, normalized the release workflow, and
  synchronized the fake API's calls in tests.

---
### feat/repo-list-columns: batch repositories in columns (backfilled history)

- Branch: `feat/repo-list-columns`, base `main`
- Worked on by: Byron Jones (author, merger)
- Written: 2026-11-23T14:00:00Z (backfilled from git history)
- What it did: rendered the batch table in columns; allowed unloading repos.

---
### feat/Add-Bulk-Export: export batch repositories to CSV (backfilled history)

- Branch: `feat/Add-Bulk-Export`, base `main`
- Worked on by: Byron Jones (author, merger)
- Written: 2026-11-23T14:00:00Z (backfilled from git history)
- What it did: `x` exports every loaded repo and its values to CSV.

---
### feat/Add-Individual-Properties-Screen: per-repo editing in batch mode (backfilled history)

- Branch: `feat/Add-Individual-Properties-Screen`, base `main`
- Worked on by: Byron Jones (author, merger)
- Written: 2026-11-23T14:00:00Z (backfilled from git history)
- What it did: `enter`/`e` on a batch row opens that repo's property screen.

---
### feat/makefile-and-ci: Makefile and GitHub Actions CI (backfilled history)

- Branch: `feat/makefile-and-ci`, base `main`
- Worked on by: Byron Jones (author, merger)
- Written: 2026-11-23T14:00:00Z (backfilled from git history)
- What it did: added the Makefile and CI workflow; the README demo media were
  added directly on `main` around the same time.

---
### feat/initial-build: initial build (backfilled history)

- Branch: `feat/initial-build`, base `main`
- Worked on by: Byron Jones (author, merger)
- Written: 2026-11-23T14:00:00Z (backfilled from git history)
- What it did: scaffolded the module, the GitHub client, backups, repo-list
  parsing, the Bubble Tea single-repo and batch flows, format validation for
  the Elsevier string properties, live schema sorting, scrolling and paging,
  filtering, the color palette config, Alt-key support, the top menu bar, and
  the boxed footer panel.
