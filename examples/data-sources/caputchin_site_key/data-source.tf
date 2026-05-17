data "caputchin_site_key" "existing" {
  id = "site_abc123"
}

output "site_public_key" {
  value = data.caputchin_site_key.existing.key
}
