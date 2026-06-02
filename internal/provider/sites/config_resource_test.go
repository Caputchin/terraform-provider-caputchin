// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

func TestSiteConfigGet_DecodesNullables(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/cap-config") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"site_id": "site_xyz",
			"config": map[string]any{
				"instrumentation":          true,
				"difficulty":               4,
				"challenge_count":          50,
				"obfuscation_level":        5,
				"block_automated_browsers": true,
				"block_non_browser_ua":     nil,
				"required_headers":         nil,
				"ratelimit_max":            100,
				"cors_origins":             []string{"https://example.com"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env siteConfigEnvelope
	if err := c.Get(context.Background(), configPath("site_xyz"), &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if env.Config.Difficulty != 4 {
		t.Errorf("expected difficulty=4, got %d", env.Config.Difficulty)
	}
	if !env.Config.Instrumentation {
		t.Errorf("expected instrumentation=true, got %v", env.Config.Instrumentation)
	}
	if env.Config.BlockNonBrowserUA != nil {
		t.Errorf("expected null block_non_browser_ua, got %v", env.Config.BlockNonBrowserUA)
	}
	if env.Config.RequiredHeaders != nil {
		t.Errorf("expected null required_headers, got %v", env.Config.RequiredHeaders)
	}
	if env.Config.RatelimitMax == nil || *env.Config.RatelimitMax != 100 {
		t.Errorf("expected ratelimit_max=100, got %v", env.Config.RatelimitMax)
	}
	if len(env.Config.CorsOrigins) != 1 || env.Config.CorsOrigins[0] != "https://example.com" {
		t.Errorf("expected cors_origins=[https://example.com], got %v", env.Config.CorsOrigins)
	}
}

func TestSiteConfigPatch_OnlyChangedFields(t *testing.T) {
	var bodyCapture atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		raw, _ := io.ReadAll(r.Body)
		bodyCapture.Store(raw)
		_ = json.NewEncoder(w).Encode(map[string]any{"site_id": "site_xyz", "config": map[string]any{
			"difficulty": 5, "challenge_count": 50, "obfuscation_level": 5,
			"block_automated_browsers": true,
			"block_non_browser_ua":     nil, "required_headers": nil,
			"ratelimit_max": 100, "cors_origins": nil,
		}})
	}))
	t.Cleanup(srv.Close)

	r := &siteConfigResource{client: client.NewClient(srv.URL, "cpt_pat_test", "test")}

	plan := siteConfigModel{
		SiteID:        types.StringValue("site_xyz"),
		PowDifficulty: types.Int64Value(5),
	}
	state := siteConfigModel{
		SiteID:        types.StringValue("site_xyz"),
		PowDifficulty: types.Int64Value(4),
	}
	body := r.buildPatchBody(plan, state)
	if _, ok := body["difficulty"]; !ok {
		t.Errorf("expected difficulty in body, got %v", body)
	}
	if _, ok := body["challenge_count"]; ok {
		t.Errorf("did NOT expect challenge_count in body (unchanged), got %v", body)
	}
	if v, _ := body["difficulty"].(int64); v != 5 {
		t.Errorf("expected difficulty=5, got %v", body["difficulty"])
	}
}

func TestSiteConfigPatch_InstrumentationToggle(t *testing.T) {
	r := &siteConfigResource{}
	plan := siteConfigModel{
		SiteID:          types.StringValue("site_xyz"),
		Instrumentation: types.BoolValue(false),
	}
	state := siteConfigModel{
		SiteID:          types.StringValue("site_xyz"),
		Instrumentation: types.BoolValue(true),
	}
	body := r.buildPatchBody(plan, state)
	v, ok := body["instrumentation"]
	if !ok {
		t.Fatalf("expected instrumentation in body, got %v", body)
	}
	if b, _ := v.(bool); b != false {
		t.Errorf("expected instrumentation=false, got %v", v)
	}
}

func TestSiteConfigPatch_NullRequiresHeadersIsExplicitClear(t *testing.T) {
	r := &siteConfigResource{}

	prior, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"X-Foo"})
	state := siteConfigModel{
		SiteID:          types.StringValue("site_xyz"),
		RequiredHeaders: prior,
	}
	plan := siteConfigModel{
		SiteID:          types.StringValue("site_xyz"),
		RequiredHeaders: types.ListNull(types.StringType),
	}
	body := r.buildPatchBody(plan, state)
	v, ok := body["required_headers"]
	if !ok {
		t.Fatalf("expected required_headers in body (null clear), got %v", body)
	}
	if v != nil {
		t.Errorf("expected required_headers=nil, got %v", v)
	}
}

func TestSiteConfigPatch_UnknownPlanValuesAreSkipped(t *testing.T) {
	r := &siteConfigResource{}

	state := siteConfigModel{
		SiteID:        types.StringValue("site_xyz"),
		PowDifficulty: types.Int64Value(4),
	}
	plan := siteConfigModel{
		SiteID:        types.StringValue("site_xyz"),
		PowDifficulty: types.Int64Unknown(),
	}
	body := r.buildPatchBody(plan, state)
	if len(body) != 0 {
		t.Errorf("expected empty body when plan is Unknown, got %v", body)
	}
}

func TestSiteConfigToModel_PreservesNullables(t *testing.T) {
	cfg := apiSiteConfig{
		Difficulty: 4, ChallengeCount: 50, ObfuscationLevel: 5,
		BlockAutomatedBrowsers: true,
		BlockNonBrowserUA:      nil,
		RequiredHeaders:        nil,
		RatelimitMax:           nil,
		CorsOrigins:            nil,
	}
	model := cfg.toModel(context.Background(), "site_xyz", &fakeDiags{})
	if !model.BlockNonBrowserUA.IsNull() {
		t.Errorf("expected block_non_browser_ua null, got %v", model.BlockNonBrowserUA)
	}
	if !model.RequiredHeaders.IsNull() {
		t.Errorf("expected required_headers null, got %v", model.RequiredHeaders)
	}
	if !model.RatelimitMax.IsNull() {
		t.Errorf("expected ratelimit_max null, got %v", model.RatelimitMax)
	}
	if !model.CorsOrigins.IsNull() {
		t.Errorf("expected cors_origins null, got %v", model.CorsOrigins)
	}
}

type fakeDiags struct{ errors []string }

func (f *fakeDiags) AddError(summary, detail string) {
	f.errors = append(f.errors, summary+": "+detail)
}
