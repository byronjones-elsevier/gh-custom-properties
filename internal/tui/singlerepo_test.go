package tui

import (
	"testing"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	tea "github.com/charmbracelet/bubbletea"
)

func newLoadedModel(t *testing.T, props []ghclient.PropertyValue, schema []ghclient.PropertyDefinition, schemaErr error) *singleRepoModel {
	t.Helper()
	m := newSingleRepoModel(&fakeAPI{}, t.TempDir(), "octocat", "hello-world")
	next, _ := m.handleLoaded(loadedMsg{owner: "octocat", repo: "hello-world", properties: props, schema: schema, schemaErr: schemaErr})
	return next.(*singleRepoModel)
}

func runeKey(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func TestSingleRepoModel_DeleteConfirmAndCancel(t *testing.T) {
	m := newLoadedModel(t, []ghclient.PropertyValue{{Name: "team", Value: "platform"}}, nil, ghclient.ErrSchemaUnavailable)

	next, _ := m.Update(runeKey('d'))
	m = next.(*singleRepoModel)
	if m.screen != screenConfirmDelete || m.pendingDeleteAt != 0 {
		t.Fatalf("after 'd': screen=%v pendingDeleteAt=%v, want screenConfirmDelete/0", m.screen, m.pendingDeleteAt)
	}

	next, _ = m.Update(runeKey('n'))
	m = next.(*singleRepoModel)
	if m.screen != screenList || len(m.properties) != 1 {
		t.Fatalf("after 'n': screen=%v properties=%v, want screenList with 1 property unchanged", m.screen, m.properties)
	}

	next, _ = m.Update(runeKey('d'))
	m = next.(*singleRepoModel)
	next, _ = m.Update(runeKey('y'))
	m = next.(*singleRepoModel)
	if m.screen != screenList || len(m.properties) != 0 {
		t.Fatalf("after 'd' then 'y': screen=%v properties=%v, want screenList with property removed", m.screen, m.properties)
	}
}

func TestSingleRepoModel_EditCancelLeavesValueUnchanged(t *testing.T) {
	m := newLoadedModel(t, []ghclient.PropertyValue{{Name: "team", Value: "platform"}}, nil, ghclient.ErrSchemaUnavailable)

	next, _ := m.Update(runeKey('e'))
	m = next.(*singleRepoModel)
	if m.screen != screenEdit || m.editor == nil {
		t.Fatalf("after 'e': screen=%v editor=%v, want screenEdit with an editor", m.screen, m.editor)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(*singleRepoModel)
	if m.screen != screenList || m.editor != nil {
		t.Fatalf("after esc: screen=%v editor=%v, want screenList with no editor", m.screen, m.editor)
	}
	if m.properties[0].Value != "platform" {
		t.Errorf("value changed after cancel: %v", m.properties[0].Value)
	}
}

func TestSingleRepoModel_EditSaveUpdatesValue(t *testing.T) {
	m := newLoadedModel(t, []ghclient.PropertyValue{{Name: "team", Value: "platform"}}, nil, ghclient.ErrSchemaUnavailable)

	next, _ := m.Update(runeKey('e'))
	m = next.(*singleRepoModel)

	m.editor.stringInput.SetValue("infra")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*singleRepoModel)

	if m.screen != screenList || m.editor != nil {
		t.Fatalf("after enter: screen=%v editor=%v, want screenList with no editor", m.screen, m.editor)
	}
	if m.properties[0].Value != "infra" {
		t.Errorf("properties[0].Value = %v, want %q", m.properties[0].Value, "infra")
	}
}

func TestSingleRepoModel_AddWithSchema(t *testing.T) {
	schema := []ghclient.PropertyDefinition{
		{Name: "tier", Type: ghclient.PropertyTypeSingleSelect, AllowedValues: []string{"1", "2", "3"}},
	}
	m := newLoadedModel(t, nil, schema, nil)

	next, _ := m.Update(runeKey('a'))
	m = next.(*singleRepoModel)
	if m.screen != screenEdit || m.editor == nil {
		t.Fatalf("after 'a': screen=%v editor=%v, want screenEdit", m.screen, m.editor)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // pick the only candidate name: "tier"
	m = next.(*singleRepoModel)
	if m.screen != screenEdit || m.editor.step != stepValue {
		t.Fatalf("after picking name: screen=%v step=%v, want screenEdit/stepValue", m.screen, m.editor.step)
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // confirm the first allowed value: "1"
	m = next.(*singleRepoModel)
	if m.screen != screenList || len(m.properties) != 1 {
		t.Fatalf("after confirming value: screen=%v properties=%v", m.screen, m.properties)
	}
	if m.properties[0].Name != "tier" || m.properties[0].Value != "1" {
		t.Errorf("properties[0] = %+v, want tier=1", m.properties[0])
	}
}

func TestSingleRepoModel_AddNoCandidatesShowsError(t *testing.T) {
	schema := []ghclient.PropertyDefinition{{Name: "tier", Type: ghclient.PropertyTypeString}}
	m := newLoadedModel(t, []ghclient.PropertyValue{{Name: "tier", Value: "1"}}, schema, nil)

	next, _ := m.Update(runeKey('a'))
	m = next.(*singleRepoModel)
	if m.screen != screenList || m.editor != nil || m.err == nil {
		t.Fatalf("after 'a' with no candidates: screen=%v editor=%v err=%v, want screenList/no editor/an error", m.screen, m.editor, m.err)
	}
}
