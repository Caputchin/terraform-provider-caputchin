resource "caputchin_troop" "acme" {
  name = "acme"
}

# Register the custom game first. The run-artifact resource refuses with
# `game-not-registered` (409) until the install row exists.
resource "caputchin_customized_game" "my_game" {
  troop_id = caputchin_troop.acme.id
  game_id  = "customer/my-game"
  source   = "custom"
}

# Upload the headless replay artifact so this custom game becomes
# gate-eligible. The customer-hosted playable bundle keeps living on the
# customer's CDN (the widget's `game-src` attribute); only the deterministic
# replay artifact lives on Caputchin.
#
# The resource never reads local files at plan time, so editing run.js alone
# won't trigger an update. The `source_hash` below is the recommended drift
# signal: when any file content changes, the hash changes, and Terraform
# re-uploads on the next apply.
resource "caputchin_custom_game_run_artifact" "my_game" {
  troop_id     = caputchin_troop.acme.id
  game_id      = caputchin_customized_game.my_game.game_id
  run_path     = "${path.module}/dist/run.js"
  module_paths = ["${path.module}/dist/engine.wasm"]

  source_hash = sha256(join("", concat(
    [filesha256("${path.module}/dist/run.js")],
    [filesha256("${path.module}/dist/engine.wasm")],
  )))
}
