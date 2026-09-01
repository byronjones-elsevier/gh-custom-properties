package ghclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// wirePropertyValue mirrors the JSON shape of one element returned by, and
// accepted by, /repos/{owner}/{repo}/properties/values.
type wirePropertyValue struct {
	PropertyName string `json:"property_name"`
	Value        any    `json:"value"`
}

// wirePropertyDefinition mirrors one element returned by
// /orgs/{org}/properties/schema.
type wirePropertyDefinition struct {
	PropertyName  string   `json:"property_name"`
	ValueType     string   `json:"value_type"`
	Required      bool     `json:"required"`
	DefaultValue  any      `json:"default_value"`
	AllowedValues []string `json:"allowed_values"`
}

// GetRepoProperties fetches the current custom-property values set on a repo.
func (c *Client) GetRepoProperties(ctx context.Context, owner, repo string) ([]PropertyValue, error) {
	path := fmt.Sprintf("/repos/%s/%s/properties/values", owner, repo)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GET %s: read response: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &apiError{StatusCode: resp.StatusCode, Method: http.MethodGet, Path: path, Body: string(body)}
	}

	var wire []wirePropertyValue
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("GET %s: decode response: %w", path, err)
	}

	values := make([]PropertyValue, len(wire))
	for i, w := range wire {
		values[i] = PropertyValue{Name: w.PropertyName, Value: w.Value}
	}
	return values, nil
}

// SetRepoProperties upserts the given property values on a repo. A
// PropertyValue with a nil Value clears that property from the repo.
func (c *Client) SetRepoProperties(ctx context.Context, owner, repo string, properties []PropertyValue) error {
	path := fmt.Sprintf("/repos/%s/%s/properties/values", owner, repo)

	wire := make([]wirePropertyValue, len(properties))
	for i, p := range properties {
		wire[i] = wirePropertyValue{PropertyName: p.Name, Value: p.Value}
	}
	payload, err := json.Marshal(struct {
		Properties []wirePropertyValue `json:"properties"`
	}{Properties: wire})
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPatch, path, payload)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("PATCH %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return &apiError{StatusCode: resp.StatusCode, Method: http.MethodPatch, Path: path, Body: string(body)}
	}
	return nil
}

// GetOrgSchema fetches the custom-property schema defined for an org. If the
// token lacks permission to read it (403/404), it returns ErrSchemaUnavailable
// so callers can fall back to freeform editing instead of failing outright.
func (c *Client) GetOrgSchema(ctx context.Context, org string) ([]PropertyDefinition, error) {
	path := fmt.Sprintf("/orgs/%s/properties/schema", org)
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("GET %s: read response: %w", path, err)
	}

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusNotFound {
		return nil, ErrSchemaUnavailable
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &apiError{StatusCode: resp.StatusCode, Method: http.MethodGet, Path: path, Body: string(body)}
	}

	var wire []wirePropertyDefinition
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("GET %s: decode response: %w", path, err)
	}

	defs := make([]PropertyDefinition, len(wire))
	for i, w := range wire {
		defs[i] = PropertyDefinition{
			Name:          w.PropertyName,
			Type:          PropertyType(w.ValueType),
			Required:      w.Required,
			DefaultValue:  w.DefaultValue,
			AllowedValues: w.AllowedValues,
		}
	}
	return defs, nil
}
