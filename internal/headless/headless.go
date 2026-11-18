// Package headless implements the non-interactive --list and --set modes of
// the CLI: print a repo's custom properties, and/or update some of them,
// without starting the TUI.
package headless

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/backup"
	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
)

// Assignment is one parsed --set name=value pair. Clear is true when the
// value was empty (e.g. "team="), meaning the property should be removed.
type Assignment struct {
	Name  string
	Value string
	Clear bool
}

// ParseAssignment parses "name=value". The name is everything before the
// first '='; the value may itself contain '='. An empty value clears the
// property.
func ParseAssignment(s string) (Assignment, error) {
	name, value, ok := strings.Cut(s, "=")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return Assignment{}, fmt.Errorf("invalid --set %q: expected property-name=value", s)
	}
	return Assignment{Name: name, Value: value, Clear: value == ""}, nil
}

// Options configures Run.
type Options struct {
	Owner, Repo string
	List        bool
	Set         []Assignment
	BackupDir   string
}

// Run applies Set (if any), then prints the repo's properties when List is
// set. Output goes to out; informational messages (backup path, changes) go
// to errOut so that --list output stays clean for scripting.
func Run(ctx context.Context, api ghclient.PropertiesAPI, opts Options, out, errOut io.Writer) error {
	if len(opts.Set) > 0 {
		if err := applySet(ctx, api, opts, errOut); err != nil {
			return err
		}
	}
	if opts.List {
		props, err := api.GetRepoProperties(ctx, opts.Owner, opts.Repo)
		if err != nil {
			return fmt.Errorf("fetch properties: %w", err)
		}
		writeList(out, props)
	}
	return nil
}

func applySet(ctx context.Context, api ghclient.PropertiesAPI, opts Options, errOut io.Writer) error {
	schema, err := api.GetOrgSchema(ctx, opts.Owner)
	if err != nil && !errors.Is(err, ghclient.ErrSchemaUnavailable) {
		return fmt.Errorf("fetch org schema: %w", err)
	}
	defs := make(map[string]ghclient.PropertyDefinition, len(schema))
	for _, d := range schema {
		defs[d.Name] = d
	}

	updates := make([]ghclient.PropertyValue, 0, len(opts.Set))
	seen := map[string]bool{}
	for _, a := range opts.Set {
		if seen[a.Name] {
			return fmt.Errorf("property %q given more than once", a.Name)
		}
		seen[a.Name] = true
		pv, err := toPropertyValue(a, defs, schema != nil)
		if err != nil {
			return err
		}
		updates = append(updates, pv)
	}

	before, err := api.GetRepoProperties(ctx, opts.Owner, opts.Repo)
	if err != nil {
		return fmt.Errorf("fetch properties: %w", err)
	}
	path, err := backup.Write(opts.BackupDir, backup.ModeSingle, []backup.RepoSnapshot{
		{Owner: opts.Owner, Repo: opts.Repo, PropertiesBefore: before},
	})
	if err != nil {
		return fmt.Errorf("write backup: %w", err)
	}
	if err := api.SetRepoProperties(ctx, opts.Owner, opts.Repo, updates); err != nil {
		return fmt.Errorf("update properties (backup at %s): %w", path, err)
	}
	for _, u := range updates {
		_, _ = fmt.Fprintf(errOut, "set %s=%s\n", u.Name, formatValue(u.Value))
	}
	_, _ = fmt.Fprintf(errOut, "backup of prior values: %s\n", path)
	return nil
}

// toPropertyValue converts an assignment into the API value shape, validating
// it against the org schema when one is available (haveSchema).
func toPropertyValue(a Assignment, defs map[string]ghclient.PropertyDefinition, haveSchema bool) (ghclient.PropertyValue, error) {
	def, known := defs[a.Name]
	if haveSchema && !known {
		return ghclient.PropertyValue{}, fmt.Errorf("unknown property %q (not defined in the org's schema)", a.Name)
	}
	if a.Clear {
		if known && def.Required {
			return ghclient.PropertyValue{}, fmt.Errorf("property %q is required and cannot be cleared", a.Name)
		}
		return ghclient.PropertyValue{Name: a.Name, Value: nil}, nil
	}
	if !known {
		return ghclient.PropertyValue{Name: a.Name, Value: a.Value}, nil
	}

	switch def.Type {
	case ghclient.PropertyTypeMultiSelect:
		var items []string
		for _, part := range strings.Split(a.Value, ",") {
			if part = strings.TrimSpace(part); part != "" {
				items = append(items, part)
			}
		}
		for _, it := range items {
			if err := checkAllowed(a.Name, it, def.AllowedValues); err != nil {
				return ghclient.PropertyValue{}, err
			}
		}
		return ghclient.PropertyValue{Name: a.Name, Value: items}, nil
	case ghclient.PropertyTypeSingleSelect:
		if err := checkAllowed(a.Name, a.Value, def.AllowedValues); err != nil {
			return ghclient.PropertyValue{}, err
		}
	case ghclient.PropertyTypeTrueFalse:
		v := strings.ToLower(a.Value)
		if v != "true" && v != "false" {
			return ghclient.PropertyValue{}, fmt.Errorf("property %q expects true or false, got %q", a.Name, a.Value)
		}
		return ghclient.PropertyValue{Name: a.Name, Value: v}, nil
	}
	return ghclient.PropertyValue{Name: a.Name, Value: a.Value}, nil
}

func checkAllowed(name, value string, allowed []string) error {
	if len(allowed) == 0 {
		return nil
	}
	for _, a := range allowed {
		if a == value {
			return nil
		}
	}
	return fmt.Errorf("property %q: %q is not one of: %s", name, value, strings.Join(allowed, ", "))
}

// writeList prints one name=value line per property, sorted by name.
// Multi-select values are comma-joined.
func writeList(w io.Writer, props []ghclient.PropertyValue) {
	sorted := append([]ghclient.PropertyValue{}, props...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	for _, p := range sorted {
		_, _ = fmt.Fprintf(w, "%s=%s\n", p.Name, formatValue(p.Value))
	}
}

func formatValue(v any) string {
	switch v := v.(type) {
	case nil:
		return ""
	case string:
		return v
	case []string:
		return strings.Join(v, ",")
	default:
		return fmt.Sprint(v)
	}
}
