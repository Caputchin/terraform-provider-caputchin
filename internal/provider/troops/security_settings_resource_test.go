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

func TestTroopSecurityGet_DecodesForbidReuse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"troop_id": "troop_xyz",
			"settings": map[string]any{"forbid_reuse": true},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env troopSecuritySettingsEnvelope
	if err := c.Get(context.Background(), troopSecurityPath("troop_xyz"), &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !env.Settings.ForbidReuse {
		t.Errorf("expected forbid_reuse=true, got %v", env.Settings.ForbidReuse)
	}
	m := env.Settings.toModel("troop_xyz")
	if !m.ForbidReuse.ValueBool() {
		t.Errorf("toModel mismatch: %+v", m)
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

func TestTroopSecurityPatch_ForbidReuseOnlyChangedFields(t *testing.T) {
	r := &troopSecuritySettingsResource{}

	// plan true vs state false → forbid_reuse in body, other fields untouched.
	body := r.buildPatchBody(
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForceGame: types.BoolValue(false), ForbidReuse: types.BoolValue(true)},
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForceGame: types.BoolValue(false), ForbidReuse: types.BoolValue(false)},
	)
	if v, ok := body["forbid_reuse"].(bool); !ok || !v {
		t.Errorf("expected forbid_reuse=true in body, got %v", body)
	}
	if _, ok := body["force_game"]; ok {
		t.Errorf("expected force_game absent from body, got %v", body)
	}

	// unchanged → empty body.
	if b := r.buildPatchBody(
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForbidReuse: types.BoolValue(true)},
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForbidReuse: types.BoolValue(true)},
	); len(b) != 0 {
		t.Errorf("expected empty body when unchanged, got %v", b)
	}

	// Unknown plan → empty body.
	if b := r.buildPatchBody(
		troopSecuritySettingsModel{TroopID: types.StringValue("t"), ForbidReuse: types.BoolUnknown()},
		troopSecuritySettingsModel{},
	); len(b) != 0 {
		t.Errorf("expected empty body when plan unknown, got %v", b)
	}
}
