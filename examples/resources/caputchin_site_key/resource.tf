resource "caputchin_troop" "marketing" {
  name = "marketing"
}

resource "caputchin_site_key" "blog" {
  name     = "blog-prod"
  troop_id = caputchin_troop.marketing.id

  # Bump this to trigger an in-place secret rotation. The
  # site `id` and public `key` stay the same; only the `secret`
  # changes. Initial Create ignores the value (the mint already
  # returns a fresh secret).
  secret_version = 1
}

# Scheduled rotation: time_rotating + terraform_data intermediary +
# replace_triggered_by (the direct time_rotating + replace_triggered_by
# path is broken upstream per terraform-provider-time issue #118, open
# since 2022). When the trigger fires, rotation_triggers forces full
# replacement of the site key (new id, new key, new secret); for in-
# place rotation, bump secret_version instead.
resource "time_rotating" "schedule" {
  rotation_days = 90
}

resource "terraform_data" "rotation_stamp" {
  triggers_replace = {
    stamp = time_rotating.schedule.rotation_rfc3339
  }
}

resource "caputchin_site_key" "blog_scheduled" {
  name     = "blog-scheduled-prod"
  troop_id = caputchin_troop.marketing.id

  rotation_triggers = {
    schedule = terraform_data.rotation_stamp.output.stamp
  }
}

# Hand the secret to your secrets manager. Sensitive but lands in
# Terraform state; treat the state file as secret-bearing.
output "blog_site_secret" {
  value     = caputchin_site_key.blog.secret
  sensitive = true
}

output "blog_site_public_key" {
  value = caputchin_site_key.blog.key
}
