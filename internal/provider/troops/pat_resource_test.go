// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package troops

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

func TestPatAttach_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/management/troops/troop_x/pats" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]any
		_ = json.Unmarshal(body, &in)
		if in["pat_id"] != "tok_abc" {
			t.Errorf("expected pat_id=tok_abc, got %v", in["pat_id"])
		}
		scope, _ := in["scope"].(map[string]any)
		if scope["kind"] != "partial" {
			t.Errorf("expected scope.kind=partial, got %v", scope["kind"])
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pat": map[string]any{
				"id":         "mem_pat_1",
				"troop_id":   "troop_x",
				"pat_id":     "tok_abc",
				"pat_name":   "ci-prod",
				"pat_prefix": "cpt_pat_prefix",
				"perms":      map[string]bool{"create": true, "edit": false, "read": true, "manage": false},
				"scope":      map[string]any{"kind": "partial", "site_ids": []string{"site_a", "site_b"}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env patEnvelope
	body := map[string]any{
		"pat_id": "tok_abc",
		"perms":  map[string]bool{"create": true, "edit": false, "read": true, "manage": false},
		"scope":  map[string]any{"kind": "partial", "site_ids": []string{"site_a", "site_b"}},
	}
	if err := c.Post(context.Background(), "/v1/management/troops/troop_x/pats", body, &env); err != nil {
		t.Fatalf("post failed: %v", err)
	}
	if env.Pat.ID != "mem_pat_1" {
		t.Errorf("expected id=mem_pat_1, got %q", env.Pat.ID)
	}
	if env.Pat.Scope.Kind != "partial" || len(env.Pat.Scope.SiteIDs) != 2 {
		t.Errorf("scope decode mismatch: %+v", env.Pat.Scope)
	}
}

func TestPatList_DecodesAndFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pats": []map[string]any{
				{
					"id":         "mem_pat_1",
					"troop_id":   "troop_x",
					"pat_id":     "tok_abc",
					"pat_name":   "ci-prod",
					"pat_prefix": "cpt_pat_prefix",
					"perms":      map[string]bool{"create": false, "edit": false, "read": true, "manage": false},
					"scope":      map[string]any{"kind": "full", "site_ids": []string{}},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env patListEnvelope
	if err := c.Get(context.Background(), "/v1/management/troops/troop_x/pats", &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(env.Pats) != 1 || env.Pats[0].Scope.Kind != "full" {
		t.Errorf("unexpected list shape: %+v", env.Pats)
	}
}

func TestPatDetach_Succeeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || !strings.HasPrefix(r.URL.Path, "/v1/management/troops/troop_x/pats/mem_pat_1") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"removed": true})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	if err := c.Delete(context.Background(), "/v1/management/troops/troop_x/pats/mem_pat_1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestSplitImportID_RoundTrips(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"troop_x:mem_1", []string{"troop_x", "mem_1"}},
		{"a:b:c", []string{"a", "b:c"}}, // SplitN(2) preserves trailing colons
		{"missing", nil},
		{":empty", nil},
		{"empty:", nil},
	}
	for _, tc := range cases {
		got := splitImportID(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("input %q: got %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("input %q: got %v, want %v", tc.in, got, tc.want)
				break
			}
		}
	}
}
