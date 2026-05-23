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
  # Endpoint defaults to https://api.caputchin.com. Override with the
  # `endpoint` attribute or the CAPUTCHIN_ENDPOINT environment variable.
  management_token = var.caputchin_management_token
}
