data "caputchin_troop" "existing" {
  id = "troop_abc123"
}

output "troop_name" {
  value = data.caputchin_troop.existing.name
}
