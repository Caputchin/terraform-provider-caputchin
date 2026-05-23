resource "caputchin_troop" "marketing" {
  name = "marketing"
}

# A custom-game schema requires the game registered first. Declare the parent
# (a custom game absent from the marketplace catalog resolves source = "custom")
# and reference its game_id so Terraform creates it before the schema.
resource "caputchin_customized_game" "space_shooter" {
  troop_id = caputchin_troop.marketing.id
  game_id  = "my-org/space-shooter"
}

# Declare the editable fields of a custom (non-marketplace) game, per axis.
# The schema describes which keys a preset for this game may set; it carries
# no plan-tier gate (it is metadata). Pair it with caputchin_white_label_preset
# resources that supply the actual values.
resource "caputchin_custom_game_schema" "space_shooter_skin" {
  troop_id = caputchin_troop.marketing.id
  game_id  = caputchin_customized_game.space_shooter.game_id
  axis     = "skin"
  schema = jsonencode({
    ship_color = "color"
    bg_image   = "image"
  })
}
