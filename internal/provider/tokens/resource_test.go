// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package tokens

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

func TestTokenCreate_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/management/tokens" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]string
		_ = json.Unmarshal(body, &in)
		if in["name"] != "ci-prod" {
			t.Errorf("expected name=ci-prod, got %q", in["name"])
		}
		if in["type"] != "troop" {
			t.Errorf("expected type=troop, got %q", in["type"])
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token": map[string]any{
				"id":         "tok_abc",
				"name":       "ci-prod",
				"type":       "troop",
				"prefix":     "cpt_pat_abc12345",
				"value":      "cpt_pat_abc12345_full_secret_value",
				"created_at": int64(1747500000000),
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env createEnvelope
	if err := c.Post(context.Background(), "/v1/management/tokens", map[string]string{"name": "ci-prod", "type": "troop"}, &env); err != nil {
		t.Fatalf("post failed: %v", err)
	}
	if env.Token.ID != "tok_abc" {
		t.Errorf("expected id=tok_abc, got %q", env.Token.ID)
	}
	if env.Token.Value == "" {
		t.Errorf("expected value to be present on create")
	}
}

func TestTokenList_DecodesAndFiltersById(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tokens": []map[string]any{
				{
					"id":           "tok_keep",
					"name":         "ci-prod",
					"type":         "troop",
					"prefix":       "cpt_pat_keep",
					"last_used_at": nil,
					"created_at":   int64(1747500000000),
				},
				{
					"id":           "tok_other",
					"name":         "other",
					"type":         "account",
					"prefix":       "cpt_pat_other",
					"last_used_at": int64(1747600000000),
					"created_at":   int64(1747400000000),
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env listEnvelope
	if err := c.Get(context.Background(), "/v1/management/tokens", &env); err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if len(env.Tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d", len(env.Tokens))
	}
	if env.Tokens[0].ID != "tok_keep" {
		t.Errorf("expected first id=tok_keep, got %q", env.Tokens[0].ID)
	}
	if env.Tokens[1].Type != "account" {
		t.Errorf("expected second type=account, got %q", env.Tokens[1].Type)
	}
}

func TestTokenDelete_Succeeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/management/tokens/tok_abc" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	if err := c.Delete(context.Background(), "/v1/management/tokens/tok_abc"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}
