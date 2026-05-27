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

// TestTokenRotate_HappyPath exercises the POST /tokens/{id}/rotate wire
// contract via the client directly. The provider's Update
// branch for secret_version bumps is exercised end-to-end at the
// acceptance layer; this test pins the wire shape so a route-side
// rename of `token` or `prefix` would surface here. Mirrors the
// sibling TestRotateSecret_HappyPath shape from sites/resource_test.go.
//
// The response carries both the new bearer (token) and the new prefix
// because rotation generates a fresh credential. prefix rotates
// together with the secret half. Cross-rotation correlation lives on
// the token's id + name, not the prefix.
func TestTokenRotate_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/management/tokens/tok_abc/rotate" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"token":  "cpt_pat_rotated001_rotated_secret_tailpad",
			"prefix": "cpt_pat_rotated0",
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env rotateEnvelope
	if err := c.Post(context.Background(), "/v1/management/tokens/tok_abc/rotate", map[string]any{}, &env); err != nil {
		t.Fatalf("rotate post failed: %v", err)
	}
	if env.Token != "cpt_pat_rotated001_rotated_secret_tailpad" {
		t.Errorf("expected token=cpt_pat_rotated001_rotated_secret_tailpad, got %q", env.Token)
	}
	if env.Prefix != "cpt_pat_rotated0" {
		t.Errorf("expected prefix=cpt_pat_rotated0, got %q", env.Prefix)
	}
}

// TestTokenRotate_EmptyToken pins the contract-violation guard for the
// case where the management API returns a 200 with an absent / empty
// `token` field. The provider's Update branch refuses to write an empty
// string into state and surfaces a `missing-secret-on-rotate`
// diagnostic; this test fixes the wire shape that would trip that
// guard.
func TestTokenRotate_EmptyToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env rotateEnvelope
	if err := c.Post(context.Background(), "/v1/management/tokens/tok_abc/rotate", map[string]any{}, &env); err != nil {
		t.Fatalf("rotate post should succeed at the transport layer: %v", err)
	}
	if env.Token != "" {
		t.Errorf("expected empty token to round-trip, got %q", env.Token)
	}
	if env.Prefix != "" {
		t.Errorf("expected empty prefix to round-trip, got %q", env.Prefix)
	}
}
