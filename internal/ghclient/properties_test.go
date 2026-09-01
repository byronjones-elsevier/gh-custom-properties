package ghclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &Client{httpClient: srv.Client(), baseURL: srv.URL, token: "test-token"}
}

func TestGetRepoProperties(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/repos/octocat/hello-world/properties/values"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"property_name":"team","value":"platform"},{"property_name":"tags","value":["a","b"]},{"property_name":"unset","value":null}]`))
	})

	got, err := c.GetRepoProperties(context.Background(), "octocat", "hello-world")
	if err != nil {
		t.Fatalf("GetRepoProperties: %v", err)
	}
	want := []PropertyValue{
		{Name: "team", Value: "platform"},
		{Name: "tags", Value: []any{"a", "b"}},
		{Name: "unset", Value: nil},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d properties, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Name != want[i].Name {
			t.Errorf("property[%d].Name = %q, want %q", i, got[i].Name, want[i].Name)
		}
	}
}

func TestGetRepoProperties_ErrorStatus(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	})

	_, err := c.GetRepoProperties(context.Background(), "octocat", "missing")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestSetRepoProperties(t *testing.T) {
	var gotBody map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %q, want PATCH", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	err := c.SetRepoProperties(context.Background(), "octocat", "hello-world", []PropertyValue{
		{Name: "team", Value: "platform"},
		{Name: "cleared", Value: nil},
	})
	if err != nil {
		t.Fatalf("SetRepoProperties: %v", err)
	}

	props, ok := gotBody["properties"].([]any)
	if !ok || len(props) != 2 {
		t.Fatalf("request body properties = %#v", gotBody["properties"])
	}
}

func TestGetOrgSchema(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.URL.Path, "/orgs/acme/properties/schema"; got != want {
			t.Errorf("path = %q, want %q", got, want)
		}
		_, _ = w.Write([]byte(`[{"property_name":"tier","value_type":"single_select","required":true,"allowed_values":["1","2","3"]}]`))
	})

	defs, err := c.GetOrgSchema(context.Background(), "acme")
	if err != nil {
		t.Fatalf("GetOrgSchema: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("got %d definitions, want 1", len(defs))
	}
	d := defs[0]
	if d.Name != "tier" || d.Type != PropertyTypeSingleSelect || !d.Required || len(d.AllowedValues) != 3 {
		t.Errorf("unexpected definition: %+v", d)
	}
}

func TestGetOrgSchema_Unavailable(t *testing.T) {
	for _, status := range []int{http.StatusForbidden, http.StatusNotFound} {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
		})
		_, err := c.GetOrgSchema(context.Background(), "acme")
		if err != ErrSchemaUnavailable {
			t.Errorf("status %d: err = %v, want ErrSchemaUnavailable", status, err)
		}
	}
}
