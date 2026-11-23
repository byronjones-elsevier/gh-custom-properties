package tui

import (
	"testing"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
	tea "github.com/charmbracelet/bubbletea"
)

func TestValueEditor_BlankUnsetsOptionalProperties(t *testing.T) {
	tests := []struct {
		name      string
		prop      string
		input     string
		wantDone  bool
		wantValue any
	}{
		{"SystemID blank unsets", "SystemID", "", true, nil},
		{"CostCode blank unsets", "CostCode", "", true, nil},
		{"SystemName blank unsets", "SystemName", "", true, nil},
		{"value kept when present", "SystemID", "ABC123", true, "ABC123"},
		{"invalid still rejected", "SystemID", "bad id", false, nil},
		{"owner blank still rejected", "owner", "", false, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEditEditor(tt.prop, nil, "previous", 20)
			e.stringInput.SetValue(tt.input)

			_, outcome := e.updateValueStep(tea.KeyMsg{Type: tea.KeyEnter})
			if done := outcome == outcomeDone; done != tt.wantDone {
				t.Fatalf("done = %v, want %v (err: %v)", done, tt.wantDone, e.validationErr)
			}
			if tt.wantDone {
				if got := e.Result().Value; got != tt.wantValue {
					t.Errorf("Result value = %#v, want %#v", got, tt.wantValue)
				}
			}
		})
	}
}

func TestSingleRepoModel_BlankEditStaysListedAsUnset(t *testing.T) {
	m := newLoadedModel(t, nil, nil, nil)
	m.properties = append(m.properties, ghclient.PropertyValue{Name: "SystemID", Value: "OLD"})

	next, _ := m.Update(runeKey('e'))
	m = settle(next.(*singleRepoModel))
	m.editor.stringInput.SetValue("")
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(*singleRepoModel)

	if len(m.properties) != 1 || m.properties[0].Value != nil {
		t.Errorf("properties = %#v, want SystemID kept with a nil value so apply sends a clear", m.properties)
	}
}
