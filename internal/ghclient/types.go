package ghclient

import "errors"

// ErrSchemaUnavailable is returned by GetOrgSchema when the token lacks
// permission to read the org's custom-property schema (403/404). Callers
// should fall back to freeform string editing rather than treat this as fatal.
var ErrSchemaUnavailable = errors.New("org custom-property schema unavailable")

// PropertyType is the kind of value a custom property accepts.
type PropertyType string

const (
	PropertyTypeString       PropertyType = "string"
	PropertyTypeSingleSelect PropertyType = "single_select"
	PropertyTypeMultiSelect  PropertyType = "multi_select"
	PropertyTypeTrueFalse    PropertyType = "true_false"
)

// PropertyDefinition describes one property in an org's custom-property schema.
type PropertyDefinition struct {
	Name          string
	Type          PropertyType
	Required      bool
	DefaultValue  any
	AllowedValues []string
}

// PropertyValue is a single property name/value pair on a repository.
// Value is one of string, []string, or nil (nil clears the property). The
// GitHub API represents true_false values as the strings "true"/"false"
// rather than a JSON boolean, so callers editing a true_false property
// should use those string values.
type PropertyValue struct {
	Name  string
	Value any
}

// RepoProperties bundles a repo's identity with its current property values,
// used when fetching/backing up many repos at once in batch mode.
type RepoProperties struct {
	Owner      string
	Repo       string
	Properties []PropertyValue
	Err        error // non-nil if the fetch for this repo failed
}
