# terraform-provider-caputchin

Official Terraform / OpenTofu provider for the [Caputchin](https://caputchin.com) management API.

Manage Caputchin troops, site keys, per-site configuration, and read account / stats data from your Terraform or OpenTofu configurations.

| Registry | Source address |
|---|---|
| Terraform Registry | `caputchin/caputchin` |
| OpenTofu Registry | `caputchin/caputchin` |

Both consume the **same binary release**. The provider implements the standard Terraform plugin protocol (gRPC v6).

## Requirements

| Requirement | Version |
|---|---|
| Terraform | 1.5+ |
| OpenTofu | 1.6+ |
| Caputchin management token | account-PAT or troop-PAT (see [management API docs](https://caputchin.com/docs/management-api)) |

## Quick start

```hcl
terraform {
  required_providers {
    caputchin = {
      source  = "caputchin/caputchin"
      version = "~> 0.1"
    }
  }
}

provider "caputchin" {
  # Endpoint defaults to https://api.caputchin.com.
  # Override for local development against a wrangler-dev worker.
  # endpoint = "http://localhost:8787"

  # Management token. Also reads CAPUTCHIN_MANAGEMENT_TOKEN env var.
  management_token = var.caputchin_pat
}

resource "caputchin_troop" "marketing" {
  name = "marketing"
}

resource "caputchin_site_key" "blog" {
  name     = "blog-prod"
  troop_id = caputchin_troop.marketing.id
}

resource "caputchin_site_config" "blog" {
  site_id      = caputchin_site_key.blog.id
  cors_origins = ["https://blog.example.com"]

  pow_difficulty           = 4
  pow_challenge_count      = 50
  obfuscation_level        = 6
  block_automated_browsers = true
  ratelimit_max            = 100
}

output "site_secret" {
  value     = caputchin_site_key.blog.secret
  sensitive = true
}
```

## Resources and data sources

| Type | Name | Purpose |
|---|---|---|
| Resource | `caputchin_troop` | Tenant boundary. Create / read / rename / delete shared troops |
| Resource | `caputchin_site_key` | Site key with rotatable secret. `troop_id` is immutable (replace-on-change) |
| Resource | `caputchin_site_config` | Per-site configuration (origin allowlist, PoW knobs, security filters, rate limit) |
| Data source | `caputchin_account` | Current account metadata. Account-PAT or session only |
| Data source | `caputchin_troop` | Look up an existing troop without owning it |
| Data source | `caputchin_site_key` | Look up an existing site key without owning it |
| Data source | `caputchin_site_stats` | Lifetime counters for a site key |

See [`docs/`](docs/) for the full schema reference (also rendered on the Terraform / OpenTofu Registry pages).

## Configuration

| Argument | Type | Default | Notes |
|---|---|---|---|
| `endpoint` | string | `https://api.caputchin.com` | Override for staging or local development |
| `management_token` | string (sensitive) | reads `CAPUTCHIN_MANAGEMENT_TOKEN` env if unset | Management API PAT (see [management API docs](https://caputchin.com/docs/management-api)) |

## OpenTofu lock file

When switching between `terraform` and `tofu` CLIs against the same configuration, **regenerate `.terraform.lock.hcl`**. The Terraform Registry and OpenTofu Registry sign the binary with different keys, so the lock hashes will differ:

```sh
rm .terraform.lock.hcl
tofu init           # or: terraform init
```

## Contributing

See [CLAUDE.md](CLAUDE.md) for the workspace conventions. Issues and pull requests welcome on GitHub.

## License

[MPL-2.0](LICENSE). Copyright (c) 2026 Caputchin. See [TRADEMARK.md](TRADEMARK.md) for brand-usage policy; the license does not grant trademark rights.
