package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/repolist"
	tea "github.com/charmbracelet/bubbletea"
)

func newTableModel(t *testing.T, api *fakeAPI, rows []repoRow, schemaByOrg map[string][]ghclient.PropertyDefinition) *batchModel {
	t.Helper()
	entries := make([]repolist.Entry, len(rows))
	for i, r := range rows {
		entries[i] = repolist.Entry{Owner: r.owner, Repo: r.repo, Line: i + 1}
	}
	m := newBatchModel(api, t.TempDir(), entries, nil)
	m.chunksTotal = 1
	next, _ := m.handleChunk(batchChunkMsg{rows: rows, schemas: schemaByOrg})
	return next.(*batchModel)
}

// settleBatch fires the pending flash (see keys.go's pendingFlash) directly,
// without waiting out the real flashDuration, so a hub-screen command key
// (b/q) reaches its final state in tests immediately.
func settleBatch(m *batchModel) *batchModel {
	next, _ := m.Update(flashElapsedMsg{})
	return next.(*batchModel)
}

func manyRows(n int) []repoRow {
	rows := make([]repoRow, n)
	for i := range rows {
		rows[i] = repoRow{owner: "acme", repo: fmt.Sprintf("repo-%02d", i)}
	}
	return rows
}

func TestBatchModel_TableScrollsWhenTerminalIsShort(t *testing.T) {
	m := newTableModel(t, &fakeAPI{}, manyRows(40), nil)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = next.(*batchModel)

	view := m.View()
	if !strings.Contains(view, "more below") {
		t.Errorf("expected a scroll-down indicator with 40 rows in a 20-row terminal:\n%s", view)
	}
	if strings.Contains(view, "repo-39") {
		t.Errorf("last row should be scrolled out of view initially:\n%s", view)
	}

	for i := 0; i < 39; i++ {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = next.(*batchModel)
	}
	view = m.View()
	if !strings.Contains(view, "more above") {
		t.Errorf("expected a scroll-up indicator once cursor reaches the last row:\n%s", view)
	}
	if !strings.Contains(view, "repo-39") {
		t.Errorf("last row should be visible once the cursor reaches it:\n%s", view)
	}
}

func TestBatchModel_TableHasHeadersAndScrollsHorizontally(t *testing.T) {
	rows := []repoRow{
		{
			owner: "acme",
			repo:  "service-one",
			properties: []ghclient.PropertyValue{
				{Name: "data-classification", Value: "confidential"},
				{Name: "deployment-environment", Value: "production"},
				{Name: "service-owner", Value: "platform-engineering"},
			},
		},
	}
	m := newTableModel(t, &fakeAPI{}, rows, nil)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = next.(*batchModel)

	initial := m.View()
	if !strings.Contains(initial, "Repository") || !strings.Contains(initial, "data-classification") {
		t.Fatalf("initial table is missing column headers:\n%s", initial)
	}
	if !strings.Contains(initial, "service-one") || !strings.Contains(initial, "confidential") {
		t.Fatalf("initial table is missing aligned row values:\n%s", initial)
	}
	if !strings.Contains(initial, "›") {
		t.Fatalf("wide table is missing its right overflow indicator:\n%s", initial)
	}

	for i := 0; i < 20; i++ {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
		m = next.(*batchModel)
	}
	if m.horizontal == 0 {
		t.Fatal("right arrow did not move the horizontal viewport")
	}
	scrolled := m.View()
	if scrolled == initial || !strings.Contains(scrolled, "‹") {
		t.Fatalf("table did not render its horizontally scrolled state:\n%s", scrolled)
	}
	if !strings.Contains(scrolled, "Status") {
		t.Fatalf("rightmost column is not visible after scrolling to the end:\n%s", scrolled)
	}

	for i := 0; i < 20; i++ {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
		m = next.(*batchModel)
	}
	if m.horizontal != 0 {
		t.Fatalf("left arrow offset = %d, want 0", m.horizontal)
	}
}

func TestRenderTableRow_TruncatesRepositoryFromLeft(t *testing.T) {
	columns := []tableColumn{{name: "Repository", width: 20, truncateFromLeft: true}}
	got := renderTableRow(columns, []string{"acme/a-very-long-repository-name"})

	if !strings.HasPrefix(got, "...") {
		t.Fatalf("renderTableRow() = %q, want a leading ellipsis", got)
	}
	if !strings.HasSuffix(got, "repository-name") {
		t.Fatalf("renderTableRow() = %q, want the repository-name suffix preserved", got)
	}
	if len([]rune(got)) != columns[0].width {
		t.Fatalf("rendered width = %d, want %d", len([]rune(got)), columns[0].width)
	}
}

func TestBatchModel_FooterPinnedToLastRow(t *testing.T) {
	m := newTableModel(t, &fakeAPI{}, manyRows(3), nil)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = next.(*batchModel)

	lines := strings.Split(m.View(), "\n")
	if len(lines) != 20 {
		t.Fatalf("got %d lines, want 20:\n%s", len(lines), m.View())
	}
	if !strings.Contains(lines[len(lines)-1], "└") {
		t.Errorf("last line = %q, want the footer panel's bottom border", lines[len(lines)-1])
	}
	if !strings.Contains(m.View(), "quit") {
		t.Errorf("view is missing the quit hint:\n%s", m.View())
	}
}

func TestBatchModel_PageDownAndUp(t *testing.T) {
	m := newTableModel(t, &fakeAPI{}, manyRows(40), nil)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = next.(*batchModel)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = next.(*batchModel)
	if m.cursor == 0 {
		t.Fatal("PgDown should have moved the cursor")
	}

	for i := 0; i < 10; i++ {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyF8})
		m = next.(*batchModel)
	}
	if m.cursor != len(m.rows)-1 {
		t.Errorf("cursor = %d, want clamped to last index %d", m.cursor, len(m.rows)-1)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	m = next.(*batchModel)
	if m.cursor == len(m.rows)-1 {
		t.Error("PgUp-equivalent should have moved the cursor back")
	}
}

func TestBatchModel_F5Refetches(t *testing.T) {
	api := &fakeAPI{}
	m := newTableModel(t, api, manyRows(2), nil)
	if api.getCallsCount != 0 {
		t.Fatalf("setup: getCallsCount = %d, want 0 before any F5", api.getCallsCount)
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyF5})
	m = next.(*batchModel)
	if m.screen != bScreenLoading || len(m.rows) != 0 {
		t.Fatalf("after F5: screen=%v rows=%v, want bScreenLoading with rows reset", m.screen, m.rows)
	}
	if cmd == nil {
		t.Fatal("F5 should return a fetch command")
	}

	batch, ok := cmd().(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected a tea.BatchMsg, got %T", cmd())
	}
	for _, c := range batch {
		if chunkMsg, ok := c().(batchChunkMsg); ok {
			next, _ = m.Update(chunkMsg)
			m = next.(*batchModel)
		}
	}

	if m.screen != bScreenTable || len(m.rows) != 2 {
		t.Fatalf("after refetch completes: screen=%v rows=%v", m.screen, m.rows)
	}
	if api.getCallsCount != 2 {
		t.Errorf("getCallsCount = %d, want 2 (one per row)", api.getCallsCount)
	}
}

func TestBatchModel_UnloadRemovesRepoFromActiveList(t *testing.T) {
	api := &fakeAPI{}
	rows := []repoRow{
		{owner: "acme", repo: "one"},
		{owner: "acme", repo: "two"},
	}
	m := newTableModel(t, api, rows, nil)

	next, _ := m.Update(runeKey('u'))
	m = settleBatch(next.(*batchModel))
	if m.screen != bScreenConfirmUnload || m.pendingUnloadAt != 0 {
		t.Fatalf("after unload command: screen=%v pending=%d, want confirmation for first row", m.screen, m.pendingUnloadAt)
	}
	if !strings.Contains(m.View(), "Unload acme/one") {
		t.Fatalf("confirmation view missing selected repo:\n%s", m.View())
	}

	next, _ = m.Update(runeKey('n'))
	m = next.(*batchModel)
	if len(m.rows) != 2 || m.screen != bScreenTable {
		t.Fatalf("after cancel: rows=%d screen=%v, want 2/table", len(m.rows), m.screen)
	}

	next, _ = m.Update(runeKey('u'))
	m = settleBatch(next.(*batchModel))
	next, _ = m.Update(runeKey('y'))
	m = next.(*batchModel)
	if len(m.rows) != 1 || m.rows[0].repo != "two" {
		t.Fatalf("after confirm: rows=%v, want only acme/two", m.rows)
	}
	if len(m.entries) != 1 || m.entries[0].Repo != "two" {
		t.Fatalf("after confirm: entries=%v, want only acme/two", m.entries)
	}

	// A refresh must not reintroduce the unloaded repo.
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyF5})
	m = next.(*batchModel)
	if cmd == nil || len(m.entries) != 1 {
		t.Fatalf("after refresh: cmd=%v entries=%v, want one retained entry", cmd != nil, m.entries)
	}
}

func TestBatchModel_RefreshWithNoActiveReposReturnsToTable(t *testing.T) {
	m := newTableModel(t, &fakeAPI{}, []repoRow{{owner: "acme", repo: "one"}}, nil)
	next, _ := m.Update(runeKey('u'))
	m = settleBatch(next.(*batchModel))
	next, _ = m.Update(runeKey('y'))
	m = next.(*batchModel)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyF5})
	m = next.(*batchModel)
	if cmd != nil || m.screen != bScreenTable || len(m.rows) != 0 {
		t.Fatalf("refresh with no repos: cmd=%v screen=%v rows=%d, want nil/table/0", cmd != nil, m.screen, len(m.rows))
	}
}

func TestBatchModel_FilterNarrowsTable(t *testing.T) {
	rows := []repoRow{
		{owner: "acme", repo: "alpha"},
		{owner: "acme", repo: "beta"},
		{owner: "acme", repo: "gamma"},
	}
	m := newTableModel(t, &fakeAPI{}, rows, nil)

	next, _ := m.Update(runeKey('?'))
	m = next.(*batchModel)
	if !m.filtering {
		t.Fatal("'?' should enter filtering mode")
	}
	for _, r := range "beta" {
		next, _ = m.Update(runeKey(r))
		m = next.(*batchModel)
	}
	indices := m.filteredRowIndices()
	if len(indices) != 1 || m.rows[indices[0]].repo != "beta" {
		t.Fatalf("filteredRowIndices() = %v, want exactly the beta row", indices)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc}) // clear the filter
	m = next.(*batchModel)
	if m.filter != "" || m.filtering {
		t.Errorf("filter=%q filtering=%v, want cleared", m.filter, m.filtering)
	}
	if len(m.filteredRowIndices()) != 3 {
		t.Errorf("filteredRowIndices() after clear = %d, want all 3", len(m.filteredRowIndices()))
	}
}

func TestBatchModel_AltBTriggersBulkEdit(t *testing.T) {
	m := newTableModel(t, &fakeAPI{}, manyRows(2), nil)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true})
	m = settleBatch(next.(*batchModel))
	if m.screen != bScreenChooseAction {
		t.Errorf("after alt+b: screen=%v, want bScreenChooseAction", m.screen)
	}
}

func TestBatchModel_HeaderShowsOrgAndCount(t *testing.T) {
	api := &fakeAPI{}
	rows := []repoRow{
		{owner: "acme", repo: "a"},
		{owner: "acme", repo: "b"},
	}
	m := newTableModel(t, api, rows, nil)
	if !strings.Contains(m.View(), "acme — 2 repo(s)") {
		t.Errorf("table screen View() missing org/count header:\n%s", m.View())
	}

	next, _ := m.Update(runeKey('b'))
	m = settleBatch(next.(*batchModel))
	if !strings.Contains(m.View(), "acme — 2 repo(s)") {
		t.Errorf("choose-action screen View() missing org/count header:\n%s", m.View())
	}
}

func TestBatchModel_HeaderNotesMixedOrgs(t *testing.T) {
	api := &fakeAPI{}
	rows := []repoRow{
		{owner: "acme", repo: "a"},
		{owner: "other-org", repo: "b"},
	}
	m := newTableModel(t, api, rows, nil)
	if !strings.Contains(m.View(), "multiple orgs") {
		t.Errorf("table screen View() should note mixed orgs:\n%s", m.View())
	}
}

func TestBatchModel_SchemaSortedAlphabetically(t *testing.T) {
	api := &fakeAPI{}
	rows := []repoRow{{owner: "acme", repo: "a"}}
	schema := map[string][]ghclient.PropertyDefinition{
		"acme": {
			{Name: "Zebra", Type: ghclient.PropertyTypeString},
			{Name: "Apple", Type: ghclient.PropertyTypeSingleSelect, AllowedValues: []string{"Charlie", "Alpha", "Bravo"}},
			{Name: "Mango", Type: ghclient.PropertyTypeString},
		},
	}
	m := newTableModel(t, api, rows, schema)

	got := m.schemaByOrg["acme"]
	if got[0].Name != "Apple" || got[1].Name != "Mango" || got[2].Name != "Zebra" {
		t.Fatalf("schemaByOrg[acme] names = %v, want alphabetical order", []string{got[0].Name, got[1].Name, got[2].Name})
	}
	if av := got[0].AllowedValues; av[0] != "Alpha" || av[1] != "Bravo" || av[2] != "Charlie" {
		t.Errorf("Apple.AllowedValues = %v, want alphabetical order", av)
	}
}

func TestBatchModel_BulkSetAcrossRepos(t *testing.T) {
	api := &fakeAPI{}
	rows := []repoRow{
		{owner: "acme", repo: "a", properties: nil, loaded: nil},
		{owner: "acme", repo: "b", properties: nil, loaded: nil},
	}
	schema := map[string][]ghclient.PropertyDefinition{
		"acme": {{Name: "tier", Type: ghclient.PropertyTypeSingleSelect, AllowedValues: []string{"1", "2"}}},
	}
	m := newTableModel(t, api, rows, schema)
	if m.screen != bScreenTable || len(m.rows) != 2 {
		t.Fatalf("setup: screen=%v rows=%v, want bScreenTable/2 rows", m.screen, m.rows)
	}

	next, _ := m.Update(runeKey('b'))
	m = settleBatch(next.(*batchModel))
	if m.screen != bScreenChooseAction {
		t.Fatalf("after 'b': screen=%v, want bScreenChooseAction", m.screen)
	}

	next, _ = m.Update(runeKey('1')) // bulk set
	m = next.(*batchModel)
	if m.screen != bScreenChooseProperty || m.editor == nil {
		t.Fatalf("after '1': screen=%v editor=%v, want bScreenChooseProperty", m.screen, m.editor)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // pick "tier" (only candidate)
	m = next.(*batchModel)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // confirm value "1"
	m = next.(*batchModel)
	if m.screen != bScreenChooseTargets || m.targets == nil {
		t.Fatalf("after confirming value: screen=%v targets=%v, want bScreenChooseTargets", m.screen, m.targets)
	}
	if len(m.targets.selectedValues()) != 2 {
		t.Fatalf("targets default selection = %v, want all 2 repos selected", m.targets.selectedValues())
	}

	cmd := m.startApply()
	if cmd == nil {
		t.Fatal("startApply returned a nil cmd")
	}
	msg := cmd()
	next, _ = m.Update(msg)
	m = next.(*batchModel)

	if m.screen != bScreenResult {
		t.Fatalf("after apply: screen=%v, want bScreenResult", m.screen)
	}
	if len(m.applyResults) != 2 {
		t.Fatalf("applyResults = %v, want 2 entries", m.applyResults)
	}
	for _, r := range m.applyResults {
		if r.err != nil {
			t.Errorf("unexpected error applying to %s/%s: %v", r.owner, r.repo, r.err)
		}
	}
	for _, row := range m.rows {
		if len(row.properties) != 1 || row.properties[0].Name != "tier" || row.properties[0].Value != "1" {
			t.Errorf("row %s/%s properties = %v, want tier=1 reflected after apply", row.owner, row.repo, row.properties)
		}
	}
	if len(api.setCalls) != 2 {
		t.Errorf("SetRepoProperties called %d times, want 2", len(api.setCalls))
	}
}

func TestBatchModel_BulkDeleteSendsNilValue(t *testing.T) {
	api := &fakeAPI{}
	rows := []repoRow{
		{owner: "acme", repo: "a", properties: []ghclient.PropertyValue{{Name: "tier", Value: "1"}}, loaded: []ghclient.PropertyValue{{Name: "tier", Value: "1"}}},
	}
	schema := map[string][]ghclient.PropertyDefinition{
		"acme": {{Name: "tier", Type: ghclient.PropertyTypeSingleSelect, AllowedValues: []string{"1", "2"}}},
	}
	m := newTableModel(t, api, rows, schema)

	next, _ := m.Update(runeKey('b'))
	m = settleBatch(next.(*batchModel))
	next, _ = m.Update(runeKey('2')) // bulk delete
	m = next.(*batchModel)
	if !m.editor.deleteMode {
		t.Fatalf("expected deleteMode editor")
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // pick "tier", completes immediately in delete mode
	m = next.(*batchModel)
	if m.screen != bScreenChooseTargets {
		t.Fatalf("after picking name in delete mode: screen=%v, want bScreenChooseTargets", m.screen)
	}
	if m.pendingAction.Value != nil {
		t.Errorf("pendingAction.Value = %v, want nil for delete", m.pendingAction.Value)
	}

	cmd := m.startApply()
	msg := cmd()
	next, _ = m.Update(msg)
	m = next.(*batchModel)

	if len(m.rows[0].properties) != 0 {
		t.Errorf("row properties after delete = %v, want empty", m.rows[0].properties)
	}
	if len(api.setCalls) != 1 || api.setCalls[0][0].Value != nil {
		t.Errorf("SetRepoProperties calls = %v, want one call with nil value", api.setCalls)
	}
}

func TestBatchModel_OpensIndividualRepoDetail(t *testing.T) {
	api := &fakeAPI{}
	rows := []repoRow{{
		owner:      "acme",
		repo:       "service",
		properties: []ghclient.PropertyValue{{Name: "team", Value: "platform"}},
		loaded:     []ghclient.PropertyValue{{Name: "team", Value: "platform"}},
	}}
	schema := map[string][]ghclient.PropertyDefinition{
		"acme": {
			{Name: "lifecycle", Type: ghclient.PropertyTypeSingleSelect, AllowedValues: []string{"active", "legacy"}},
			{Name: "team", Type: ghclient.PropertyTypeString},
		},
	}
	m := newTableModel(t, api, rows, schema)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m = next.(*batchModel)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = settleBatch(next.(*batchModel))
	if m.detail == nil || m.screen != bScreenDetail {
		t.Fatalf("after enter: detail=%v screen=%v, want detail screen", m.detail, m.screen)
	}
	if m.detail.width != 60 || m.detail.height != 20 {
		t.Fatalf("detail size = %dx%d, want 60x20", m.detail.width, m.detail.height)
	}
	if got := m.detail.properties; len(got) != 2 || got[0].Name != "lifecycle" || got[0].Value != nil {
		t.Fatalf("detail properties = %#v, want unset schema field included first", got)
	}
	if !strings.Contains(m.View(), "lifecycle") || !strings.Contains(m.View(), "team") {
		t.Errorf("detail view should show all fields:\n%s", m.View())
	}

	// Edit the unset lifecycle field and apply it from the detail view.
	next, _ = m.detail.Update(runeKey('e'))
	m.detail = next.(*singleRepoModel)
	m = settleBatch(m)
	next, _ = m.detail.Update(tea.KeyMsg{Type: tea.KeyEnter}) // choose lifecycle
	m.detail = next.(*singleRepoModel)
	next, _ = m.detail.Update(tea.KeyMsg{Type: tea.KeyEnter}) // choose active
	m.detail = next.(*singleRepoModel)
	if m.detail.screen != screenList {
		t.Fatalf("after detail edit: screen=%v, want list", m.detail.screen)
	}

	next, _ = m.detail.Update(runeKey('s'))
	m.detail = next.(*singleRepoModel)
	m = settleBatch(m)
	msg := m.detail.applyCmd()()
	next, _ = m.detail.Update(msg)
	m.detail = next.(*singleRepoModel)
	if m.detail.screen != screenResult || len(api.setCalls) != 1 {
		t.Fatalf("detail apply: screen=%v setCalls=%d, want result/1", m.detail.screen, len(api.setCalls))
	}

	// The next key dismisses the detail result and reflects the saved value.
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*batchModel)
	if m.detail != nil || m.screen != bScreenTable {
		t.Fatalf("after dismissing detail: detail=%v screen=%v, want table", m.detail, m.screen)
	}
	if len(m.rows[0].properties) != 2 || m.rows[0].properties[0].Name != "lifecycle" {
		t.Errorf("row properties after detail save = %#v, want lifecycle and team", m.rows[0].properties)
	}
}
