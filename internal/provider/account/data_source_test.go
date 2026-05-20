// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package account

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

func TestAccountRead_Decodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/management/me/account" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"account": map[string]any{
				"id":         "acct_x",
				"email":      "you@example.com",
				"created_at": int64(1747500000000),
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env accountEnvelope
	if err := c.Get(context.Background(), "/v1/management/me/account", &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if env.Account.ID != "acct_x" {
		t.Errorf("expected acct_x, got %q", env.Account.ID)
	}
	if env.Account.Email != "you@example.com" {
		t.Errorf("expected email, got %q", env.Account.Email)
	}
}

// TestAccountRead_TroopPATForbiddenPassthrough validates the APIError-code
// passthrough contract. The specific error code is forward-compat with
// ADR-0027 (troop-PAT rejection on me/* routes is not yet enforced by the
// platform — see me/account/route.ts:7-15).
func TestAccountRead_TroopPATForbiddenPassthrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "troop-pat-cannot-access-account"})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env accountEnvelope
	err := c.Get(context.Background(), "/v1/management/me/account", &env)
	if err == nil {
		t.Fatal("expected error from 403")
	}
	apiErr, ok := err.(*client.APIError)
	if !ok {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.Code != "troop-pat-cannot-access-account" {
		t.Errorf("expected code passthrough, got %q", apiErr.Code)
	}
}
