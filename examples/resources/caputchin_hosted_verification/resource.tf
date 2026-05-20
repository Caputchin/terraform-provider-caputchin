resource "caputchin_troop" "marketing" {
  name = "marketing"
}

resource "caputchin_site_key" "blog" {
  name     = "blog-prod"
  troop_id = caputchin_troop.marketing.id
}

# Hosted verification is Alpha tier or above. Set destinations only when
# `enabled = true`; leave them null to remove a destination.
resource "caputchin_hosted_verification" "blog" {
  site_id     = caputchin_site_key.blog.id
  enabled     = true
  webhook_url = "https://example.com/caputchin-webhook"
  email_to    = "ops@example.com"
}
