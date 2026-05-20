// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

func TestSitesList_DecodesEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sites": []map[string]any{
				{
					"id":         "site_a",
					"key":        "cpt_pub_a",
					"name":       "blog",
					"troop_id":   "troop_x",
					"tier":       "troop",
					"disabled":   false,
					"created_at": int64(1747500000000),
				},
				{
					"id":         "site_b",
					"key":        "cpt_pub_b",
					"name":       "marketing",
					"troop_id":   "troop_x",
					"tier":       "troop",
					"disabled":   true,
					"created_at": int64(1747500000001),
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env siteListEnvelope
	if err := c.Get(context.Background(), "/v1/management/sites", &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(env.Sites) != 2 || env.Sites[1].Disabled == nil || *env.Sites[1].Disabled != true {
		t.Errorf("unexpected list shape: %+v", env.Sites)
	}
}

// TestSitesList_NamesOnlyRowDecodes documents the mixed-shape wire
// contract: the platform's GET /sites emits a names-only shape for
// manage-no-read memberships. The list data source's Read filters these
// out, but the decode itself must NOT zero-fill; pointer fields stay
// nil, and the data-source's iterator skip-on-nil keeps lies out of TF
// state.
func TestSitesList_NamesOnlyRowDecodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sites": []map[string]any{
				{
					"id":       "site_namesonly",
					"name":     "blog",
					"troop_id": "troop_x",
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env siteListEnvelope
	if err := c.Get(context.Background(), "/v1/management/sites", &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(env.Sites) != 1 {
		t.Fatalf("expected 1 row, got %d", len(env.Sites))
	}
	if env.Sites[0].Key != nil || env.Sites[0].Tier != nil || env.Sites[0].Disabled != nil || env.Sites[0].CreatedAt != nil {
		t.Errorf("names-only row should decode pointer fields as nil, got %+v", env.Sites[0])
	}
}
