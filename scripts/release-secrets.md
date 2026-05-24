# Release secrets storage

The canonical end-to-end release runbook is [`RELEASING.md`](../RELEASING.md)
(zero-touch flow, one-time setup, verification, failure modes). This file covers
only the **secret-storage detail**: where the release-signing GPG key lives
canonically (Infisical), the secret-name mapping into GitHub Actions, and key
rotation.

Provision all of this before the first release PR is merged, or the `goreleaser`
job fails with no signing key and the registries never index a build.

## Canonical storage in Infisical (`infra-core`)

Generate the key per [`RELEASING.md` step 1](../RELEASING.md#1-generate-a-release-signing-gpg-key),
then store the canonical copy in the `infra-core` Infisical project:

| Key | Value | Notes |
|---|---|---|
| `TF_PROVIDER_GPG_PRIVATE_KEY` | full contents of `gpg-private-key.asc` | Includes the `-----BEGIN/END PGP PRIVATE KEY BLOCK-----` markers |
| `TF_PROVIDER_GPG_PASSPHRASE` | the passphrase you chose | Required to decrypt the key in CI |
| `TF_PROVIDER_GPG_FINGERPRINT` | `gpg --list-secret-keys --keyid-format=long info@caputchin.com` | 40-char fingerprint, no spaces |

## Mirror into GitHub Actions secrets

On `Caputchin/terraform-provider-caputchin`, **Settings → Secrets and variables → Actions**:

| Repo secret | Value source |
|---|---|
| `GPG_PRIVATE_KEY` | `TF_PROVIDER_GPG_PRIVATE_KEY` from Infisical |
| `PASSPHRASE` | `TF_PROVIDER_GPG_PASSPHRASE` from Infisical |

`GITHUB_TOKEN` is provisioned automatically by Actions. The release-please trigger
also needs the org-level `caputchin-release` GitHub App plus the `RELEASE_APP_ID`
and `RELEASE_APP_PRIVATE_KEY` org secrets granted to this repo; those are shared
with the SDK pipeline and managed at the org level, not stored here. See
[`RELEASING.md` step 3](../RELEASING.md#3-grant-the-release-app-and-its-secrets-to-this-repository).

## Rotate the key

Generate a new key, publish the new public key to both registries BEFORE retiring
the old one, then update the Infisical copy and re-mirror the GitHub Actions
secrets. Releases tagged after the swap are signed with the new key; older
releases retain the old signature (registries cache by tag).
