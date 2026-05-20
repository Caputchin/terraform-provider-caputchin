resource "caputchin_troop" "marketing" {
  name = "marketing"
}

# Mint a troop-PAT once. The same token may be attached to multiple
# troops via separate caputchin_troop_pat resources.
resource "caputchin_account_token" "ci" {
  name = "ci-prod"
  # type = "troop"  # default
}

resource "caputchin_troop_pat" "ci_marketing" {
  troop_id = caputchin_troop.marketing.id
  pat_id   = caputchin_account_token.ci.id

  perms = {
    create = true
    edit   = true
    read   = true
    manage = false
  }

  # Restrict the PAT to a specific subset of sites in this troop.
  scope = {
    kind     = "partial"
    site_ids = [caputchin_site_key.blog.id]
  }
}

resource "caputchin_site_key" "blog" {
  name     = "blog-prod"
  troop_id = caputchin_troop.marketing.id
}
