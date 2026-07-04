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
#
# preview_mode sets the troop-wide default: any site key that leaves its own
# preview_mode null inherits this value. When effectively on, those site keys
# still serve their normal experience (the game and its shells/chrome), but
# the backend auto-approves every verification regardless of the solve (game
# replay and cap not enforced), disabling bot protection while on.
#
# reuse / reuse_window_ms / reuse_persist set the troop-wide verification-reuse
# default: any site key that leaves its own value null inherits these. When
# effectively on, one solve grants a short-lived clearance so later widget mounts
# skip replaying the game.
resource "caputchin_troop_security_settings" "marketing" {
  troop_id        = caputchin_troop.marketing.id
  force_game      = true
  preview_mode    = true
  reuse           = false
  reuse_window_ms = 300000
  reuse_persist   = false

  depends_on = [caputchin_customized_game.leaf]
}
