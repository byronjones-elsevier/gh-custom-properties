package ghclient

import "context"

// PropertiesAPI is the subset of Client used by the TUI, kept as an interface
// so tests can supply a hand-rolled fake instead of hitting the network.
type PropertiesAPI interface {
	GetRepoProperties(ctx context.Context, owner, repo string) ([]PropertyValue, error)
	SetRepoProperties(ctx context.Context, owner, repo string, properties []PropertyValue) error
	GetOrgSchema(ctx context.Context, org string) ([]PropertyDefinition, error)
}

var _ PropertiesAPI = (*Client)(nil)
