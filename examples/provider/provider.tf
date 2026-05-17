terraform {
  required_providers {
    caputchin = {
      source  = "caputchin/caputchin"
      version = "~> 0.1"
    }
  }
}

variable "caputchin_management_token" {
  type      = string
  sensitive = true
}

provider "caputchin" {
  # Endpoint defaults to https://api.caputchin.com.
  # Override for local development against a wrangler-dev worker:
  #   endpoint = "http://localhost:8787"
  management_token = var.caputchin_management_token
}
