resource "caputchin_troop" "marketing" {
  name = "marketing"
}

resource "caputchin_site_key" "checkout" {
  name     = "checkout-prod"
  troop_id = caputchin_troop.marketing.id
}

# Install a marketplace game on the site key so it can serve as the gate.
resource "caputchin_customized_game" "leaf" {
  site_id = caputchin_site_key.checkout.id
  game_id = "caputchin/games/leaf"
}

# Require a game to pass verification on this site key (instead of proof-of-work
# only). Enabling needs at least one installed marketplace game with a
# replayable artifact for the site (its own, as above, or inherited from the
# troop). The API rejects the change otherwise.
resource "caputchin_site_security_settings" "checkout" {
  site_id      = caputchin_site_key.checkout.id
  require_game = true

  depends_on = [caputchin_customized_game.leaf]
}
