# Mint a troop-scoped PAT (default type). Consumes one PAT seat per
# ADR-0028. Attach to a specific troop with caputchin_troop_pat.
resource "caputchin_account_token" "ci_prod" {
  name = "ci-prod"
}

# Account-PAT — master, free, capped at 1 active per account.
resource "caputchin_account_token" "automation" {
  name = "ops-automation"
  type = "account"
}

# Hand the secret to your secrets manager. Returned ONCE at create time —
# rotate (destroy + recreate) the resource to issue a new value.
output "ci_prod_token" {
  value     = caputchin_account_token.ci_prod.secret
  sensitive = true
}
