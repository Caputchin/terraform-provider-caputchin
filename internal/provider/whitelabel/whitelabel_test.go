// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package whitelabel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

func ts(s string) types.String { return types.StringValue(s) }

func TestPresetPath(t *testing.T) {
	cases := []struct {
		name   string
		troop  types.String
		site   types.String
		game   types.String
		axis   string
		preset string
		want   string
	}{
		{
			name: "troop widget-shell", troop: ts("troop_1"), site: types.StringNull(), game: types.StringNull(),
			axis: "skin", preset: "midnight",
			want: "/v1/management/troops/troop_1/white-label/skin/preset?name=midnight",
		},
		{
			name: "site widget-shell", troop: types.StringNull(), site: ts("site_1"), game: types.StringNull(),
			axis: "locale", preset: "en",
			want: "/v1/management/sites/site_1/white-label/locale/preset?name=en",
		},
		{
			// Game id contains slashes, so it must be query-encoded, never a path segment.
			name: "troop game-axis with slashy game id", troop: ts("troop_1"), site: types.StringNull(), game: ts("caputchin/games/leaf"),
			axis: "configuration", preset: "hard",
			want: "/v1/management/troops/troop_1/game-customization/configuration/preset?game=caputchin%2Fgames%2Fleaf&name=hard",
		},
		{
			name: "site game-axis", troop: types.StringNull(), site: ts("site_1"), game: ts("caputchin/games/leaf"),
			axis: "skin", preset: "dark",
			want: "/v1/management/sites/site_1/game-customization/skin/preset?game=caputchin%2Fgames%2Fleaf&name=dark",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := presetPath(c.troop, c.site, c.game, c.axis, c.preset)
			if got != c.want {
				t.Errorf("presetPath = %q, want %q", got, c.want)
			}
		})
	}
}

func TestSchemaAndGamePaths(t *testing.T) {
	if got := schemaPath(ts("troop_1"), types.StringNull(), ts("caputchin/games/leaf"), "configuration"); got != "/v1/management/troops/troop_1/game-customization/configuration/schema?game=caputchin%2Fgames%2Fleaf" {
		t.Errorf("schemaPath = %q", got)
	}
	if got := gamesPath(types.StringNull(), ts("site_1")); got != "/v1/management/sites/site_1/game-customization/games" {
		t.Errorf("gamesPath = %q", got)
	}
	if got := gamePath(ts("troop_1"), types.StringNull(), ts("caputchin/games/leaf")); got != "/v1/management/troops/troop_1/game-customization/game?game=caputchin%2Fgames%2Fleaf" {
		t.Errorf("gamePath = %q", got)
	}
}

func TestJSONHelpers_RoundTrip(t *testing.T) {
	n := jsontypes.NewNormalizedValue(`{"primary":"#000","_theme":"dark"}`)
	m, err := jsonToMap(n)
	if err != nil {
		t.Fatalf("jsonToMap: %v", err)
	}
	if m["primary"] != "#000" || m["_theme"] != "dark" {
		t.Errorf("decoded map wrong: %v", m)
	}
	back, err := mapToNormalized(m)
	if err != nil {
		t.Fatalf("mapToNormalized: %v", err)
	}
	// Semantic equality: key order must not matter.
	eq, _ := back.StringSemanticEquals(context.Background(), n)
	if !eq {
		t.Errorf("round-trip not semantically equal: got %q", back.ValueString())
	}
}

func TestJSONHelpers_NullToEmpty(t *testing.T) {
	m, err := jsonToMap(jsontypes.NewNormalizedNull())
	if err != nil {
		t.Fatalf("jsonToMap(null): %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map for null, got %v", m)
	}
}

func TestPresetWrite_WireContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/v1/management/troops/troop_1/white-label/skin/preset" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("name") != "midnight" {
			t.Errorf("expected name=midnight, got %q", r.URL.Query().Get("name"))
		}
		body, _ := io.ReadAll(r.Body)
		var in struct {
			Values map[string]any `json:"values"`
		}
		_ = json.Unmarshal(body, &in)
		if in.Values["primary"] != "#000" {
			t.Errorf("expected values.primary=#000, got %v", in.Values["primary"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"preset": map[string]any{
				"axis":       "skin",
				"game_id":    nil,
				"name":       "midnight",
				"values":     map[string]any{"primary": "#000"},
				"updated_at": "2026-05-23T00:00:00.000Z",
			},
			"saved_at":            "2026-05-23T00:00:00.000Z",
			"affected_site_count": 3,
			"purge":               map[string]any{"ok": true},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env presetEnvelope
	body := map[string]any{"values": map[string]any{"primary": "#000"}}
	if err := c.Put(context.Background(), presetPath(ts("troop_1"), types.StringNull(), types.StringNull(), "skin", "midnight"), body, &env); err != nil {
		t.Fatalf("put failed: %v", err)
	}
	if env.Preset.Name != "midnight" || env.Preset.UpdatedAt == "" {
		t.Errorf("decoded preset wrong: %+v", env.Preset)
	}
}

func TestGameRegister_WireContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/management/troops/troop_1/game-customization/games" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]any
		_ = json.Unmarshal(body, &in)
		if in["game"] != "caputchin/games/leaf" {
			t.Errorf("expected game in body, got %v", in["game"])
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"game": map[string]any{"game_id": "caputchin/games/leaf", "source": "marketplace"},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env gameEnvelope
	if err := c.Post(context.Background(), gamesPath(ts("troop_1"), types.StringNull()), map[string]any{"game": "caputchin/games/leaf"}, &env); err != nil {
		t.Fatalf("post failed: %v", err)
	}
	if env.Game.Source != "marketplace" {
		t.Errorf("expected source=marketplace, got %q", env.Game.Source)
	}
}
