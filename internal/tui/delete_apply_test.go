package tui

import (
	"reflect"
	"testing"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
)

func TestSingleRepoModel_ApplySendsClearForDeletedProperty(t *testing.T) {
	api := &fakeAPI{}
	m := newSingleRepoModel(api, t.TempDir(), "o", "r")
	next, _ := m.handleLoaded(loadedMsg{owner: "o", repo: "r", properties: []ghclient.PropertyValue{
		{Name: "CostCode", Value: "C1"},
		{Name: "SystemID", Value: "S1"},
	}})
	m = next.(*singleRepoModel)

	// Equivalent of selecting SystemID, pressing d, then y.
	m.properties = m.properties[:1]
	if msg := m.applyCmd()(); msg.(appliedMsg).err != nil {
		t.Fatalf("apply failed: %v", msg.(appliedMsg).err)
	}

	want := []ghclient.PropertyValue{{Name: "CostCode", Value: "C1"}, {Name: "SystemID", Value: nil}}
	if len(api.setCalls) != 1 || !reflect.DeepEqual(api.setCalls[0], want) {
		t.Errorf("setCalls = %#v, want one call with %#v", api.setCalls, want)
	}
}
