package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/backup"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/repolist"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	menubar "github.com/jejacks0n/bubbletea-menubar"
)

// batchTableHelpKeys adapts the repo table's key hints to bubbles/help's
// KeyMap interface for renderFooterPanel. Unlike singleListHelpKeys, these
// aren't real key.Bindings wired to key.Matches dispatch (handleTableKey
// switches on msg.String() directly), so keyBinding display-only bindings
// are used throughout.
type batchTableHelpKeys struct{}

func (batchTableHelpKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		keyBinding("↑/k", "up"), keyBinding("↓/j", "down"), keyBinding("enter/e", "repo details"), keyBinding("b", "bulk edit"), keyBinding("x", "export CSV"),
		keyBinding("?", "filter"), keyBinding("F5", "refresh"), keyBinding("F1", "help"),
		keyBinding("q", "quit"),
	}
}

func (batchTableHelpKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{keyBinding("↑/k", "up"), keyBinding("↓/j", "down")},
		{keyBinding("enter/e", "repo details")},
		{keyBinding("b", "bulk edit")},
		{keyBinding("x", "export CSV")},
		{keyBinding("?", "filter"), keyBinding("F5", "refresh"), keyBinding("F1", "help")},
		{keyBinding("q", "quit")},
	}
}

// batchFetchWorkers/batchApplyWorkers bound how many repos are fetched or
// patched concurrently, so a large repo list doesn't open hundreds of
// simultaneous connections to the GitHub API at once.
const (
	batchFetchWorkers = 8
	batchApplyWorkers = 8
)

type batchScreen int

const (
	bScreenLoading batchScreen = iota
	bScreenTable
	bScreenDetail
	bScreenConfirmUnload
	bScreenChooseAction
	bScreenChooseProperty
	bScreenChooseTargets
	bScreenApplying
	bScreenResult
	bScreenExporting
	bScreenExportResult
)

type bulkActionKind int

const (
	bulkSet bulkActionKind = iota
	bulkDelete
)

// repoRow is one repo's state within the batch session.
type repoRow struct {
	owner, repo string
	properties  []ghclient.PropertyValue // current in-memory values
	loaded      []ghclient.PropertyValue // last confirmed remote state, used as the next backup's "before"
	err         error                    // set if the initial fetch for this repo failed
}

func (r repoRow) label() string { return r.owner + "/" + r.repo }

// batchChunkMsg is emitted once per fetch-worker chunk as it finishes.
type batchChunkMsg struct {
	rows    []repoRow
	schemas map[string][]ghclient.PropertyDefinition
}

// batchApplyResult is the outcome of applying the bulk action to one repo.
type batchApplyResult struct {
	owner, repo string
	err         error
}

// batchAppliedMsg is emitted once the whole apply run (backup + all patches) finishes.
type batchAppliedMsg struct {
	backupPath string
	backupErr  error
	results    []batchApplyResult
}

type batchModel struct {
	api       ghclient.PropertiesAPI
	backupDir string

	screen batchScreen
	spin   spinner.Model
	prog   progress.Model

	entries      []repolist.Entry
	skipped      []repolist.Skipped
	chunksTotal  int
	chunksDone   int
	rows         []repoRow
	schemaByOrg  map[string][]ghclient.PropertyDefinition
	cursor       int
	filter       string // "?" starts composing this; narrows the table to a case-insensitive substring match on owner/repo
	filtering    bool
	pendingFlash *pendingFlash // set by a hub-screen command key; see keys.go
	help         help.Model    // renders the boxed footer panel; see renderFooterPanel

	action        bulkActionKind
	editor        *valueEditor
	pendingAction ghclient.PropertyValue
	targets       *optionPicker // multi-select over labels of fetchable rows
	targetRow     []int         // targets.options[i] -> index into m.rows

	applyResults []batchApplyResult
	backupPath   string
	backupErr    error
	exportPath   string
	exportErr    error

	width, height int // last known terminal size, from tea.WindowSizeMsg

	err error

	// detail reuses the single-repository property editor for the selected
	// row. It is kept here so the batch table can receive the saved values
	// when the detail screen is dismissed.
	detail          *singleRepoModel
	detailRow       int
	pendingUnloadAt int
}

func newBatchModel(api ghclient.PropertiesAPI, backupDir string, entries []repolist.Entry, skipped []repolist.Skipped) *batchModel {
	return &batchModel{
		api:       api,
		backupDir: backupDir,
		screen:    bScreenLoading,
		spin:      newSpinner(),
		prog:      progress.New(progress.WithDefaultGradient()),
		help:      newHelpModel(),
		entries:   entries,
		skipped:   skipped,
	}
}

func (m *batchModel) Init() tea.Cmd {
	return m.startFetch()
}

// startFetch (re)issues the chunked fetch across all entries, resetting any
// previously accumulated rows/schema/progress — used both by Init and by
// F5 refresh from the table screen.
func (m *batchModel) startFetch() tea.Cmd {
	m.rows = nil
	m.schemaByOrg = nil
	m.chunksDone = 0
	m.cursor = 0
	m.err = nil

	chunks := chunkEntries(m.entries, batchFetchWorkers)
	m.chunksTotal = len(chunks)
	if m.chunksTotal == 0 {
		m.screen = bScreenTable
		return nil
	}
	cmds := make([]tea.Cmd, 0, len(chunks)+1)
	cmds = append(cmds, m.spin.Tick)
	for _, c := range chunks {
		cmds = append(cmds, fetchChunkCmd(m.api, c))
	}
	return tea.Batch(cmds...)
}

func chunkEntries(entries []repolist.Entry, workers int) [][]repolist.Entry {
	if len(entries) == 0 {
		return nil
	}
	if workers > len(entries) {
		workers = len(entries)
	}
	chunks := make([][]repolist.Entry, workers)
	for i, e := range entries {
		chunks[i%workers] = append(chunks[i%workers], e)
	}
	return chunks
}

func fetchChunkCmd(api ghclient.PropertiesAPI, chunk []repolist.Entry) tea.Cmd {
	return func() tea.Msg {
		schemas := map[string][]ghclient.PropertyDefinition{}
		rows := make([]repoRow, 0, len(chunk))
		for _, e := range chunk {
			ctx := context.Background()
			props, err := api.GetRepoProperties(ctx, e.Owner, e.Repo)
			if err != nil {
				rows = append(rows, repoRow{owner: e.Owner, repo: e.Repo, err: err})
				continue
			}
			row := repoRow{
				owner:      e.Owner,
				repo:       e.Repo,
				properties: append([]ghclient.PropertyValue{}, props...),
				loaded:     append([]ghclient.PropertyValue{}, props...),
			}
			if _, ok := schemas[e.Owner]; !ok {
				if s, err := api.GetOrgSchema(ctx, e.Owner); err == nil {
					schemas[e.Owner] = s
				}
			}
			rows = append(rows, row)
		}
		return batchChunkMsg{rows: rows, schemas: schemas}
	}
}

func (m *batchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.detail != nil {
		// A detail result is dismissed back to the batch table, rather than
		// quitting the application as it does in standalone mode.
		if m.detail.screen == screenResult {
			if m.detail.err == nil {
				m.rows[m.detailRow].properties = propertiesWithValues(m.detail.properties)
				m.rows[m.detailRow].loaded = append([]ghclient.PropertyValue{}, m.rows[m.detailRow].properties...)
			}
			m.detail = nil
			m.screen = bScreenTable
			return m, nil
		}
		if km, ok := msg.(tea.KeyMsg); ok && m.detail.screen == screenList &&
			(km.String() == "q" || km.String() == "alt+q" || km.String() == "esc") {
			m.detail = nil
			m.screen = bScreenTable
			return m, nil
		}
		if wsm, ok := msg.(tea.WindowSizeMsg); ok {
			m.width, m.height = wsm.Width, wsm.Height
		}
		next, cmd := m.detail.Update(msg)
		m.detail = next.(*singleRepoModel)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.editor != nil {
			m.editor.setMaxVisible(availableRows(m.height))
		}
		if m.targets != nil {
			m.targets.maxVisible = availableRows(m.height)
		}
		return m, nil
	case batchChunkMsg:
		return m.handleChunk(msg)
	case batchAppliedMsg:
		m.applyResults = msg.results
		m.backupPath = msg.backupPath
		m.backupErr = msg.backupErr
		m.applyResultsToRows(msg.results)
		m.screen = bScreenResult
		return m, nil
	case batchExportedMsg:
		m.exportPath = msg.path
		m.exportErr = msg.err
		m.screen = bScreenExportResult
		return m, nil
	case flashElapsedMsg:
		if m.pendingFlash == nil {
			return m, nil
		}
		action := m.pendingFlash.action
		m.pendingFlash = nil
		return action()
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *batchModel) handleChunk(msg batchChunkMsg) (tea.Model, tea.Cmd) {
	m.rows = append(m.rows, msg.rows...)
	if m.schemaByOrg == nil {
		m.schemaByOrg = map[string][]ghclient.PropertyDefinition{}
	}
	for org, defs := range msg.schemas {
		if _, ok := m.schemaByOrg[org]; !ok {
			sortSchema(defs)
			m.schemaByOrg[org] = defs
		}
	}
	m.chunksDone++
	if m.chunksDone < m.chunksTotal {
		return m, nil
	}
	sort.Slice(m.rows, func(i, j int) bool { return m.rows[i].label() < m.rows[j].label() })
	m.screen = bScreenTable
	return m, nil
}

func (m *batchModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.screen {
	case bScreenLoading:
		if s := msg.String(); s == "q" || s == "alt+q" {
			return m, quitRequestedCmd
		}
	case bScreenTable:
		return m.handleTableKey(msg)
	case bScreenDetail:
		return m, nil
	case bScreenConfirmUnload:
		return m.handleConfirmUnloadKey(msg)
	case bScreenChooseAction:
		return m.handleChooseActionKey(msg)
	case bScreenChooseProperty:
		return m.handleChoosePropertyKey(msg)
	case bScreenChooseTargets:
		return m.handleChooseTargetsKey(msg)
	case bScreenResult:
		m.screen = bScreenTable
	case bScreenExportResult:
		m.screen = bScreenTable
	}
	return m, nil
}

// filteredRowIndices returns the indices into m.rows matching m.filter
// (against "owner/repo") — all of them when the filter is empty.
func (m *batchModel) filteredRowIndices() []int {
	labels := make([]string, len(m.rows))
	for i, r := range m.rows {
		labels[i] = r.label()
	}
	return filterIndices(labels, m.filter)
}

func (m *batchModel) clearFilter() {
	m.filter = ""
	m.filtering = false
	m.cursor = 0
}

func (m *batchModel) handleTableKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc && (m.filtering || m.filter != "") {
		m.clearFilter()
		return m, nil
	}

	if m.filtering {
		indices := m.filteredRowIndices()
		switch s := msg.String(); {
		case s == "enter":
			m.filtering = false
		case msg.Type == tea.KeyBackspace:
			if m.filter != "" {
				runes := []rune(m.filter)
				m.filter = string(runes[:len(runes)-1])
				m.cursor = 0
			}
		case s == "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case s == "down":
			if m.cursor < len(indices)-1 {
				m.cursor++
			}
		case isPageUpKey(s):
			m.cursor -= pageSize(availableRows(m.height))
			if m.cursor < 0 {
				m.cursor = 0
			}
		case isPageDownKey(s):
			m.cursor += pageSize(availableRows(m.height))
			if last := len(indices) - 1; m.cursor > last {
				m.cursor = last
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
		case msg.Type == tea.KeySpace:
			m.filter += " "
			m.cursor = 0
		case msg.Type == tea.KeyRunes:
			m.filter += string(msg.Runes)
			m.cursor = 0
		}
		return m, nil
	}

	indices := m.filteredRowIndices()
	switch s := msg.String(); {
	case s == "q" || s == "alt+q":
		return m.deferAction("q", func() (tea.Model, tea.Cmd) { return m, quitRequestedCmd })
	case msg.Type == tea.KeyF5:
		m.screen = bScreenLoading
		return m, m.startFetch()
	case s == "?":
		m.filtering = true
	case s == "up" || s == "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case s == "down" || s == "j":
		if m.cursor < len(indices)-1 {
			m.cursor++
		}
	case isPageUpKey(s):
		m.cursor -= pageSize(availableRows(m.height))
		if m.cursor < 0 {
			m.cursor = 0
		}
	case isPageDownKey(s):
		m.cursor += pageSize(availableRows(m.height))
		if last := len(indices) - 1; m.cursor > last {
			m.cursor = last
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	case s == "b" || s == "alt+b":
		m.err = nil
		return m.deferAction("b", func() (tea.Model, tea.Cmd) {
			m.screen = bScreenChooseAction
			return m, nil
		})
	case s == "enter" || s == "e" || s == "alt+e":
		if m.cursor < 0 || m.cursor >= len(indices) {
			return m, nil
		}
		rowIndex := indices[m.cursor]
		if m.rows[rowIndex].err != nil {
			m.err = fmt.Errorf("cannot edit %s: properties failed to load", m.rows[rowIndex].label())
			return m, nil
		}
		return m.deferAction("enter/e", func() (tea.Model, tea.Cmd) {
			m.openDetail(rowIndex)
			return m, nil
		})
	case s == "x" || s == "alt+x":
		m.err = nil
		return m.deferAction("x", func() (tea.Model, tea.Cmd) {
			m.screen = bScreenExporting
			return m, m.exportCmd()
		})
	case s == "u" || s == "alt+u":
		if m.cursor < 0 || m.cursor >= len(indices) {
			return m, nil
		}
		rowIndex := indices[m.cursor]
		return m.deferAction("u", func() (tea.Model, tea.Cmd) {
			m.pendingUnloadAt = rowIndex
			m.screen = bScreenConfirmUnload
			return m, nil
		})
	}
	return m, nil
}

func (m *batchModel) handleConfirmUnloadKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		m.unloadRow(m.pendingUnloadAt)
		m.pendingUnloadAt = -1
		m.screen = bScreenTable
	case "n", "esc":
		m.pendingUnloadAt = -1
		m.screen = bScreenTable
	}
	return m, nil
}

func (m *batchModel) unloadRow(rowIndex int) {
	if rowIndex < 0 || rowIndex >= len(m.rows) {
		return
	}
	row := m.rows[rowIndex]
	rows := make([]repoRow, 0, len(m.rows)-1)
	rows = append(rows, m.rows[:rowIndex]...)
	rows = append(rows, m.rows[rowIndex+1:]...)
	m.rows = rows
	entries := make([]repolist.Entry, 0, len(m.entries))
	for _, entry := range m.entries {
		if entry.Owner != row.owner || entry.Repo != row.repo {
			entries = append(entries, entry)
		}
	}
	m.entries = entries
	if indices := m.filteredRowIndices(); m.cursor >= len(indices) && m.cursor > 0 {
		m.cursor = len(indices) - 1
	}
}

type batchExportedMsg struct {
	path string
	err  error
}

func (m *batchModel) exportCmd() tea.Cmd {
	rows := append([]repoRow{}, m.rows...)
	schemas := m.schemaByOrg
	path := fmt.Sprintf("custom-properties-%s.csv", time.Now().Format("20060102-150405"))
	return func() tea.Msg {
		return batchExportedMsg{path: path, err: exportBatchCSV(path, rows, schemas)}
	}
}

// openDetail creates a single-repository editor from the row selected in the
// batch table. Schema-defined properties that are not currently set are
// included as nil values so the detail screen shows every field, not only
// fields returned by GitHub for that repository.
func (m *batchModel) openDetail(rowIndex int) {
	row := m.rows[rowIndex]
	detail := newSingleRepoModel(m.api, m.backupDir, row.owner, row.repo)
	detail.screen = screenList
	detail.width, detail.height = m.width, m.height
	detail.properties = propertiesWithSchema(row.properties, m.schemaByOrg[row.owner])
	detail.loaded = append([]ghclient.PropertyValue{}, row.properties...)
	detail.schema = append([]ghclient.PropertyDefinition{}, m.schemaByOrg[row.owner]...)
	detail.schemaByName = make(map[string]ghclient.PropertyDefinition, len(detail.schema))
	for _, def := range detail.schema {
		detail.schemaByName[def.Name] = def
	}
	m.detail = detail
	m.detailRow = rowIndex
	m.screen = bScreenDetail
}

func propertiesWithSchema(properties []ghclient.PropertyValue, schema []ghclient.PropertyDefinition) []ghclient.PropertyValue {
	result := append([]ghclient.PropertyValue{}, properties...)
	set := make(map[string]bool, len(result))
	for _, property := range result {
		set[property.Name] = true
	}
	for _, def := range schema {
		if !set[def.Name] {
			result = append(result, ghclient.PropertyValue{Name: def.Name})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func propertiesWithValues(properties []ghclient.PropertyValue) []ghclient.PropertyValue {
	result := make([]ghclient.PropertyValue, 0, len(properties))
	for _, property := range properties {
		if property.Value != nil {
			result = append(result, property)
		}
	}
	return result
}

// deferAction defers a hub-screen command's effect until its footer key
// finishes flashing (see keys.go's pendingFlash), instead of running it
// immediately.
func (m *batchModel) deferAction(label string, action func() (tea.Model, tea.Cmd)) (tea.Model, tea.Cmd) {
	m.pendingFlash = &pendingFlash{label: label, action: action}
	return m, flashTick()
}

func (m *batchModel) handleChooseActionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "1":
		m.action = bulkSet
		m.startChooseProperty()
		return m, m.editor.Init()
	case "2":
		m.action = bulkDelete
		m.startChooseProperty()
		return m, m.editor.Init()
	case "esc":
		m.screen = bScreenTable
	}
	return m, nil
}

func (m *batchModel) handleChoosePropertyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cmd, outcome := m.editor.Update(msg)
	switch outcome {
	case outcomeDone:
		m.startChooseTargets(m.editor.Result())
		m.editor = nil
		return m, nil
	case outcomeCancelled:
		m.editor = nil
		m.screen = bScreenTable
		return m, nil
	default:
		return m, cmd
	}
}

func (m *batchModel) handleChooseTargetsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch s := msg.String(); {
	case s == "up" || s == "k":
		m.targets.up()
	case s == "down" || s == "j":
		m.targets.down()
	case isPageUpKey(s):
		m.targets.pageUp()
	case isPageDownKey(s):
		m.targets.pageDown()
	case s == " ":
		m.targets.toggle()
	case s == "a":
		for i := range m.targetRow {
			m.targets.checked[i] = true
		}
	case s == "n":
		m.targets.checked = map[int]bool{}
	case s == "enter":
		return m, m.startApply()
	case s == "esc":
		m.screen = bScreenTable
	}
	return m, nil
}

// bulkSchema returns the schema used to drive the bulk-edit property picker:
// the first successfully loaded row's org schema, or nil (freeform) if
// unavailable. Batch mode assumes the repo list is drawn from a single org;
// mixed-org batches fall back to per-repo API errors on apply if a chosen
// property isn't valid for a given repo's org.
func (m *batchModel) bulkSchema() []ghclient.PropertyDefinition {
	for _, r := range m.rows {
		if r.err == nil {
			return m.schemaByOrg[r.owner]
		}
	}
	return nil
}

func (m *batchModel) startChooseProperty() {
	schema := m.bulkSchema()
	maxVisible := availableRows(m.height)
	if m.action == bulkDelete {
		m.editor = newDeleteEditor(schema, schema == nil, maxVisible)
	} else {
		m.editor = newAddEditor(schema, schema == nil, maxVisible)
	}
	m.screen = bScreenChooseProperty
}

func (m *batchModel) startChooseTargets(result ghclient.PropertyValue) {
	m.pendingAction = result
	labels := make([]string, 0, len(m.rows))
	m.targetRow = nil
	for i, r := range m.rows {
		if r.err != nil {
			continue
		}
		labels = append(labels, r.label())
		m.targetRow = append(m.targetRow, i)
	}
	m.targets = newOptionPicker(labels, true)
	m.targets.maxVisible = availableRows(m.height)
	for i := range labels {
		m.targets.checked[i] = true
	}
	m.screen = bScreenChooseTargets
}

func (m *batchModel) startApply() tea.Cmd {
	var targetIdx []int
	for i := range m.targets.options {
		if m.targets.checked[i] {
			targetIdx = append(targetIdx, m.targetRow[i])
		}
	}
	if len(targetIdx) == 0 {
		return nil
	}

	name, value := m.pendingAction.Name, m.pendingAction.Value

	snapshots := make([]backup.RepoSnapshot, 0, len(targetIdx))
	for _, idx := range targetIdx {
		r := m.rows[idx]
		snapshots = append(snapshots, backup.RepoSnapshot{Owner: r.owner, Repo: r.repo, PropertiesBefore: r.loaded})
	}

	dir := m.backupDir
	api := m.api
	rows := append([]repoRow{}, m.rows...)
	m.screen = bScreenApplying

	return func() tea.Msg {
		path, err := backup.Write(dir, backup.ModeBatch, snapshots)
		if err != nil {
			return batchAppliedMsg{backupErr: fmt.Errorf("write backup: %w", err)}
		}
		results := applyToTargets(api, rows, targetIdx, name, value)
		return batchAppliedMsg{backupPath: path, results: results}
	}
}

// applyToTargets patches the given property on each target repo, using a
// bounded pool of goroutines, and returns one result per target.
func applyToTargets(api ghclient.PropertiesAPI, rows []repoRow, targetIdx []int, name string, value any) []batchApplyResult {
	jobs := make(chan repoRow)
	results := make(chan batchApplyResult, len(targetIdx))

	workers := batchApplyWorkers
	if workers > len(targetIdx) {
		workers = len(targetIdx)
	}
	if workers < 1 {
		workers = 1
	}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for row := range jobs {
				err := api.SetRepoProperties(context.Background(), row.owner, row.repo,
					[]ghclient.PropertyValue{{Name: name, Value: value}})
				results <- batchApplyResult{owner: row.owner, repo: row.repo, err: err}
			}
		}()
	}

	go func() {
		for _, idx := range targetIdx {
			jobs <- rows[idx]
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
	}()

	out := make([]batchApplyResult, 0, len(targetIdx))
	for r := range results {
		out = append(out, r)
	}
	return out
}

// applyResultsToRows reflects successful patches back into m.rows, so the
// table view and any subsequent bulk edit's backup "before" state stay in
// sync with what was actually written to GitHub.
func (m *batchModel) applyResultsToRows(results []batchApplyResult) {
	name, value := m.pendingAction.Name, m.pendingAction.Value
	for _, r := range results {
		if r.err != nil {
			continue
		}
		for i := range m.rows {
			if m.rows[i].owner != r.owner || m.rows[i].repo != r.repo {
				continue
			}
			m.rows[i].properties = upsertOrDeleteProperty(m.rows[i].properties, name, value)
			m.rows[i].loaded = upsertOrDeleteProperty(m.rows[i].loaded, name, value)
			break
		}
	}
}

// upsertOrDeleteProperty returns props with name set to value, or with name
// removed entirely when value is nil.
func upsertOrDeleteProperty(props []ghclient.PropertyValue, name string, value any) []ghclient.PropertyValue {
	if value == nil {
		out := make([]ghclient.PropertyValue, 0, len(props))
		for _, p := range props {
			if p.Name != name {
				out = append(out, p)
			}
		}
		return out
	}
	for i, p := range props {
		if p.Name == name {
			props[i] = ghclient.PropertyValue{Name: name, Value: value}
			return props
		}
	}
	return append(props, ghclient.PropertyValue{Name: name, Value: value})
}

// headerText returns the plain identity text shown on the right side of the
// top bar: the org the batch is drawn from (or a note that it spans
// multiple orgs) and how many repos are involved.
func (m *batchModel) headerText() string {
	count := len(m.entries)
	org := ownerOf(m.entries)
	if len(m.rows) > 0 {
		count = len(m.rows)
		org = ownerOfRows(m.rows)
	}
	if org == "" {
		return fmt.Sprintf("%d repo(s) — multiple orgs", count)
	}
	return fmt.Sprintf("%s — %d repo(s)", org, count)
}

// topBarItems returns the commands shown in the top bar for the current
// screen — only the repo table has commands worth showing there.
func (m *batchModel) topBarItems() []menubar.MenuItem {
	if m.screen != bScreenTable {
		return nil
	}
	return []menubar.MenuItem{
		{Label: "Bulk edit", Hotkey: "b"},
		menubar.Separator(),
		{Label: "Filter", Hotkey: "?"},
		{Label: "Refresh", Hotkey: "F5"},
		{Label: "Help", Hotkey: "F1"},
		menubar.Separator(),
		{Label: "Quit", Hotkey: "q"},
	}
}

// ownerOf/ownerOfRows return the common owner across entries/rows, or "" if
// they span more than one.
func ownerOf(entries []repolist.Entry) string {
	if len(entries) == 0 {
		return ""
	}
	org := entries[0].Owner
	for _, e := range entries {
		if e.Owner != org {
			return ""
		}
	}
	return org
}

func ownerOfRows(rows []repoRow) string {
	if len(rows) == 0 {
		return ""
	}
	org := rows[0].owner
	for _, r := range rows {
		if r.owner != org {
			return ""
		}
	}
	return org
}

func (m *batchModel) View() string {
	if m.detail != nil {
		return m.detail.View()
	}
	top := renderTopBar(m.topBarItems(), m.headerText(), m.width)
	if m.screen == bScreenTable {
		content, footer := m.viewTableParts()
		return boxStyle.Render(pinFooter(top+"\n\n"+content, footer, m.height))
	}
	return boxStyle.Render(top + "\n\n" + m.viewBody())
}

func (m *batchModel) viewBody() string {
	switch m.screen {
	case bScreenLoading:
		return m.viewLoading()
	case bScreenChooseAction:
		return m.viewChooseAction()
	case bScreenChooseProperty:
		return m.editor.View()
	case bScreenChooseTargets:
		return m.viewChooseTargets()
	case bScreenApplying:
		return fmt.Sprintf("\n  %s Applying changes to %d repo(s)...\n", m.spin.View(), len(m.targetRow))
	case bScreenResult:
		return m.viewResult()
	case bScreenExporting:
		return fmt.Sprintf("\n  %s Exporting %d repo(s)...\n", m.spin.View(), len(m.rows))
	case bScreenExportResult:
		return m.viewExportResult()
	case bScreenConfirmUnload:
		return m.viewConfirmUnload()
	}
	return ""
}

func (m *batchModel) viewConfirmUnload() string {
	if m.pendingUnloadAt < 0 || m.pendingUnloadAt >= len(m.rows) {
		return ""
	}
	return warnStyle.Render(fmt.Sprintf("Unload %s from the active repo list?", m.rows[m.pendingUnloadAt].label())) + "\n\n" +
		helpLine(keyBinding("y/enter", "confirm"), keyBinding("n/esc", "cancel"))
}

func (m *batchModel) viewExportResult() string {
	if m.exportErr != nil {
		return errorStyle.Render(wrapToWidth("Failed to export CSV: "+m.exportErr.Error(), m.width-outerFrameWidth)) + "\n\n" +
			helpLine(keyBinding("any key", "back to repo list"))
	}
	return successStyle.Render("Exported CSV") + "\n" +
		dimStyle.Render(wrapToWidth("File: "+m.exportPath, m.width-outerFrameWidth)) + "\n\n" +
		helpLine(keyBinding("any key", "back to repo list"))
}

func (m *batchModel) viewLoading() string {
	pct := 0.0
	if m.chunksTotal > 0 {
		pct = float64(m.chunksDone) / float64(m.chunksTotal)
	}
	return fmt.Sprintf("\n  %s Loading %d repo(s)...\n\n  %s\n", m.spin.View(), len(m.entries), m.prog.ViewAs(pct))
}

// viewTableParts renders the repo table screen, split into scrollable
// content and its footer so View can pin the footer to the terminal's last
// row instead of letting it float wherever the content happens to end.
func (m *batchModel) viewTableParts() (content, footer string) {
	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("%d repo(s) loaded", len(m.rows))) + "\n\n")
	if len(m.skipped) > 0 {
		b.WriteString(warnStyle.Render(fmt.Sprintf("%d line(s) in the repo list could not be parsed and were skipped", len(m.skipped))) + "\n\n")
	}

	if m.filtering || m.filter != "" {
		b.WriteString(dimStyle.Render("Filter: "+m.filter) + "\n")
	}

	indices := m.filteredRowIndices()
	if len(m.rows) > 0 && len(indices) == 0 {
		b.WriteString(dimStyle.Render("(no matches)") + "\n")
	}
	start, end := visibleWindow(len(indices), m.cursor, availableRows(m.height))
	if start > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↑ %d more above", start)) + "\n")
	}
	for i := start; i < end; i++ {
		r := m.rows[indices[i]]
		var line string
		switch {
		case r.err != nil:
			line = fmt.Sprintf("%-40s %s", r.label(), errorStyle.Render("fetch failed: "+r.err.Error()))
		case len(r.properties) == 0:
			line = fmt.Sprintf("%-40s %s", r.label(), dimStyle.Render("(no custom properties set)"))
		default:
			line = fmt.Sprintf("%-40s %s", r.label(), summarizeProperties(r.properties))
		}
		line = truncateToWidth(line, rowContentWidth(m.width))
		if i == m.cursor {
			b.WriteString(cursorStyle.Render("> ") + selectedStyle.Render(line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}
	if end < len(indices) {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ %d more below", len(indices)-end)) + "\n")
	}

	if m.err != nil {
		b.WriteString("\n" + errorStyle.Render(m.err.Error()) + "\n")
	}

	footer = renderFooterPanel(m.help, m.width, batchTableHelpKeys{})
	return b.String(), footer
}

func summarizeProperties(props []ghclient.PropertyValue) string {
	parts := make([]string, len(props))
	for i, p := range props {
		parts[i] = p.Name + "=" + formatValue(p.Value)
	}
	return strings.Join(parts, ", ")
}

func (m *batchModel) viewChooseAction() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Bulk edit") + "\n\n")
	b.WriteString("1) Set a property across selected repos\n")
	b.WriteString("2) Delete a property from selected repos\n")
	b.WriteString("\n" + helpLine(keyBinding("1/2", "choose"), keyBinding("esc", "back")))
	return b.String()
}

func (m *batchModel) viewChooseTargets() string {
	var b strings.Builder
	verb := "Set"
	if m.action == bulkDelete {
		verb = "Delete"
	}
	desc := m.pendingAction.Name
	if m.action == bulkSet {
		desc = fmt.Sprintf("%s = %s", m.pendingAction.Name, formatValue(m.pendingAction.Value))
	}
	b.WriteString(titleStyle.Render(fmt.Sprintf("%s %s — choose target repos", verb, desc)) + "\n\n")
	b.WriteString(m.targets.View())
	b.WriteString("\n" + helpLine(
		keyBinding("space", "toggle"), keyBinding("a", "all"), keyBinding("n", "none"),
		keyBinding("enter", "apply"), keyBinding("esc", "cancel"),
	))
	return b.String()
}

func (m *batchModel) viewResult() string {
	var b strings.Builder
	if m.backupErr != nil {
		contentWidth := m.width - outerFrameWidth
		b.WriteString(errorStyle.Render(wrapToWidth("Failed to write backup, no changes were applied: "+m.backupErr.Error(), contentWidth)) + "\n")
		b.WriteString("\n" + helpLine(keyBinding("any key", "back to repo list")))
		return b.String()
	}

	succeeded, failed := 0, 0
	for _, r := range m.applyResults {
		if r.err != nil {
			failed++
		} else {
			succeeded++
		}
	}
	b.WriteString(dimStyle.Render("Backup of prior values: "+m.backupPath) + "\n\n")
	b.WriteString(successStyle.Render(fmt.Sprintf("%d succeeded", succeeded)))
	if failed > 0 {
		b.WriteString("  " + errorStyle.Render(fmt.Sprintf("%d failed", failed)))
	}
	b.WriteString("\n\n")
	for _, r := range m.applyResults {
		if r.err != nil {
			contentWidth := m.width - outerFrameWidth
			b.WriteString(errorStyle.Render(wrapToWidth(fmt.Sprintf("  %s/%s: %s", r.owner, r.repo, r.err), contentWidth)) + "\n")
		}
	}
	b.WriteString("\n" + helpLine(keyBinding("any key", "back to repo list")))
	return b.String()
}
