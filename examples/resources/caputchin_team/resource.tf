resource "caputchin_team" "marketing" {
  name = "marketing"
}

output "marketing_team_id" {
  value = caputchin_team.marketing.id
}
