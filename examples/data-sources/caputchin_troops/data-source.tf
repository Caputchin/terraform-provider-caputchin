data "caputchin_troops" "all" {}

output "shared_troop_ids" {
  value = [for t in data.caputchin_troops.all.troops : t.id if t.kind == "shared"]
}
