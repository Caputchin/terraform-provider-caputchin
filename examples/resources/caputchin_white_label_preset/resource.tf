resource "caputchin_troop" "marketing" {
  name = "marketing"
}

# Widget-shell (white-label) preset, Apex tier. Applies to every site key
# under the troop. Omit game_id for widget-shell presets, they need no game
# registration.
resource "caputchin_white_label_preset" "midnight" {
  troop_id = caputchin_troop.marketing.id
  axis     = "skin"
  name     = "midnight"
  values = jsonencode({
    _theme  = "dark"
    primary = "#0F1810"
  })
}

# A game-axis preset requires the game registered first. Declare the parent
# and reference its game_id so Terraform creates it before the preset.
resource "caputchin_customized_game" "leaf" {
  troop_id = caputchin_troop.marketing.id
  game_id  = "caputchin/games/leaf"
}

# Game-axis preset. Tier depends on the axis: configuration is Solo+,
# skin / locale are Alpha+. game_id references the customized_game above.
resource "caputchin_white_label_preset" "leaf_hard" {
  troop_id = caputchin_troop.marketing.id
  game_id  = caputchin_customized_game.leaf.game_id
  axis     = "configuration"
  name     = "hard"
  values = jsonencode({
    difficulty = "hard"
  })
}

# Per-site override. Overrides the troop baseline for one site key only.
# A widget-shell (locale) preset here (no game_id), so it needs no game
# registration.
resource "caputchin_site_key" "blog" {
  name     = "blog-prod"
  troop_id = caputchin_troop.marketing.id
}

resource "caputchin_white_label_preset" "blog_en" {
  site_id = caputchin_site_key.blog.id
  axis    = "locale"
  name    = "en"
  values = jsonencode({
    _lang      = "en"
    main_title = "Verify you are human"
  })
}
