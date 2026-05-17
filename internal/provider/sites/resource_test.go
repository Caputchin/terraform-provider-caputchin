package sites

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

func TestSiteCreate_ReturnsSecret(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/management/sites" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]string
		_ = json.Unmarshal(body, &in)
		if in["name"] != "blog-prod" {
			t.Errorf("expected name=blog-prod, got %q", in["name"])
		}
		if in["team_id"] != "team_abc" {
			t.Errorf("expected team_id=team_abc, got %q", in["team_id"])
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"site": map[string]any{
				"id":         "site_xyz",
				"key":        "cpt_pub_abc123",
				"name":       "blog-prod",
				"team_id":    "team_abc",
				"tier":       "team",
				"disabled":   false,
				"created_at": int64(1747500000000),
			},
			"secret": "cpt_sec_supersecret",
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env siteEnvelope
	if err := c.Post(context.Background(), "/v1/management/sites", map[string]string{
		"name":    "blog-prod",
		"team_id": "team_abc",
	}, &env); err != nil {
		t.Fatalf("post failed: %v", err)
	}
	if env.Site.Key != "cpt_pub_abc123" {
		t.Errorf("expected key=cpt_pub_abc123, got %q", env.Site.Key)
	}
	if env.Secret != "cpt_sec_supersecret" {
		t.Errorf("expected secret to round-trip, got %q", env.Secret)
	}
}

func TestSiteUpdate_OnlySendsChangedFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/management/sites/site_xyz") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]any
		_ = json.Unmarshal(body, &in)
		if _, ok := in["team_id"]; ok {
			t.Errorf("Update body must NOT include team_id (immutable)")
		}
		if _, ok := in["name"]; !ok {
			t.Errorf("expected name in update body")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"site": map[string]any{
				"id":         "site_xyz",
				"key":        "cpt_pub_abc123",
				"name":       "blog-staging",
				"team_id":    "team_abc",
				"tier":       "team",
				"disabled":   false,
				"created_at": int64(1747500000000),
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env siteEnvelope
	if err := c.Patch(context.Background(), "/v1/management/sites/site_xyz", map[string]any{"name": "blog-staging"}, &env); err != nil {
		t.Fatalf("patch failed: %v", err)
	}
	if env.Site.Name != "blog-staging" {
		t.Errorf("expected name=blog-staging, got %q", env.Site.Name)
	}
}

func TestSiteRead_404ReturnsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "not-found"})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env siteEnvelope
	err := c.Get(context.Background(), "/v1/management/sites/site_gone", &env)
	if !client.IsNotFound(err) {
		t.Fatalf("expected IsNotFound, got %v", err)
	}
}

func TestSiteDelete_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	if err := c.Delete(context.Background(), "/v1/management/sites/site_xyz"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestApiSite_ToModel_PreservesSecret(t *testing.T) {
	src := apiSite{
		ID: "site_x", Key: "cpt_pub_x", Name: "n", TeamID: "team_x",
		Tier: "team", Disabled: false, CreatedAt: 1747500000000,
	}
	got := src.toModel(types.StringValue("cpt_sec_preserved"))
	if got.Secret.ValueString() != "cpt_sec_preserved" {
		t.Errorf("expected secret preserved, got %q", got.Secret.ValueString())
	}
	if got.Disabled.ValueBool() != false {
		t.Errorf("expected disabled=false, got true")
	}
}

func TestApiSite_ToModel_NullSecret(t *testing.T) {
	// Import path: secret unknown after import; toModel(types.StringNull())
	// produces a null state attr — Read then refreshes other fields without
	// touching the secret.
	src := apiSite{ID: "site_x"}
	got := src.toModel(types.StringNull())
	if !got.Secret.IsNull() {
		t.Errorf("expected null secret, got %v", got.Secret)
	}
}
