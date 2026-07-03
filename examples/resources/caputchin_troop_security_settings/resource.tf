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
# preview_mode null inherits this value. When effectively on, the backend
# auto-approves every verification for those site keys (no game, no
# proof-of-work), disabling bot protection while on.
#
# forbid_reuse is a safety ceiling: it forces the reuse clearance capability
# off for every site key in the troop, regardless of each site's own setting.
resource "caputchin_troop_security_settings" "marketing" {
  troop_id     = caputchin_troop.marketing.id
  force_game   = true
  preview_mode = true
  forbid_reuse = false

  depends_on = [caputchin_customized_game.leaf]
}
