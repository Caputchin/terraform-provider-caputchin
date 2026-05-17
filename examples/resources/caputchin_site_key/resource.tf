resource "caputchin_team" "marketing" {
  name = "marketing"
}

resource "caputchin_site_key" "blog" {
  name    = "blog-prod"
  team_id = caputchin_team.marketing.id
}

# Hand the secret to your backend's secrets manager (Vault / Doppler /
# Infisical / AWS Secrets Manager / etc.). The secret is returned only at
# creation time — rotate the key if you lose it.
output "blog_site_secret" {
  value     = caputchin_site_key.blog.secret
  sensitive = true
}

output "blog_site_public_key" {
  value = caputchin_site_key.blog.key
}
