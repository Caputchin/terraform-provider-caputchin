// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package whitelabel

import (
	"context"
	"encoding/json"
	"errors"
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

func TestRunArtifactPaths(t *testing.T) {
	// PUT/GET-bytes/DELETE endpoint.
	if got := runArtifactPath(ts("troop_1"), types.StringNull(), ts("customer/my-game")); got != "/v1/management/troops/troop_1/game-customization/run-artifact?game=customer%2Fmy-game" {
		t.Errorf("runArtifactPath troop = %q", got)
	}
	if got := runArtifactPath(types.StringNull(), ts("site_1"), ts("customer/my-game")); got != "/v1/management/sites/site_1/game-customization/run-artifact?game=customer%2Fmy-game" {
		t.Errorf("runArtifactPath site = %q", got)
	}
	// JSON detail endpoint (different path; Read consumes this).
	if got := runArtifactDetailPath(ts("troop_1"), types.StringNull(), ts("customer/my-game")); got != "/v1/management/troops/troop_1/game-customization/run-artifact/detail?game=customer%2Fmy-game" {
		t.Errorf("runArtifactDetailPath = %q", got)
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

func TestRunArtifactUpload_WireContract(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/v1/management/troops/troop_1/game-customization/run-artifact" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("game") != "customer/my-game" {
			t.Errorf("expected game=customer/my-game, got %q", r.URL.Query().Get("game"))
		}
		if ct := r.Header.Get("Content-Type"); ct == "" || ct == "application/json" {
			t.Errorf("expected multipart/form-data content-type, got %q", ct)
		}
		// Parse the multipart body; the helper packs run.js as `run` and each
		// module under field `module` with the file's basename.
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		runParts := r.MultipartForm.File["run"]
		if len(runParts) != 1 {
			t.Fatalf("expected exactly one run part, got %d", len(runParts))
		}
		if runParts[0].Filename != "run.js" {
			t.Errorf("run filename = %q, want run.js", runParts[0].Filename)
		}
		modParts := r.MultipartForm.File["module"]
		if len(modParts) != 1 || modParts[0].Filename != "engine.wasm" {
			t.Errorf("module parts = %+v, want one engine.wasm", modParts)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version_hash":  "abcdef0123456789",
			"self_check_ok": true,
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	parts := []client.MultipartFile{
		{FieldName: "run", Filename: "run.js", Bytes: []byte("export function run(){return{passed:true,score:1,durationMs:1}}")},
		{FieldName: "module", Filename: "engine.wasm", Bytes: []byte("\x00asm")},
	}
	var out runArtifactUploadResponseWire
	if err := c.PutMultipart(context.Background(), runArtifactPath(ts("troop_1"), types.StringNull(), ts("customer/my-game")), parts, &out); err != nil {
		t.Fatalf("multipart put: %v", err)
	}
	if out.VersionHash != "abcdef0123456789" || !out.SelfCheckOK {
		t.Errorf("decoded response wrong: %+v", out)
	}
}

func TestRunArtifactUpload_SelfCheckFailedSurfacesAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"self-check-failed","details":"no-verdict"}`))
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var out runArtifactUploadResponseWire
	err := c.PutMultipart(context.Background(), runArtifactPath(ts("troop_1"), types.StringNull(), ts("customer/my-game")), []client.MultipartFile{
		{FieldName: "run", Filename: "run.js", Bytes: []byte("garbage")},
	}, &out)
	if err == nil {
		t.Fatalf("expected error on 400, got nil")
	}
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *client.APIError, got %T (%v)", err, err)
	}
	if apiErr.Code != "self-check-failed" {
		t.Errorf("error code = %q, want self-check-failed", apiErr.Code)
	}
}

func TestRunArtifactDetail_WireDecode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/management/troops/troop_1/game-customization/run-artifact/detail" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version_hash":  "abcdef0123456789",
			"uploaded_at":   "2026-05-28T12:00:00.000Z",
			"self_check_ok": true,
			"run":           map[string]any{"name": "run.js", "type": "entry", "integrity": "sha384-run", "size": 1024},
			"modules": []any{
				map[string]any{"name": "engine.wasm", "type": "wasm", "integrity": "sha384-eng", "size": 2048},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var detail runArtifactDetailWire
	if err := c.Get(context.Background(), runArtifactDetailPath(ts("troop_1"), types.StringNull(), ts("customer/my-game")), &detail); err != nil {
		t.Fatalf("get detail: %v", err)
	}
	if detail.VersionHash != "abcdef0123456789" || detail.UploadedAt == "" || !detail.SelfCheckOK {
		t.Errorf("detail decoded wrong: %+v", detail)
	}
	if len(detail.Modules) != 1 || detail.Modules[0].Name != "engine.wasm" {
		t.Errorf("modules decoded wrong: %+v", detail.Modules)
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
