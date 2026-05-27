// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

func TestSiteSecurityGet_DecodesRequireGame(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/security-settings") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"site_id":  "site_xyz",
			"settings": map[string]any{"require_game": true},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env siteSecuritySettingsEnvelope
	if err := c.Get(context.Background(), siteSecurityPath("site_xyz"), &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !env.Settings.RequireGame {
		t.Errorf("expected require_game=true, got %v", env.Settings.RequireGame)
	}
	m := env.Settings.toModel("site_xyz")
	if !m.RequireGame.ValueBool() || m.SiteID.ValueString() != "site_xyz" {
		t.Errorf("toModel mismatch: %+v", m)
	}
}

func TestSiteSecurityPatch_OnlyChangedFields(t *testing.T) {
	r := &siteSecuritySettingsResource{}

	// plan true vs state false → require_game in body.
	body := r.buildPatchBody(
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), RequireGame: types.BoolValue(true)},
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), RequireGame: types.BoolValue(false)},
	)
	if v, ok := body["require_game"].(bool); !ok || !v {
		t.Errorf("expected require_game=true in body, got %v", body)
	}

	// unchanged → empty body.
	if b := r.buildPatchBody(
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), RequireGame: types.BoolValue(true)},
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), RequireGame: types.BoolValue(true)},
	); len(b) != 0 {
		t.Errorf("expected empty body when unchanged, got %v", b)
	}

	// Unknown plan (Create with no value set) → empty body (no PATCH, just read).
	if b := r.buildPatchBody(
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), RequireGame: types.BoolUnknown()},
		siteSecuritySettingsModel{},
	); len(b) != 0 {
		t.Errorf("expected empty body when plan unknown, got %v", b)
	}
}
