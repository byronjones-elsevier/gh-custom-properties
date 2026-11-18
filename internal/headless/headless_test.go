package headless

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/ByronJones-Elsevier/gh-custom-properties/internal/ghclient"
)

type fakeAPI struct {
	props     []ghclient.PropertyValue
	schema    []ghclient.PropertyDefinition
	schemaErr error
	setCalls  [][]ghclient.PropertyValue
}

func (f *fakeAPI) GetRepoProperties(context.Context, string, string) ([]ghclient.PropertyValue, error) {
	return f.props, nil
}
func (f *fakeAPI) SetRepoProperties(_ context.Context, _, _ string, p []ghclient.PropertyValue) error {
	f.setCalls = append(f.setCalls, p)
	return nil
}
func (f *fakeAPI) GetOrgSchema(context.Context, string) ([]ghclient.PropertyDefinition, error) {
	return f.schema, f.schemaErr
}

func TestParseAssignment(t *testing.T) {
	tests := []struct {
		in      string
		want    Assignment
		wantErr bool
	}{
		{"team=core", Assignment{Name: "team", Value: "core"}, false},
		{"url=a=b", Assignment{Name: "url", Value: "a=b"}, false},
		{"team=", Assignment{Name: "team", Clear: true}, false},
		{"=x", Assignment{}, true},
		{"noequals", Assignment{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseAssignment(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRunList(t *testing.T) {
	api := &fakeAPI{props: []ghclient.PropertyValue{
		{Name: "tags", Value: []string{"a", "b"}},
		{Name: "team", Value: "core"},
	}}
	var out, errOut bytes.Buffer
	if err := Run(context.Background(), api, Options{Owner: "o", Repo: "r", List: true}, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if want := "tags=a,b\nteam=core\n"; out.String() != want {
		t.Errorf("out = %q, want %q", out.String(), want)
	}
	if len(api.setCalls) != 0 {
		t.Error("list must not write")
	}
}

func TestRunSet(t *testing.T) {
	schema := []ghclient.PropertyDefinition{
		{Name: "team", Type: ghclient.PropertyTypeString},
		{Name: "tier", Type: ghclient.PropertyTypeSingleSelect, AllowedValues: []string{"gold", "silver"}},
		{Name: "tags", Type: ghclient.PropertyTypeMultiSelect, AllowedValues: []string{"a", "b", "c"}},
		{Name: "public", Type: ghclient.PropertyTypeTrueFalse},
		{Name: "owner", Type: ghclient.PropertyTypeString, Required: true},
	}
	tests := []struct {
		name      string
		schema    []ghclient.PropertyDefinition
		schemaErr error
		set       []string
		want      []ghclient.PropertyValue
		wantErr   string
	}{
		{
			name: "multiple pairs with types", schema: schema,
			set: []string{"team=core", "tier=gold", "tags=a, c", "public=TRUE"},
			want: []ghclient.PropertyValue{
				{Name: "team", Value: "core"}, {Name: "tier", Value: "gold"},
				{Name: "tags", Value: []string{"a", "c"}}, {Name: "public", Value: "true"},
			},
		},
		{name: "clear", schema: schema, set: []string{"team="}, want: []ghclient.PropertyValue{{Name: "team", Value: nil}}},
		{name: "clear required", schema: schema, set: []string{"owner="}, wantErr: "required"},
		{name: "bad select", schema: schema, set: []string{"tier=bronze"}, wantErr: "not one of"},
		{name: "bad multi", schema: schema, set: []string{"tags=a,z"}, wantErr: "not one of"},
		{name: "bad bool", schema: schema, set: []string{"public=yes"}, wantErr: "true or false"},
		{name: "unknown with schema", schema: schema, set: []string{"nope=x"}, wantErr: "unknown property"},
		{name: "duplicate", schema: schema, set: []string{"team=a", "team=b"}, wantErr: "more than once"},
		{
			name: "schema unavailable is freeform", schemaErr: ghclient.ErrSchemaUnavailable,
			set: []string{"anything=x"}, want: []ghclient.PropertyValue{{Name: "anything", Value: "x"}},
		},
		{name: "schema error is fatal", schemaErr: errors.New("boom"), set: []string{"a=b"}, wantErr: "boom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &fakeAPI{schema: tt.schema, schemaErr: tt.schemaErr, props: []ghclient.PropertyValue{{Name: "team", Value: "old"}}}
			var set []Assignment
			for _, s := range tt.set {
				a, err := ParseAssignment(s)
				if err != nil {
					t.Fatal(err)
				}
				set = append(set, a)
			}
			var out, errOut bytes.Buffer
			err := Run(context.Background(), api, Options{Owner: "o", Repo: "r", Set: set, BackupDir: t.TempDir()}, &out, &errOut)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
				}
				if len(api.setCalls) != 0 {
					t.Error("must not write on validation error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(api.setCalls) != 1 || !reflect.DeepEqual(api.setCalls[0], tt.want) {
				t.Errorf("setCalls = %#v, want %#v", api.setCalls, tt.want)
			}
		})
	}
}
