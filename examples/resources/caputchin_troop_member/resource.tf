resource "caputchin_troop" "marketing" {
  name = "marketing"
}

resource "caputchin_troop_member" "alice" {
  troop_id = caputchin_troop.marketing.id
  email    = "alice@example.com"

  perms = {
    create = true
    edit   = true
    read   = true
    manage = false
  }

  # Grant access across every site in the troop.
  scope = {
    kind = "full"
  }
}
