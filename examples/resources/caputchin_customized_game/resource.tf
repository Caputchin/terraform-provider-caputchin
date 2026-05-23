resource "caputchin_troop" "marketing" {
  name = "marketing"
}

# Register a game in the troop's customized-games list (Solo+ tier). This is
# the REQUIRED parent for any game-axis preset or custom schema: those reject
# with `game-not-registered` unless the game is registered first. Reference
# this resource's game_id from those children so they apply after it.
# `source` auto-derives from the marketplace catalog when omitted.
#
# WARNING: `terraform destroy` of this resource cascade-deletes the ENTIRE
# game customization for the scope: every preset (all axes) and every custom
# schema for the game. When children reference its game_id, Terraform destroys
# them first, so the cascade is a backstop.
resource "caputchin_customized_game" "leaf" {
  troop_id = caputchin_troop.marketing.id
  game_id  = "caputchin/games/leaf"
}
