resource "caputchin_troop" "marketing" {
  name = "marketing"
}

# Declare the editable fields of a custom (non-marketplace) game, per axis.
# The schema describes which keys a preset for this game may set; it carries
# no plan-tier gate (it is metadata). Pair it with caputchin_white_label_preset
# resources that supply the actual values.
resource "caputchin_custom_game_schema" "space_shooter_skin" {
  troop_id = caputchin_troop.marketing.id
  game_id  = "my-org/space-shooter"
  axis     = "skin"
  schema = jsonencode({
    ship_color = "color"
    bg_image   = "image"
  })
}
