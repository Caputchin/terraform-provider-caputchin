resource "caputchin_troop" "marketing" {
  name = "marketing"
}

# Install a marketplace game at the troop level so every site key inherits it.
resource "caputchin_customized_game" "leaf" {
  troop_id = caputchin_troop.marketing.id
  game_id  = "caputchin/games/leaf"
}

# Force EVERY site key in the troop to gate verification with a game, regardless
# of each site key's own setting. Enabling needs at least one installed
# troop-level marketplace game with a replayable artifact (above) so every site
# is covered by inheritance. The API rejects the change otherwise.
resource "caputchin_troop_security_settings" "marketing" {
  troop_id   = caputchin_troop.marketing.id
  force_game = true

  depends_on = [caputchin_customized_game.leaf]
}
