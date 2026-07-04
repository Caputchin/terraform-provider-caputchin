// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package troops

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

func TestTroopSecurityGet_DecodesForceGame(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/security-settings") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"troop_id": "troop_xyz",
			"settings": map[string]any{"force_game": true, "preview_mode": true},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env troopSecuritySettingsEnvelope
	if err := c.Get(context.Background(), troopSecurityPath("troop_xyz"), &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !env.Settings.ForceGame {
		t.Errorf("expected force_game=true, got %v", env.Settings.ForceGame)
	}
	if !env.Settings.PreviewMode {
		t.Errorf("expected preview_mode=true, got %v", env.Settings.PreviewMode)
	}
	m := env.Settings.toModel("troop_xyz")
	if !m.ForceGame.ValueBool() || !m.PreviewMode.ValueBool() || m.TroopID.ValueString() != "troop_xyz" {
		t.Errorf("toModel mismatch: %+v", m)
	}
}

func TestTroopSecurityGet_DecodesReuse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"troop_id": "troop_xyz",
			"settings": map[string]any{"reuse": true, "reuse_window_ms": 120000, "reuse_persist": true},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env troopSecuritySettingsEnvelope
	if err := c.Get(context.Background(), troopSecurityPath("troop_xyz"), &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if env.Settings.Reuse == nil || !*env.Settings.Reuse {
		t.Errorf("expected reuse=true, got %v", env.Settings.Reuse)
	}
	m := env.Settings.toModel("troop_xyz")
	if !m.Reuse.ValueBool() || m.ReuseWindowMs.ValueInt64() != 120000 || !m.ReusePersist.ValueBool() {
		t.Errorf("toModel mismatch: %+v", m)
	}
}

// A null troop reuse default must stay null in state (no troop default), not
// collapse to false — the default+override nullable contract.
func TestTroopSecurityGet_ReuseNullStaysNull(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"troop_id": "troop_xyz",
			"settings": map[string]any{"reuse": nil, "reuse_window_ms": nil, "reuse_persist": nil},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env troopSecuritySettingsEnvelope
	if err := c.Get(context.Background(), troopSecurityPath("troop_xyz"), &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	m := env.Settings.toModel("troop_xyz")
	if !m.Reuse.IsNull() || !m.ReuseWindowMs.IsNull() || !m.ReusePersist.IsNull() {
		t.Errorf("expected null reuse fields to stay null, got %+v", m)
	}
}

func TestTroopSecurityPatch_OnlyChangedFields(t *testing.T) {
	r := &troopSecuritySettingsResource{}

	// plan true vs state false → force_game in body.
	body := r.buildPatchBody(
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForceGame: types.BoolValue(true)},
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForceGame: types.BoolValue(false)},
	)
	if v, ok := body["force_game"].(bool); !ok || !v {
		t.Errorf("expected force_game=true in body, got %v", body)
	}

	// unchanged → empty body.
	if b := r.buildPatchBody(
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForceGame: types.BoolValue(true)},
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForceGame: types.BoolValue(true)},
	); len(b) != 0 {
		t.Errorf("expected empty body when unchanged, got %v", b)
	}

	// Unknown plan → empty body.
	if b := r.buildPatchBody(
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForceGame: types.BoolUnknown()},
		troopSecuritySettingsModel{},
	); len(b) != 0 {
		t.Errorf("expected empty body when plan unknown, got %v", b)
	}
}

func TestTroopSecurityPatch_PreviewModeOnlyChangedFields(t *testing.T) {
	r := &troopSecuritySettingsResource{}

	// plan true vs state false → preview_mode in body, force_game untouched.
	body := r.buildPatchBody(
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForceGame: types.BoolValue(false), PreviewMode: types.BoolValue(true)},
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForceGame: types.BoolValue(false), PreviewMode: types.BoolValue(false)},
	)
	if v, ok := body["preview_mode"].(bool); !ok || !v {
		t.Errorf("expected preview_mode=true in body, got %v", body)
	}
	if _, ok := body["force_game"]; ok {
		t.Errorf("expected force_game absent from body, got %v", body)
	}

	// unchanged → empty body.
	if b := r.buildPatchBody(
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), PreviewMode: types.BoolValue(true)},
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), PreviewMode: types.BoolValue(true)},
	); len(b) != 0 {
		t.Errorf("expected empty body when unchanged, got %v", b)
	}

	// Unknown plan → empty body.
	if b := r.buildPatchBody(
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), PreviewMode: types.BoolUnknown()},
		troopSecuritySettingsModel{},
	); len(b) != 0 {
		t.Errorf("expected empty body when plan unknown, got %v", b)
	}
}

func TestTroopSecurityPatch_ReuseOnlyChangedFields(t *testing.T) {
	r := &troopSecuritySettingsResource{}

	// plan reuse=true + window vs state false → reuse trio in body, force_game untouched.
	body := r.buildPatchBody(
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForceGame: types.BoolValue(false), Reuse: types.BoolValue(true), ReuseWindowMs: types.Int64Value(120000), ReusePersist: types.BoolValue(true)},
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForceGame: types.BoolValue(false), Reuse: types.BoolValue(false)},
	)
	if v, ok := body["reuse"].(bool); !ok || !v {
		t.Errorf("expected reuse=true in body, got %v", body)
	}
	if v, ok := body["reuse_window_ms"].(int64); !ok || v != 120000 {
		t.Errorf("expected reuse_window_ms=120000 in body, got %v", body)
	}
	if _, ok := body["force_game"]; ok {
		t.Errorf("expected force_game absent from body, got %v", body)
	}

	// null plan vs non-null state → sends explicit null (clear the troop default).
	nb := r.buildPatchBody(
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), Reuse: types.BoolNull()},
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), Reuse: types.BoolValue(true)},
	)
	if v, ok := nb["reuse"]; !ok || v != nil {
		t.Errorf("expected reuse=nil in body when cleared, got %v", nb)
	}

	// Unknown plan → empty body.
	if b := r.buildPatchBody(
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), Reuse: types.BoolUnknown()},
		troopSecuritySettingsModel{},
	); len(b) != 0 {
		t.Errorf("expected empty body when plan unknown, got %v", b)
	}
}
