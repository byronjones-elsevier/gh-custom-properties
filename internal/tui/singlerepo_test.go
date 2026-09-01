package tui

import (
	"fmt"
	"strings"
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

func manyProperties(n int) []ghclient.PropertyValue {
	props := make([]ghclient.PropertyValue, n)
	for i := range props {
		props[i] = ghclient.PropertyValue{Name: fmt.Sprintf("prop-%02d", i), Value: "v"}
	}
	return props
}

func TestSingleRepoModel_PropertyListScrollsWhenTerminalIsShort(t *testing.T) {
	m := newLoadedModel(t, manyProperties(40), nil, ghclient.ErrSchemaUnavailable)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = next.(*singleRepoModel)

	view := m.View()
	if !strings.Contains(view, "more below") {
		t.Errorf("expected a scroll-down indicator with 40 properties in a 20-row terminal:\n%s", view)
	}
	if strings.Contains(view, "prop-39") {
		t.Errorf("last property should be scrolled out of view initially:\n%s", view)
	}

	for i := 0; i < 39; i++ {
		next, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = next.(*singleRepoModel)
	}
	view = m.View()
	if !strings.Contains(view, "more above") {
		t.Errorf("expected a scroll-up indicator once cursor reaches the last property:\n%s", view)
	}
	if !strings.Contains(view, "prop-39") {
		t.Errorf("last property should be visible once the cursor reaches it:\n%s", view)
	}
}

func TestSingleRepoModel_OptionPickerScrollsWhenTerminalIsShort(t *testing.T) {
	allowed := make([]string, 40)
	for i := range allowed {
		allowed[i] = fmt.Sprintf("option-%02d", i)
	}
	schema := []ghclient.PropertyDefinition{{Name: "TechOrgGroup", Type: ghclient.PropertyTypeSingleSelect, AllowedValues: allowed}}
	m := newLoadedModel(t, nil, schema, nil)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = next.(*singleRepoModel)

	next, _ = m.Update(runeKey('a'))
	m = next.(*singleRepoModel)
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // pick the only candidate name
	m = next.(*singleRepoModel)

	view := m.View()
	if !strings.Contains(view, "more below") {
		t.Errorf("expected the value picker to scroll with 40 allowed values in a 20-row terminal:\n%s", view)
	}
	if strings.Contains(view, "option-39") {
		t.Errorf("last option should be scrolled out of view initially:\n%s", view)
	}
}

func TestSingleRepoModel_FooterPinnedToLastRow(t *testing.T) {
	m := newLoadedModel(t, []ghclient.PropertyValue{{Name: "team", Value: "platform"}}, nil, ghclient.ErrSchemaUnavailable)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	m = next.(*singleRepoModel)

	lines := strings.Split(m.View(), "\n")
	if len(lines) != 20 {
		t.Fatalf("got %d lines, want 20:\n%s", len(lines), m.View())
	}
	if !strings.Contains(lines[len(lines)-1], "quit") {
		t.Errorf("last line = %q, want the footer", lines[len(lines)-1])
	}
}

func TestSingleRepoModel_HeaderShownOnEveryScreen(t *testing.T) {
	m := newLoadedModel(t, []ghclient.PropertyValue{{Name: "team", Value: "platform"}}, nil, ghclient.ErrSchemaUnavailable)
	const want = "octocat/hello-world"

	if !strings.Contains(m.View(), want) {
		t.Errorf("list screen View() missing header %q:\n%s", want, m.View())
	}

	next, _ := m.Update(runeKey('e'))
	m = next.(*singleRepoModel)
	if !strings.Contains(m.View(), want) {
		t.Errorf("edit screen View() missing header %q:\n%s", want, m.View())
	}

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = next.(*singleRepoModel)
	next, _ = m.Update(runeKey('d'))
	m = next.(*singleRepoModel)
	if !strings.Contains(m.View(), want) {
		t.Errorf("confirm-delete screen View() missing header %q:\n%s", want, m.View())
	}
}

func TestSingleRepoModel_InputScreenHasNoHeader(t *testing.T) {
	m := newSingleRepoModel(&fakeAPI{}, t.TempDir(), "", "")
	if strings.Contains(m.View(), "─") {
		t.Errorf("input screen View() should have no header rule yet (no repo chosen):\n%s", m.View())
	}
}

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

func TestSingleRepoModel_EditRejectsInvalidKnownFormat(t *testing.T) {
	m := newLoadedModel(t, []ghclient.PropertyValue{{Name: "owner", Value: "b.jones1@elsevier.com"}}, nil, ghclient.ErrSchemaUnavailable)

	next, _ := m.Update(runeKey('e'))
	m = next.(*singleRepoModel)

	m.editor.stringInput.SetValue("not-an-email")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*singleRepoModel)
	if m.screen != screenEdit || m.editor == nil || m.editor.validationErr == nil {
		t.Fatalf("after enter with invalid email: screen=%v editor=%v, want still screenEdit with a validation error", m.screen, m.editor)
	}
	if m.properties[0].Value != "b.jones1@elsevier.com" {
		t.Errorf("value changed despite invalid input: %v", m.properties[0].Value)
	}

	m.editor.stringInput.SetValue("new.owner@elsevier.com")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*singleRepoModel)
	if m.screen != screenList || m.editor != nil {
		t.Fatalf("after enter with a valid email: screen=%v editor=%v, want screenList with no editor", m.screen, m.editor)
	}
	if m.properties[0].Value != "new.owner@elsevier.com" {
		t.Errorf("properties[0].Value = %v, want the new valid email", m.properties[0].Value)
	}
}

func TestSingleRepoModel_SchemaSortedAlphabetically(t *testing.T) {
	schema := []ghclient.PropertyDefinition{
		{Name: "Zebra", Type: ghclient.PropertyTypeString},
		{Name: "Apple", Type: ghclient.PropertyTypeSingleSelect, AllowedValues: []string{"Charlie", "Alpha", "Bravo"}},
		{Name: "Mango", Type: ghclient.PropertyTypeString},
	}
	m := newLoadedModel(t, nil, schema, nil)

	if got := []string{m.schema[0].Name, m.schema[1].Name, m.schema[2].Name}; got[0] != "Apple" || got[1] != "Mango" || got[2] != "Zebra" {
		t.Fatalf("m.schema names = %v, want alphabetical order", got)
	}
	if got := m.schema[0].AllowedValues; got[0] != "Alpha" || got[1] != "Bravo" || got[2] != "Charlie" {
		t.Errorf("Apple.AllowedValues = %v, want alphabetical order", got)
	}

	// The add-property name picker should list candidates in that same order.
	next, _ := m.Update(runeKey('a'))
	m = next.(*singleRepoModel)
	if got := m.editor.namePicker.options; got[0] != "Apple" || got[1] != "Mango" || got[2] != "Zebra" {
		t.Errorf("add-editor namePicker.options = %v, want alphabetical order", got)
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
