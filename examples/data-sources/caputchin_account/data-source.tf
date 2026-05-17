data "caputchin_account" "self" {}

output "account_email" {
  value = data.caputchin_account.self.email
}

output "account_id" {
  value = data.caputchin_account.self.id
}
