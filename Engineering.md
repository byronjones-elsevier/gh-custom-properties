# Engineering notes

## Package layout

```
main.go                  flag parsing, wiring, tea.NewProgram
internal/
  clidoc/                 single source of truth for the CLI spec; renders
                          text (-h), man page, and HTML help from one Spec
  config/                 $HOME/.gh-custom-properties/config loader; file <
                          env < flag precedence
  ghclient/               GitHub REST client for custom properties (no SDK
                          dependency — 3 endpoints, plain net/http)
  backup/                 timestamped JSON snapshot writer
  repolist/               parses --file (one owner/repo or URL per line)
  knownprops/             Elsevier-specific format validators (email/date/
                          alphanumeric) for string-typed properties, keyed
                          by property name
  termkeys/               optional Kitty-keyboard-protocol detection, so
                          Alt+letter commands work where the terminal
                          supports it (see "Alt-key commands" below)
  tui/                    Bubble Tea models: single-repo flow, batch flow,
                          shared type-aware value editor, the top bar
                          (bubbletea-menubar) and boxed footer panel
                          (bubbles/help) — see their sections below
docs/
  gendocs/                generator invoked via `go generate`; writes
                          docs/gh-custom-properties.1 and .html
Makefile                  deps/build/install/test/lint/docs-gen/clean; `make
                          help` lists every target; `make all` mirrors CI
.github/workflows/ci.yml   lint (golangci-lint), test (ubuntu+macOS), build,
                          and a check that generated docs are up to date —
                          on every push/PR to main
```

## Build tooling

The Makefile is the single source of truth for the actual commands (`go
build`, `go test`, etc.); CI's `test`/`build`/`docs` jobs just run `make
test`/`make build`/`make docs/gen` rather than duplicating them, so local
and CI behavior can't drift apart. The `lint` job is the one exception —
it uses `golangci/golangci-lint-action` directly instead of `make lint`,
for that action's GitHub-native annotations and its own result caching.
`golangci-lint` needs v2.x (the project uses the v2 config schema
implicitly via its defaults — there's no `.golangci.yml` yet, just the
tool's default linter set); an older v1-targeting binary, or one built
against an older Go toolchain than what `go.mod` specifies, may not run at
all against this codebase.

Every third-party action in the workflow is pinned to a full commit SHA
(with the version in a trailing comment) rather than a floating tag, to
avoid a compromised tag silently changing what CI runs.

## GitHub API surface

Custom properties are a small, three-endpoint surface
(`internal/ghclient/properties.go`):

- `GET /repos/{owner}/{repo}/properties/values` — current values
- `PATCH /repos/{owner}/{repo}/properties/values` — upsert; only the
  properties included in the request body are touched, others are left
  alone. A property with `value: null` clears it — this is how "delete" is
  implemented, since repo properties are values against an org-defined
  schema, not independently deletable rows.
- `GET /orgs/{org}/properties/schema` — the org's valid property
  names/types/allowed values. `Client.GetOrgSchema` returns
  `ErrSchemaUnavailable` on 403/404 so callers fall back to freeform string
  editing rather than treating a permissions gap as fatal.

Because PATCH only touches properties present in the request, both the
single-repo apply and the batch bulk-apply send only the properties that
actually changed — single-repo mode sends its whole in-memory working set
(harmless: re-setting an unchanged value to itself is a no-op), batch mode
sends exactly the one property being bulk-set or -deleted.

### Why the standard property catalog isn't hardcoded

Elsevier's `single_select` custom properties (`MigrationReady`,
`SystemType`, `TargetOrg`, `TechOrg`, `TechOrgGroup`, ...) are already
available, live, from `GetOrgSchema` — deliberately not duplicated as a
static fallback list. A live check against `elsevierPTG`'s real schema
during development turned up several differences from a hand-typed
reference list of the same properties (e.g. "Researcher Products" vs.
"Research Products", "TIO Data Engineering" vs. "TIO Date Engineering",
"Incident Response" vs. "Incident Management" in `TechOrgGroup`'s allowed
values) — exactly the staleness risk a hardcoded copy would reintroduce.
`internal/knownprops` only covers what the live schema genuinely can't
express: format validation for the `string`-typed properties (email, date,
alphanumeric). Adding a new Elsevier standard `single_select` property
needs no code change here — it just needs to exist in the org's schema.

GitHub doesn't guarantee an order for either the property list or a
`single_select`/`multi_select` property's `allowed_values` — `internal/tui/schema.go`'s
`sortSchema` sorts both alphabetically (by property name, and each
property's allowed values) right after a fetch, so the name picker and
value dropdowns are always predictable regardless of API response order.

`ghclient.PropertiesAPI` is the interface the TUI depends on instead of the
concrete `*Client`; tests use a hand-rolled fake (`internal/tui/fake_api_test.go`)
rather than hitting the network. `Client` itself is tested against
`httptest.Server` in `internal/ghclient/properties_test.go`.

## Value representation

GitHub represents every custom-property value as a string or an array of
strings over the wire — including `true_false`, which is the strings
`"true"`/`"false"`, not a JSON boolean. `ghclient.PropertyValue.Value` is
`any` holding one of `string`, `[]string`, or `nil` (unset/cleared);
`normalizeValue` in `internal/ghclient/types.go` converts the `[]any` that
`encoding/json` produces for array values into `[]string` right after
unmarshaling, so nothing downstream has to deal with `[]any`.

## TUI structure

`internal/tui/app.go`'s `App` is a thin router: it holds one `innerModel`
(either `*singleRepoModel` or `*batchModel`, both satisfying `tea.Model`)
and delegates `Init`/`Update`/`View`, layering global concerns on top —
see "App-level overlays" below.

Both flows share `valueEditor` (`internal/tui/valueeditor.go`), a type-aware
widget with two steps: pick a name (from the org schema, or freeform text
if the schema is unavailable), then edit the value with a widget matching
the property's type (text input, single-select list, multi-select
checklist). `newDeleteEditor` reuses the same name-picking step but
short-circuits after it, since deleting only needs a name.

### App-level overlays: quit-confirm, help, minimum size

`App` tracks `overlay overlayKind` (`none`/`quitConfirm`/`help`) and its own
`width, height`. `ctrl+c`/`ctrl+q` are intercepted directly in `App.Update`
before ever reaching the inner model; bare `q` can't be intercepted the
same way (it must stay scoped to screens with no active text input, or
typing a "q" into a repo name/property value would quit instead), so the
inner models emit a `quitRequestedMsg{}` via a returned `tea.Cmd` instead —
cmd-produced messages route back through `App.Update` before the inner
model ever sees them again, the same mechanism `loadedMsg`/`appliedMsg`
already rely on, so `App` can gate it behind the same confirmation. While
`overlay != none`, `App.Update` swallows `KeyMsg`s itself (`y`/`enter`
confirms quit, anything else cancels) but still forwards non-`KeyMsg`
messages to the inner model, so a fetch that's still in flight isn't
frozen by the overlay. `F1` sets `overlay = help` from any screen,
rendering a static keybinding reference (`internal/tui/help.go`) instead of
the CLI's `-h` text (which documents flags, not in-TUI usage). Below
`minWidth`/`minHeight`, `App.View` shows a resize warning instead of either
model's view.

The single-repo result screen's "press any key to continue" stays an
immediate, unconfirmed quit — that's a deliberate dismissal after work is
already saved (or a failure already reported), not an accidental
interrupt, so gating it behind the same confirm would just add friction.

### Config-driven color palette

`internal/tui/styles.go`'s style `var`s are `(re)`built by `ApplyPalette`
from a `Palette` (plain `tui`-local type, not `config.Palette`, so this
package doesn't need to know about config loading) — called once in
`main.go` right after `config.Load`. `internal/config/palette.go` reads
optional `color_*` keys from the config file on top of
`defaultPalette()`, which matches the values this app has always used.

### Window size and scrolling

Both models track the terminal size from `tea.WindowSizeMsg` (`width`,
`height` fields) and use it to bound how many rows a list shows at once —
`internal/tui/scroll.go`'s `availableRows(height)` estimates how much
vertical space is left after the header/title/help chrome, and
`visibleWindow(n, cursor, maxVisible)` returns the `[start, end)` slice
that keeps the cursor in view within that budget, with a
"N more above/below" indicator when the list is clipped. This applies to
the single-repo property list, the batch repo table, and every
`optionPicker` (name pickers, single/multi-select value pickers, the batch
target picker) — anywhere a list's length isn't bounded by the data model.
`maxVisible <= 0` means "show everything unclipped," which is also the
state before the first `WindowSizeMsg` arrives (e.g. in tests that
construct a model directly without sending one).

### Paging

`internal/tui/scroll.go`'s `pageSize(maxVisible)` returns a full visible
window (falling back to `defaultPageSize` when unbounded), and
`isPageUpKey`/`isPageDownKey` recognize PgUp/PgDn, F7/F8, and
Shift-Up/Shift-Down. `optionPicker.pageUp`/`pageDown` and the equivalent
inline cursor math in the property list and repo table all move by that
amount, clamping at both ends.

### "?" filtering

`optionPicker.cursor` indexes into a computed `visibleIndices()` — every
option index when there's no filter, otherwise only those matching it via
the shared `filterIndices` helper — rather than into `options` directly;
`checked` stays keyed by the raw option index so it survives a filter
change. `singleRepoModel`/`batchModel` carry their own `filter`/`filtering`
fields and the same Esc/`?`/rune-forwarding logic for the property list and
repo table, since their underlying data isn't `[]string`. Enter while
actively typing (`filtering == true`) locks the filter in rather than
confirming the outer selection or toggling a checkbox — needed since Space
would otherwise be ambiguous between "type a space into the filter" and
"toggle the current multi-select item." Esc clears an active filter first;
only once nothing is filtered does it fall through to its normal
cancel/back meaning.

### Flash-on-keypress

`singleRepoModel`/`batchModel` defer a hub screen's command key (`a`/`e`/
`d`/`s`/`q` on the property list, `b`/`q` on the repo table) via
`pendingFlash{label, action}` and a `flashTick()` (100ms), instead of
running the action immediately, then `flashElapsedMsg` fires the stored
`action`. Scoped to these two hub screens only; sub-screens (editors,
confirm dialogs) are lower-frequency interactions where the added delay
would cost more than it's worth. There's no visual highlight on the
footer for the pending key anymore — that was dropped once the footer
moved to `bubbles/help`-based rendering (see "Boxed, width-aware footer
panel" below), which has no hook for overriding a single item's style;
`pendingFlash.label` is kept (read by tests) even though nothing renders
it now.

### Top bar

`internal/tui/styles.go`'s `renderTopBar` renders the persistent top bar —
relevant commands on the left, the owner/repo (or batch org/count) identity
right-aligned — using `github.com/jejacks0n/bubbletea-menubar`'s `Model`
purely for its label-plus-underlined-hotkey rendering (`Styles.Hotkey`
underlines whichever letter in a label matches that item's `Hotkey`, e.g.
"Add" with `Hotkey: "a"` underlines the A). It's always constructed with
`Active` left `false`: our own key handlers already dispatch every one of
these commands (and have to — `?` means something different depending on
whether the property list is currently filtering, for instance), so wiring
up the package's own navigation/activation would create a second,
conflicting dispatch path for the exact same keys. This is display-only.
`topBarItems()` on each model returns `nil` outside the hub screen, so
secondary screens (input, an open editor, ...) show just the identity text
with no commands that aren't actually available.

`ViewBarWithRightSide` only uses its `width` argument to size the spacer
before the right-aligned text — it doesn't truncate or wrap the items
themselves. Our full item set needs roughly 75-80 columns, which is wider
than `App`'s own minimum width (60), so `renderTopBar` measures its own
output and falls back to dropping the items (keeping just the identity
text) if it would overflow — the footer panel below is the guaranteed-fit
place to find the commands regardless.

### Boxed, width-aware footer panel

The property list and repo table split their render into content and a
separate `footer` return value (`viewListParts`/`viewTableParts`, mirroring
the `header()`-style split used for the top bar); `pinFooter(content,
footer, height)` (`internal/tui/scroll.go`) pads between them with blank
lines up to the known terminal height, so the footer lands on the last row
instead of floating wherever the content ends. Scoped to these two hub
screens — sub-screens are short enough that pinning adds little.

The footer itself (`renderFooterPanel`, `internal/tui/styles.go`) is a
`bubbles/help`-styled, bordered panel (`footerPanelStyle`): it tries
`help.Model.ShortHelpView` (one line) first, and if that doesn't fit
`footerContentWidth`, falls back to `wrapShortHelp`, which packs "key desc"
hints onto as many lines as needed rather than eliding any of them — using
`help.Model.Styles.ShortKey`/`ShortDesc` for the same key/description
styling `ShortHelpView` itself would use, so the two look identical. This
doesn't use `help.Model.FullHelpView` (its natural fixed-height-column
mode): that view's own width-fitting has a gap — if even a single column
doesn't fit and there's no room left for an ellipsis marker, it falls
through to rendering that oversized column anyway (see its
`shouldAddItem`) — so it doesn't reliably guarantee everything fits, which
is the whole point here.

Both the top bar and the footer panel's width math have to account for
`boxStyle`'s own horizontal padding (`outerFrameWidth`, `internal/tui/styles.go`),
since both are rendered inside that same wrap around the whole screen —
sizing either to the raw terminal width, without subtracting that, overflows
the real terminal by that much. The property list/repo table's own row
text (`%-30s`/`%-40s`-formatted) needed the same fix: `rowContentWidth` plus
`truncateToWidth` (ANSI-aware, via `charmbracelet/x/ansi`'s `Truncate`) cap
each row to what's actually available, which the fixed-width formatting
alone never did.

### Alt-key commands

TUI.md calls for every custom command to also be reachable via
`Alt-<letter>`. Cmd is unreachable from a terminal program (consumed by the
terminal emulator itself), and Option/Alt needs either the Kitty keyboard
protocol or a manually-enabled "Use Option as Meta Key" setting — neither
of which macOS's default Terminal.app has — so `internal/termkeys` probes
for Kitty protocol support once at startup (`main.go`, strictly *before*
`tea.Program.Run()`, since the probe does its own raw-mode `stdin` read
that would race with Bubble Tea's input loop if run concurrently) and asks
the terminal to enable disambiguated reporting if found. Every custom
command's `key.Binding` matches `alt+<letter>` *unconditionally* —
registering it doesn't depend on detection succeeding, since an
unsupported combination simply never arrives as a `KeyMsg`. This means Alt
shortcuts work automatically on Kitty-protocol terminals and on any
terminal where the user has already enabled Option-as-Meta, while the bare
letter — the only thing that reliably works everywhere, including default
Terminal.app — never regresses.

Batch mode fetches and applies with a bounded worker pool (8 by default,
`batchFetchWorkers`/`batchApplyWorkers` in `internal/tui/batch.go`) rather
than one goroutine per repo, so a large `--file` doesn't open hundreds of
simultaneous connections. Fetch results stream back as one `batchChunkMsg`
per worker chunk (driving the loading screen's progress bar); apply results
come back as a single `batchAppliedMsg` once the whole run — backup
included — finishes, since the backup must be written for the complete set
of targets before any patch goes out.

State-transition tests for both flows live in `internal/tui/singlerepo_test.go`
and `internal/tui/batch_test.go`, calling `Update` directly with synthetic
`tea.KeyMsg`/message values against the `fakeAPI`. Since hub-screen command
keys now defer via `pendingFlash` (see "Flash-on-keypress"), tests use a
`settle`/`settleBatch` helper that feeds a `flashElapsedMsg` directly
rather than waiting out the real 100ms `flashDuration`.

## Known scope limits

- Batch mode's bulk-edit property picker is built from the first
  successfully-loaded repo's org schema. A repo list spanning multiple
  orgs will still work, but a bulk edit using a property name that isn't
  valid for a given repo's org will show as a per-repo API error in the
  apply results rather than being pre-validated.
- There's no dedicated "retry failed" action after a batch apply; rerun the
  bulk edit and use the target picker (`a`/`n`/`space`) to select just the
  repos that failed.
- Single-repo `F5` discards any staged-but-unapplied add/edit/delete
  without a confirmation prompt (browser-refresh semantics). Batch mode has
  no staging concept — bulk edits apply immediately — so its `F5` is
  loss-free.
- `internal/termkeys`'s Kitty-protocol probe spawns a goroutine to read
  `stdin` with a 100ms timeout; if the terminal never replies, that
  goroutine blocks on the read for the life of the process. Harmless (one
  goroutine, cleaned up on exit) but worth knowing if it ever shows up in a
  goroutine dump.
