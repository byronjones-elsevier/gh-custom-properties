package tui

import (
	"sort"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
)

// sortSchema sorts property definitions by name, and each definition's
// allowed values alphabetically, in place. Live org-schema data isn't
// guaranteed to come back from GitHub in any particular order; this makes
// the name pickers and value dropdowns built from it predictable regardless
// of how the API happened to return them.
func sortSchema(defs []ghclient.PropertyDefinition) {
	sort.Slice(defs, func(i, j int) bool { return defs[i].Name < defs[j].Name })
	for i := range defs {
		sort.Strings(defs[i].AllowedValues)
	}
}
