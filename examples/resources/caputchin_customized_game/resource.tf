resource "caputchin_troop" "marketing" {
  name = "marketing"
}

# Register a game in the troop's customized-games list (Solo+ tier) so it
# shows up before any preset or schema is authored. `source` auto-derives
# from the marketplace catalog when omitted.
#
# WARNING: `terraform destroy` of this resource cascade-deletes the ENTIRE
# game customization for the scope: every preset (all axes) and every custom
# schema for the game. Do not manage this alongside individual
# caputchin_white_label_preset / caputchin_custom_game_schema resources for
# the same game unless you intend the destroy to remove them too.
resource "caputchin_customized_game" "leaf" {
  troop_id = caputchin_troop.marketing.id
  game_id  = "caputchin/games/leaf"
}
