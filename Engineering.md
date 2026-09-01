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
