terraform {
  required_providers {
    caputchin = {
      source  = "caputchin/caputchin"
      version = "~> 1.0"
    }
  }
}

variable "caputchin_management_token" {
  type      = string
  sensitive = true
}

provider "caputchin" {
  # Endpoint defaults to https://caputchin.com/api. Override with the
  # `endpoint` attribute or the CAPUTCHIN_ENDPOINT environment variable.
  management_token = var.caputchin_management_token
}
