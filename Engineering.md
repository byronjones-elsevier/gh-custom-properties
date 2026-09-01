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
  tui/                    Bubble Tea models: single-repo flow, batch flow,
                          shared type-aware value editor
docs/
  gendocs/                generator invoked via `go generate`; writes
                          docs/gh-custom-properties.1 and .html
```

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
(either `*singleRepoModel` or `*batchModel`, both satisfying `tea.Model`
plus a `Quitting() bool` escape hatch) and delegates `Init`/`Update`/`View`.

Both flows share `valueEditor` (`internal/tui/valueeditor.go`), a type-aware
widget with two steps: pick a name (from the org schema, or freeform text
if the schema is unavailable), then edit the value with a widget matching
the property's type (text input, single-select list, multi-select
checklist). `newDeleteEditor` reuses the same name-picking step but
short-circuits after it, since deleting only needs a name.

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
`tea.KeyMsg`/message values against the `fakeAPI`.

## Known scope limits

- Batch mode's bulk-edit property picker is built from the first
  successfully-loaded repo's org schema. A repo list spanning multiple
  orgs will still work, but a bulk edit using a property name that isn't
  valid for a given repo's org will show as a per-repo API error in the
  apply results rather than being pre-validated.
- There's no dedicated "retry failed" action after a batch apply; rerun the
  bulk edit and use the target picker (`a`/`n`/`space`) to select just the
  repos that failed.
