package ghclient

import (
	"errors"
	"fmt"
)

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
	Name          string       `json:"name"`
	Type          PropertyType `json:"type"`
	Required      bool         `json:"required"`
	DefaultValue  any          `json:"default_value,omitempty"`
	AllowedValues []string     `json:"allowed_values,omitempty"`
}

// PropertyValue is a single property name/value pair on a repository.
// Value is one of string, []string, or nil (nil clears the property). The
// GitHub API represents true_false values as the strings "true"/"false"
// rather than a JSON boolean, so callers editing a true_false property
// should use those string values.
type PropertyValue struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
}

// normalizeValue converts the result of unmarshaling a property value from
// JSON (where a multi_select value decodes as []any) into the plain
// string/[]string/nil shapes the rest of the codebase works with.
func normalizeValue(v any) any {
	items, ok := v.([]any)
	if !ok {
		return v
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		} else {
			out = append(out, fmt.Sprintf("%v", item))
		}
	}
	return out
}
