resource "caputchin_troop" "marketing" {
  name = "marketing"
}

resource "caputchin_site_key" "blog" {
  name     = "blog-prod"
  troop_id = caputchin_troop.marketing.id
}

# Per-site configuration. Fields you omit retain the server-side defaults
# derived from your plan tier; only set what you want to override.
resource "caputchin_site_config" "blog" {
  site_id = caputchin_site_key.blog.id

  cors_origins = ["https://blog.example.com"]

  pow_difficulty           = 4
  pow_challenge_count      = 50
  obfuscation_level        = 6
  block_automated_browsers = true
  ratelimit_max            = 100
}
