// Copyright (c) 2026 Caputchin
// SPDX-License-Identifier: MPL-2.0

// Acceptance tests for the override-pipeline resources (ADR-0061). These
// drive REAL Terraform (plan/apply/import/destroy) against a LIVE management
// API, so they are gated behind TF_ACC=1 (resource.Test skips otherwise) and
// require CAPUTCHIN_ENDPOINT + CAPUTCHIN_MANAGEMENT_TOKEN. Run locally against
// `wrangler dev` with `make testacc`; they do NOT run in CI (TF_ACC unset).
//
// External test package (whitelabel_test): the suite imports the provider
// package, which itself imports whitelabel, so an internal-package test would
// be an import cycle.
//
// Tier-safety: every test exercises Solo+ surfaces (game-customization
// `configuration` axis, custom-game schemas which carry no tier gate, and the
// customized-games registry which is Solo+), so they pass on any account tier.
// The widget-shell white-label path (Apex) is the same resource code minus the
// game id; its gate is covered by unit tests + the API. Each test is
// self-contained: it creates a throwaway troop and tears it down.

package whitelabel_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/caputchin/terraform-provider-caputchin/internal/provider"
	"github.com/caputchin/terraform-provider-caputchin/internal/provider/client"
)

const accTestGame = "tf-acc/override-game"

var protoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"caputchin": providerserver.NewProtocol6WithError(provider.New("acctest")()),
}

func testAccPreCheck(t *testing.T) {
	if os.Getenv("CAPUTCHIN_MANAGEMENT_TOKEN") == "" {
		t.Fatal("CAPUTCHIN_MANAGEMENT_TOKEN must be set for acceptance tests (an account-PAT for the test account)")
	}
	if os.Getenv("CAPUTCHIN_ENDPOINT") == "" {
		t.Fatal("CAPUTCHIN_ENDPOINT must be set (e.g. http://localhost:8787 for wrangler dev)")
	}
}

// accClient builds a management-API client from the same env the provider
// reads, used by CheckDestroy to confirm the server actually deleted the row.
func accClient() *client.Client {
	return client.NewClient(os.Getenv("CAPUTCHIN_ENDPOINT"), os.Getenv("CAPUTCHIN_MANAGEMENT_TOKEN"), "acctest")
}

// gone reports whether a GET against `path` indicates the resource is absent.
// Lenient by design: after a destroy the parent troop is also gone, so the
// API may answer 404 (preset/schema/game missing) OR a troop-scope error;
// either way the resource is not retrievable. Only a clean 200 means it
// survived destroy.
func gone(path string) error {
	var out map[string]any
	err := accClient().Get(context.Background(), path, &out)
	if err == nil {
		return fmt.Errorf("resource still retrievable after destroy: GET %s returned 200", path)
	}
	return nil
}

func troopID(s *terraform.State) (string, error) {
	rs, ok := s.RootModule().Resources["caputchin_troop.test"]
	if !ok || rs.Primary == nil {
		return "", fmt.Errorf("caputchin_troop.test not found in state")
	}
	return rs.Primary.ID, nil
}

// ---------- caputchin_white_label_preset (game-axis configuration) ----------

func TestAccWhiteLabelPreset_gameConfiguration(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             testAccCheckPresetDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPresetConfig(`{"difficulty":"hard"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("caputchin_white_label_preset.test", "axis", "configuration"),
					resource.TestCheckResourceAttr("caputchin_white_label_preset.test", "name", "hard"),
					resource.TestCheckResourceAttr("caputchin_white_label_preset.test", "game_id", accTestGame),
					resource.TestCheckResourceAttr("caputchin_white_label_preset.test", "values", `{"difficulty":"hard"}`),
					resource.TestCheckResourceAttrSet("caputchin_white_label_preset.test", "troop_id"),
					resource.TestCheckResourceAttrSet("caputchin_white_label_preset.test", "updated_at"),
				),
			},
			{
				Config: testAccPresetConfig(`{"difficulty":"insane"}`),
				Check:  resource.TestCheckResourceAttr("caputchin_white_label_preset.test", "values", `{"difficulty":"insane"}`),
			},
			{
				ResourceName:      "caputchin_white_label_preset.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					id, err := troopID(s)
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("troop|%s|%s|configuration|hard", id, accTestGame), nil
				},
			},
		},
	})
}

func testAccPresetConfig(values string) string {
	return fmt.Sprintf(`
resource "caputchin_troop" "test" { name = "tf-acc-override-preset" }

resource "caputchin_white_label_preset" "test" {
  troop_id = caputchin_troop.test.id
  game_id  = %q
  axis     = "configuration"
  name     = "hard"
  values   = %q
}
`, accTestGame, values)
}

func testAccCheckPresetDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "caputchin_white_label_preset" {
			continue
		}
		a := rs.Primary.Attributes
		q := url.Values{}
		q.Set("name", a["name"])
		q.Set("game", a["game_id"])
		path := fmt.Sprintf("/v1/management/troops/%s/game-customization/%s/preset?%s", a["troop_id"], a["axis"], q.Encode())
		if err := gone(path); err != nil {
			return err
		}
	}
	return nil
}

// ---------- caputchin_custom_game_schema (no tier gate) ----------

func TestAccCustomGameSchema(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             testAccCheckSchemaDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccSchemaConfig(`{"ship_color":"color"}`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("caputchin_custom_game_schema.test", "axis", "skin"),
					resource.TestCheckResourceAttr("caputchin_custom_game_schema.test", "game_id", accTestGame),
					resource.TestCheckResourceAttr("caputchin_custom_game_schema.test", "schema", `{"ship_color":"color"}`),
					resource.TestCheckResourceAttrSet("caputchin_custom_game_schema.test", "updated_at"),
				),
			},
			{
				Config: testAccSchemaConfig(`{"ship_color":"color","bg":"image"}`),
				Check:  resource.TestCheckResourceAttr("caputchin_custom_game_schema.test", "schema", `{"bg":"image","ship_color":"color"}`),
			},
			{
				ResourceName:      "caputchin_custom_game_schema.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					id, err := troopID(s)
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("troop|%s|%s|skin", id, accTestGame), nil
				},
			},
		},
	})
}

func testAccSchemaConfig(schema string) string {
	return fmt.Sprintf(`
resource "caputchin_troop" "test" { name = "tf-acc-override-schema" }

resource "caputchin_custom_game_schema" "test" {
  troop_id = caputchin_troop.test.id
  game_id  = %q
  axis     = "skin"
  schema   = %q
}
`, accTestGame, schema)
}

func testAccCheckSchemaDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "caputchin_custom_game_schema" {
			continue
		}
		a := rs.Primary.Attributes
		q := url.Values{}
		q.Set("game", a["game_id"])
		path := fmt.Sprintf("/v1/management/troops/%s/game-customization/%s/schema?%s", a["troop_id"], a["axis"], q.Encode())
		if err := gone(path); err != nil {
			return err
		}
	}
	return nil
}

// ---------- caputchin_customized_game (Solo+, source auto-derived) ----------

func TestAccCustomizedGame(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: protoV6ProviderFactories,
		CheckDestroy:             testAccCheckCustomizedGameDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomizedGameConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("caputchin_customized_game.test", "game_id", accTestGame),
					// accTestGame is not in the marketplace catalog → custom.
					resource.TestCheckResourceAttr("caputchin_customized_game.test", "source", "custom"),
				),
			},
			{
				ResourceName:      "caputchin_customized_game.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					id, err := troopID(s)
					if err != nil {
						return "", err
					}
					return fmt.Sprintf("troop|%s|%s", id, accTestGame), nil
				},
			},
		},
	})
}

func testAccCustomizedGameConfig() string {
	return fmt.Sprintf(`
resource "caputchin_troop" "test" { name = "tf-acc-override-game" }

resource "caputchin_customized_game" "test" {
  troop_id = caputchin_troop.test.id
  game_id  = %q
}
`, accTestGame)
}

func testAccCheckCustomizedGameDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "caputchin_customized_game" {
			continue
		}
		a := rs.Primary.Attributes
		q := url.Values{}
		q.Set("game", a["game_id"])
		path := fmt.Sprintf("/v1/management/troops/%s/game-customization/game?%s", a["troop_id"], q.Encode())
		if err := gone(path); err != nil {
			return err
		}
	}
	return nil
}
