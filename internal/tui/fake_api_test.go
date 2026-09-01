package tui

import (
	"context"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
)

// fakeAPI is a hand-rolled ghclient.PropertiesAPI for tests, avoiding any
// network dependency.
type fakeAPI struct {
	properties []ghclient.PropertyValue
	schema     []ghclient.PropertyDefinition
	schemaErr  error

	setCalls [][]ghclient.PropertyValue
	setErr   error
}

func (f *fakeAPI) GetRepoProperties(ctx context.Context, owner, repo string) ([]ghclient.PropertyValue, error) {
	return append([]ghclient.PropertyValue{}, f.properties...), nil
}

func (f *fakeAPI) SetRepoProperties(ctx context.Context, owner, repo string, properties []ghclient.PropertyValue) error {
	f.setCalls = append(f.setCalls, properties)
	return f.setErr
}

func (f *fakeAPI) GetOrgSchema(ctx context.Context, org string) ([]ghclient.PropertyDefinition, error) {
	if f.schemaErr != nil {
		return nil, f.schemaErr
	}
	return append([]ghclient.PropertyDefinition{}, f.schema...), nil
}

var _ ghclient.PropertiesAPI = (*fakeAPI)(nil)
