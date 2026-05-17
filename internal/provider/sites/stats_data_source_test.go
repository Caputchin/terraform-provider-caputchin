// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

func TestSiteStatsRead_DecodesAllCounters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/stats") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"site_id":                    "site_xyz",
			"sessions_started":           int64(12345),
			"sessions_client_completed":  int64(11000),
			"sessions_server_verified":   int64(10000),
			"failed_client_completion":   int64(1345),
			"failed_server_verification": int64(1000),
			"rate_limit_rejections":      int64(42),
			"challenge_blocked":          int64(17),
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var stats apiSiteStats
	if err := c.Get(context.Background(), "/v1/management/sites/site_xyz/stats", &stats); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if stats.SessionsStarted != 12345 {
		t.Errorf("expected 12345, got %d", stats.SessionsStarted)
	}
	if stats.RateLimitRejections != 42 {
		t.Errorf("expected 42, got %d", stats.RateLimitRejections)
	}
	if stats.ChallengeBlocked != 17 {
		t.Errorf("expected 17, got %d", stats.ChallengeBlocked)
	}
	if stats.SiteID != "site_xyz" {
		t.Errorf("expected site_xyz, got %q", stats.SiteID)
	}
}

func TestSiteStatsRead_404Passthrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "not-found"})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var stats apiSiteStats
	err := c.Get(context.Background(), "/v1/management/sites/gone/stats", &stats)
	if !client.IsNotFound(err) {
		t.Fatalf("expected IsNotFound, got %v", err)
	}
}
