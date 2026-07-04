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
#
# preview_mode is a development/integration aid: when effectively on (this
# site's own setting, or the troop default when left null), the site key
# still serves its normal experience (the game and its shells/chrome when
# gated), but the backend auto-approves every verification regardless of the
# solve (game replay and cap not enforced), disabling bot protection while
# on. Sessions still record, flagged preview.
#
# reuse grants a short-lived clearance after one successful verification, so
# later widget mounts on this site key skip replaying the game while it is
# valid. reuse_window_ms bounds the clearance lifetime (the server clamps it
# to its own min/max regardless of this value). reuse_persist lets the
# clearance survive a page reload via a first-party cookie, instead of living
# in memory only.
resource "caputchin_site_security_settings" "checkout" {
  site_id         = caputchin_site_key.checkout.id
  require_game    = true
  preview_mode    = true
  reuse           = true
  reuse_window_ms = 60000
  reuse_persist   = false

  depends_on = [caputchin_customized_game.leaf]
}
