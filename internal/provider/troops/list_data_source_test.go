// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package troops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

func TestTroopsList_DecodesEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"troops": []map[string]any{
				{
					"id":         "troop_personal",
					"name":       "personal",
					"account_id": "acct_x",
					"kind":       "personal",
					"tier":       "free",
					"created_at": int64(1747500000000),
				},
				{
					"id":         "troop_marketing",
					"name":       "marketing",
					"account_id": "acct_x",
					"kind":       "shared",
					"tier":       "troop",
					"created_at": int64(1747500000001),
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env troopListEnvelope
	if err := c.Get(context.Background(), "/v1/management/troops", &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(env.Troops) != 2 {
		t.Fatalf("expected 2 troops, got %d", len(env.Troops))
	}
	if env.Troops[0].Kind != "personal" || env.Troops[1].Kind != "shared" {
		t.Errorf("unexpected troops: %+v", env.Troops)
	}
}
