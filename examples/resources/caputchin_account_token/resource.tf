# Mint a troop-scoped PAT (default type). Mint is free under the per-troop-axis
# seat model; the seat is claimed at attach time via caputchin_troop_pat.
resource "caputchin_account_token" "ci_prod" {
  name = "ci-prod"
}

# Account-PAT: master, free, capped at 1 active per account.
resource "caputchin_account_token" "automation" {
  name = "ops-automation"
  type = "account"
}

# In-place rotation (ADR-0056). Bump secret_version to issue a fresh credential
# without destroying the resource. The token id and name stay stable; the
# prefix rotates together with the secret half (refer to tokens across rotation
# by id or name, not prefix). Any troop attachments survive untouched. The
# replacement value lands in `secret` and is shown ONCE in the apply output.
resource "caputchin_account_token" "ci_prod_rotating" {
  name           = "ci-prod"
  secret_version = 1 # increment to rotate
}

# Hand the secret to your secrets manager. Returned ONCE at create time and
# on every rotation; pipe straight to a secret store.
output "ci_prod_token" {
  value     = caputchin_account_token.ci_prod.secret
  sensitive = true
}
