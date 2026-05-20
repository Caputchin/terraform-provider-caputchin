data "caputchin_sites" "all" {}

output "site_ids" {
  value = [for s in data.caputchin_sites.all.sites : s.id]
}
