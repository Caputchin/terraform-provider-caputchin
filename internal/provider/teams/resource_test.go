// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

package teams

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

// These tests exercise the resource's CRUD path against a synthetic API.
// They drive the Client surface directly because the Framework's full
// schema/State plumbing is exercised at the acceptance-test layer
// (TF_ACC=1, plan §97-101).

func TestTeamCreate_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/management/teams" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]string
		_ = json.Unmarshal(body, &in)
		if in["name"] != "marketing" {
			t.Errorf("expected name=marketing, got %q", in["name"])
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"team": map[string]any{
				"id":         "team_abc",
				"account_id": "acct_x",
				"kind":       "shared",
				"name":       "marketing",
				"tier":       "troop",
				"created_at": int64(1747500000000),
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env teamEnvelope
	if err := c.Post(context.Background(), "/v1/management/teams", map[string]string{"name": "marketing"}, &env); err != nil {
		t.Fatalf("post failed: %v", err)
	}
	if env.Team.ID != "team_abc" {
		t.Errorf("expected id=team_abc, got %q", env.Team.ID)
	}
	if env.Team.Kind != "shared" {
		t.Errorf("expected kind=shared, got %q", env.Team.Kind)
	}
	if env.Team.CreatedAt != 1747500000000 {
		t.Errorf("expected created_at=1747500000000, got %d", env.Team.CreatedAt)
	}
}

func TestTeamRead_NotFoundClearsState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "not-found"})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env teamEnvelope
	err := c.Get(context.Background(), "/v1/management/teams/team_gone", &env)
	if !client.IsNotFound(err) {
		t.Fatalf("expected IsNotFound, got %v", err)
	}
}

func TestTeamUpdate_PatchesName(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/v1/management/teams/team_xyz") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var in map[string]string
		_ = json.Unmarshal(body, &in)
		if in["name"] != "renamed" {
			t.Errorf("expected name=renamed, got %q", in["name"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"team": map[string]any{
				"id":         "team_xyz",
				"account_id": "acct_x",
				"kind":       "shared",
				"name":       "renamed",
				"tier":       "troop",
				"created_at": int64(1747500000000),
			},
		})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	var env teamEnvelope
	if err := c.Patch(context.Background(), "/v1/management/teams/team_xyz", map[string]string{"name": "renamed"}, &env); err != nil {
		t.Fatalf("patch failed: %v", err)
	}
	if hits.Load() != 1 {
		t.Errorf("expected exactly 1 request, got %d", hits.Load())
	}
	if env.Team.Name != "renamed" {
		t.Errorf("expected name=renamed, got %q", env.Team.Name)
	}
}

func TestTeamDelete_Succeeds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"deleted": true, "cap_orphans": 0})
	}))
	t.Cleanup(srv.Close)

	c := client.NewClient(srv.URL, "cpt_pat_test", "test")
	if err := c.Delete(context.Background(), "/v1/management/teams/team_xyz"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestApiTeam_ToModel(t *testing.T) {
	src := apiTeam{
		ID:        "team_abc",
		Name:      "ops",
		AccountID: "acct_x",
		Kind:      "shared",
		Tier:      "apex",
		CreatedAt: 1747500000000,
	}
	got := src.toModel()
	if got.ID.ValueString() != "team_abc" {
		t.Errorf("ID: got %s", got.ID.ValueString())
	}
	if got.Tier.ValueString() != "apex" {
		t.Errorf("Tier: got %s", got.Tier.ValueString())
	}
	if got.CreatedAt.ValueInt64() != 1747500000000 {
		t.Errorf("CreatedAt: got %d", got.CreatedAt.ValueInt64())
	}
}
