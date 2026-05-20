resource "caputchin_troop" "marketing" {
  name = "marketing"
}

output "marketing_troop_id" {
  value = caputchin_troop.marketing.id
}
