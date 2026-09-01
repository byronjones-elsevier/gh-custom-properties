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
	m = next.(*batchModel)
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
	m = next.(*batchModel)
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
	m = next.(*batchModel)
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
