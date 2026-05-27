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
			"settings": map[string]any{"force_game": true},
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
	m := env.Settings.toModel("troop_xyz")
	if !m.ForceGame.ValueBool() || m.TroopID.ValueString() != "troop_xyz" {
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
