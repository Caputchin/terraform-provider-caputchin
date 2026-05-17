data "caputchin_team" "existing" {
  id = "team_abc123"
}

output "team_name" {
  value = data.caputchin_team.existing.name
}
