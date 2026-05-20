// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package sites

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

func TestHostedVerificationPut_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/v1/management/hosted-verification/site_x" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		webhookURL := "https://example.com/wh"
		_ = json.NewEncoder(w).Encode(map[string]any{
			"site_id":     "site_x",
			"enabled":     true,
			"webhook_url": webhookURL,
			"email_to":    nil,
			"created_at":  int64(1747500000000),
			"updated_at":  int64(1747500000000),
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var cfg apiHostedVerificationConfig
	body := map[string]any{
		"enabled":     true,
		"webhook_url": "https://example.com/wh",
		"email_to":    nil,
	}
	if err := c.Put(context.Background(), "/v1/management/hosted-verification/site_x", body, &cfg); err != nil {
		t.Fatalf("put failed: %v", err)
	}
	if cfg.WebhookURL == nil || *cfg.WebhookURL != "https://example.com/wh" {
		t.Errorf("webhook_url decode mismatch: %+v", cfg.WebhookURL)
	}
	if cfg.EmailTo != nil {
		t.Errorf("expected email_to nil, got %v", *cfg.EmailTo)
	}
}

func TestHostedVerificationGet_NotConfigured(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"site_id":     "site_x",
			"enabled":     false,
			"webhook_url": nil,
			"email_to":    nil,
			"created_at":  nil,
			"updated_at":  nil,
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var cfg apiHostedVerificationConfig
	if err := c.Get(context.Background(), "/v1/management/hosted-verification/site_x", &cfg); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if cfg.Enabled {
		t.Errorf("expected disabled empty config, got enabled=true")
	}
	if cfg.WebhookURL != nil || cfg.EmailTo != nil || cfg.CreatedAt != nil || cfg.UpdatedAt != nil {
		t.Errorf("expected all nullable fields nil, got %+v", cfg)
	}
}
