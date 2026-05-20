// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package troops

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

func TestMemberAdd_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/management/troops/troop_x/members" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]any
		_ = json.Unmarshal(body, &in)
		if in["email"] != "alice@example.com" {
			t.Errorf("expected email=alice@example.com, got %v", in["email"])
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"member": map[string]any{
				"id":         "mem_user_1",
				"troop_id":   "troop_x",
				"account_id": "acct_alice",
				"email":      "alice@example.com",
				"perms":      map[string]bool{"create": true, "edit": true, "read": true, "manage": false},
				"scope":      map[string]any{"kind": "full", "site_ids": []string{}},
			},
			"would_consume_seat": true,
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env memberEnvelope
	body := map[string]any{
		"email": "alice@example.com",
		"perms": map[string]bool{"create": true, "edit": true, "read": true, "manage": false},
		"scope": map[string]any{"kind": "full"},
	}
	if err := c.Post(context.Background(), "/v1/management/troops/troop_x/members", body, &env); err != nil {
		t.Fatalf("post failed: %v", err)
	}
	if env.Member.AccountID != "acct_alice" {
		t.Errorf("expected account_id=acct_alice, got %q", env.Member.AccountID)
	}
}

func TestMemberList_DecodesAndFilters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"members": []map[string]any{
				{
					"id":         "mem_user_1",
					"troop_id":   "troop_x",
					"account_id": "acct_alice",
					"email":      "alice@example.com",
					"perms":      map[string]bool{"create": false, "edit": false, "read": true, "manage": false},
					"scope":      map[string]any{"kind": "partial", "site_ids": []string{"site_a"}},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env memberListEnvelope
	if err := c.Get(context.Background(), "/v1/management/troops/troop_x/members", &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(env.Members) != 1 || env.Members[0].Scope.SiteIDs[0] != "site_a" {
		t.Errorf("unexpected list shape: %+v", env.Members)
	}
}
