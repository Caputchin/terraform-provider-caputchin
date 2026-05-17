# Release secrets setup

Before tagging the first release of `terraform-provider-caputchin`, generate a release-signing GPG key and provision it as GitHub Actions secrets on the repository. The key is stored in Infisical (`infra-core` project) as the canonical copy and mirrored into Actions for the `release` workflow to consume.

## Generate a release-signing GPG key

Run locally — the private key never leaves your machine except to land in Infisical and the GitHub Actions secret store.

```sh
gpg --full-generate-key
# Choose: RSA and RSA, 4096 bits, 0 (does not expire), confirm
# Name: Caputchin Releases
# Email: releases@caputchin.com
# Comment: terraform-provider-caputchin signing
# Passphrase: store in Infisical alongside the key
```

Then export the private key in ASCII-armored form:

```sh
gpg --armor --export-secret-keys releases@caputchin.com > gpg-private-key.asc
```

## Store in Infisical (`infra-core`)

Add three secrets to the `infra-core` project:

| Key | Value | Notes |
|---|---|---|
| `TF_PROVIDER_GPG_PRIVATE_KEY` | full contents of `gpg-private-key.asc` | Includes the `-----BEGIN/END PGP PRIVATE KEY BLOCK-----` markers |
| `TF_PROVIDER_GPG_PASSPHRASE` | the passphrase you chose | Required to decrypt the key in CI |
| `TF_PROVIDER_GPG_FINGERPRINT` | output of `gpg --list-secret-keys --keyid-format=long releases@caputchin.com` | 40-char fingerprint with no spaces |

## Mirror into GitHub Actions secrets

On the `Caputchin/terraform-provider-caputchin` repository, **Settings → Secrets and variables → Actions**:

| Secret name | Value source |
|---|---|
| `GPG_PRIVATE_KEY` | `TF_PROVIDER_GPG_PRIVATE_KEY` from Infisical |
| `PASSPHRASE` | `TF_PROVIDER_GPG_PASSPHRASE` from Infisical |

`GITHUB_TOKEN` is provisioned automatically by Actions.

## Register the public key with the Terraform Registry

Export the public key:

```sh
gpg --armor --export releases@caputchin.com
```

Paste the output into the Terraform Registry publisher page (Account → Signing Keys → New GPG Key). The Registry uses this public key to verify the SHA256SUMS signature attached to each release.

## Register with the OpenTofu Registry

The OpenTofu Registry follows the same flow — submit the public key in the metadata PR alongside the provider registration. See [ADR-0042](../../docs/adr/0042-publish-terraform-provider-to-both-registries-and-rename-repo.md).

## Rotate the key

Generate a new key, publish the new public key to both registries BEFORE retiring the old one, then update the Infisical and Actions secrets. Releases tagged after the swap are signed with the new key; older releases retain the old signature (registries cache by tag).

## Delete after first successful release

Once `v0.1.0` is signed and the Registry pages render with the green "signed" badge, scrub `gpg-private-key.asc` from your local disk. The Infisical copy is the only persistent storage.

Use `shred` rather than plain `rm` — modern SSDs (wear leveling) and journaled filesystems mean `rm` alone does not guarantee unrecoverability:

```sh
shred -u gpg-private-key.asc
```
