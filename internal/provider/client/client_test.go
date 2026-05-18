// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_GetSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/v1/management/me/account" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer cpt_pat_test" {
			t.Errorf("unexpected auth header: %q", got)
		}
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "terraform-provider-caputchin/") {
			t.Errorf("unexpected user-agent: %q", r.Header.Get("User-Agent"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"account": map[string]any{"id": "acct_x", "email": "you@example.com"}})
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL, "cpt_pat_test", "test")

	var out struct {
		Account struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"account"`
	}
	if err := c.Get(context.Background(), "/v1/management/me/account", &out); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Account.ID != "acct_x" {
		t.Errorf("expected acct_x, got %s", out.Account.ID)
	}
}

func TestClient_PostDecodesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "missing-name"})
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL, "cpt_pat_test", "test")

	err := c.Post(context.Background(), "/v1/management/sites", map[string]any{}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.Status != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", apiErr.Status)
	}
	if apiErr.Code != "missing-name" {
		t.Errorf("expected code=missing-name, got %q", apiErr.Code)
	}
}

func TestClient_DeleteIgnoresEmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	c := NewClient(srv.URL, "cpt_pat_test", "test")
	if err := c.Delete(context.Background(), "/v1/management/troops/troop_x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsNotFound(t *testing.T) {
	if IsNotFound(nil) {
		t.Error("nil should not be NotFound")
	}
	apiErr := &APIError{Status: http.StatusNotFound}
	if !IsNotFound(apiErr) {
		t.Error("404 APIError should be NotFound")
	}
	other := &APIError{Status: http.StatusBadRequest}
	if IsNotFound(other) {
		t.Error("400 APIError should not be NotFound")
	}
}
