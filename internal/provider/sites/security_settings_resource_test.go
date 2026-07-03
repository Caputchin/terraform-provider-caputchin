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
			"settings": map[string]any{"require_game": true, "preview_mode": true},
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
	if env.Settings.PreviewMode == nil || !*env.Settings.PreviewMode {
		t.Errorf("expected preview_mode=true, got %v", env.Settings.PreviewMode)
	}
	m := env.Settings.toModel("site_xyz")
	if !m.RequireGame.ValueBool() || !m.PreviewMode.ValueBool() || m.SiteID.ValueString() != "site_xyz" {
		t.Errorf("toModel mismatch: %+v", m)
	}
}

// TestSiteSecurityGet_PreviewModeNull covers the nullable tri-state: null on
// the wire means "inherit the troop default" and decodes as BoolNull, distinct
// from an explicit false.
func TestSiteSecurityGet_PreviewModeNull(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"site_id":  "site_xyz",
			"settings": map[string]any{"preview_mode": nil},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env siteSecuritySettingsEnvelope
	if err := c.Get(context.Background(), siteSecurityPath("site_xyz"), &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if env.Settings.PreviewMode != nil {
		t.Errorf("expected preview_mode nil on the wire, got %v", *env.Settings.PreviewMode)
	}
	m := env.Settings.toModel("site_xyz")
	if !m.PreviewMode.IsNull() {
		t.Errorf("expected preview_mode null (inherit troop default), got %v", m.PreviewMode)
	}
}

func TestSiteSecurityGet_DecodesReuse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"site_id":  "site_xyz",
			"settings": map[string]any{"reuse": true, "reuse_window_ms": 60000, "reuse_persist": true},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env siteSecuritySettingsEnvelope
	if err := c.Get(context.Background(), siteSecurityPath("site_xyz"), &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !env.Settings.Reuse {
		t.Errorf("expected reuse=true, got %v", env.Settings.Reuse)
	}
	if env.Settings.ReuseWindowMs == nil || *env.Settings.ReuseWindowMs != 60000 {
		t.Errorf("expected reuse_window_ms=60000, got %v", env.Settings.ReuseWindowMs)
	}
	if !env.Settings.ReusePersist {
		t.Errorf("expected reuse_persist=true, got %v", env.Settings.ReusePersist)
	}
	m := env.Settings.toModel("site_xyz")
	if !m.Reuse.ValueBool() || m.ReuseWindowMs.ValueInt64() != 60000 || !m.ReusePersist.ValueBool() {
		t.Errorf("toModel mismatch: %+v", m)
	}
}

// TestSiteSecurityGet_ReuseWindowMsNull covers the nullable case: absent on
// the wire falls back to the server's default window, decoded as Int64Null.
func TestSiteSecurityGet_ReuseWindowMsNull(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"site_id":  "site_xyz",
			"settings": map[string]any{"reuse": false, "reuse_window_ms": nil, "reuse_persist": false},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env siteSecuritySettingsEnvelope
	if err := c.Get(context.Background(), siteSecurityPath("site_xyz"), &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	m := env.Settings.toModel("site_xyz")
	if !m.ReuseWindowMs.IsNull() {
		t.Errorf("expected reuse_window_ms null, got %v", m.ReuseWindowMs)
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

func TestSiteSecurityPatch_PreviewModeOnlyChangedFields(t *testing.T) {
	r := &siteSecuritySettingsResource{}

	// plan true vs state false → preview_mode in body, require_game untouched.
	body := r.buildPatchBody(
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), RequireGame: types.BoolValue(false), PreviewMode: types.BoolValue(true)},
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), RequireGame: types.BoolValue(false), PreviewMode: types.BoolValue(false)},
	)
	if v, ok := body["preview_mode"].(bool); !ok || !v {
		t.Errorf("expected preview_mode=true in body, got %v", body)
	}
	if _, ok := body["require_game"]; ok {
		t.Errorf("expected require_game absent from body, got %v", body)
	}

	// unchanged → empty body.
	if b := r.buildPatchBody(
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), PreviewMode: types.BoolValue(true)},
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), PreviewMode: types.BoolValue(true)},
	); len(b) != 0 {
		t.Errorf("expected empty body when unchanged, got %v", b)
	}

	// plan explicitly null preview_mode vs a set state → clears the field
	// (nil in body, key present) so the site reverts to inheriting the troop
	// default, distinct from an explicit false.
	if b := r.buildPatchBody(
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), PreviewMode: types.BoolNull()},
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), PreviewMode: types.BoolValue(true)},
	); b["preview_mode"] != nil {
		t.Errorf("expected preview_mode=nil in body when cleared, got %v", b)
	} else if _, ok := b["preview_mode"]; !ok {
		t.Errorf("expected preview_mode key present (explicit null), got %v", b)
	}

	// Unknown plan (Create with no value set) → empty body.
	if b := r.buildPatchBody(
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), PreviewMode: types.BoolUnknown()},
		siteSecuritySettingsModel{},
	); len(b) != 0 {
		t.Errorf("expected empty body when plan unknown, got %v", b)
	}
}

func TestSiteSecurityPatch_ReuseOnlyChangedFields(t *testing.T) {
	r := &siteSecuritySettingsResource{}

	// plan true vs state false → reuse + reuse_window_ms + reuse_persist in body.
	body := r.buildPatchBody(
		siteSecuritySettingsModel{
			SiteID: types.StringValue("s"), Reuse: types.BoolValue(true),
			ReuseWindowMs: types.Int64Value(60000), ReusePersist: types.BoolValue(true),
		},
		siteSecuritySettingsModel{
			SiteID: types.StringValue("s"), Reuse: types.BoolValue(false),
			ReuseWindowMs: types.Int64Value(30000), ReusePersist: types.BoolValue(false),
		},
	)
	if v, ok := body["reuse"].(bool); !ok || !v {
		t.Errorf("expected reuse=true in body, got %v", body)
	}
	if v, ok := body["reuse_window_ms"].(int64); !ok || v != 60000 {
		t.Errorf("expected reuse_window_ms=60000 in body, got %v", body)
	}
	if v, ok := body["reuse_persist"].(bool); !ok || !v {
		t.Errorf("expected reuse_persist=true in body, got %v", body)
	}

	// unchanged → empty body.
	if b := r.buildPatchBody(
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), Reuse: types.BoolValue(true)},
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), Reuse: types.BoolValue(true)},
	); len(b) != 0 {
		t.Errorf("expected empty body when unchanged, got %v", b)
	}

	// plan explicitly null reuse_window_ms vs a set state → clears the field (nil in body).
	if b := r.buildPatchBody(
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), ReuseWindowMs: types.Int64Null()},
		siteSecuritySettingsModel{SiteID: types.StringValue("s"), ReuseWindowMs: types.Int64Value(30000)},
	); b["reuse_window_ms"] != nil {
		t.Errorf("expected reuse_window_ms=nil in body when cleared, got %v", b)
	} else if _, ok := b["reuse_window_ms"]; !ok {
		t.Errorf("expected reuse_window_ms key present (explicit null), got %v", b)
	}

	// Unknown plan (Create with no value set) → empty body.
	if b := r.buildPatchBody(
		siteSecuritySettingsModel{
			SiteID: types.StringValue("s"), Reuse: types.BoolUnknown(),
			ReuseWindowMs: types.Int64Unknown(), ReusePersist: types.BoolUnknown(),
		},
		siteSecuritySettingsModel{},
	); len(b) != 0 {
		t.Errorf("expected empty body when plan unknown, got %v", b)
	}
}
