# Releasing

End-to-end runbook for releasing `terraform-provider-caputchin` to the Terraform
and OpenTofu registries. Releases are zero-touch: you merge a release PR and the
pipeline does the rest.

## How it works

The [`release.yml`](.github/workflows/release.yml) workflow runs on every push to
`main`:

1. **release-please** reads the Conventional Commits since the last release and
   keeps an open `chore(main): release vX.Y.Z` PR up to date, with the computed
   version bump and a `CHANGELOG.md` entry. Pre-1.0, `feat` and breaking changes
   bump the minor; `fix` bumps the patch (see [Versioning](#versioning)).
2. Merging that PR makes release-please create the `vX.Y.Z` git tag and a GitHub
   Release whose body is generated from the commits.
3. In the same workflow run, the `goreleaser` job checks out the new tag, builds
   every supported OS/arch, GPG-signs the `SHA256SUMS` checksum file, and appends
   all artifacts (zips, `SHA256SUMS`, `SHA256SUMS.sig`, `manifest.json`) to that
   release.
4. The Terraform and OpenTofu registries poll the repo, fetch the release
   artifacts, verify the signature against the public key on file, and index the
   version.

You never tag by hand. The only manual action per release is merging the release
PR.

## One-time setup

These must all be in place before the first release PR is merged, or the
`goreleaser` job fails (no signing key) and the registry never indexes a build.

### 1. Generate a release-signing GPG key

Run locally. The private key never leaves your machine except to land in the
repo secret (and your password manager / Infisical as the canonical copy).

```sh
gpg --full-generate-key
# RSA and RSA, 4096 bits, does not expire (0)
# Name:    Caputchin Releases
# Email:   releases@caputchin.com
# Comment: terraform-provider-caputchin signing
# Passphrase: choose a strong one and store it
```

Note the 40-char fingerprint and export both halves:

```sh
gpg --list-secret-keys --keyid-format=long --with-colons releases@caputchin.com \
  | awk -F: '/^fpr:/{print $10; exit}'

gpg --armor --export-secret-keys releases@caputchin.com > gpg-private-key.asc   # private
gpg --armor --export releases@caputchin.com > gpg-public-key.asc                # public
```

### 2. Add the signing secrets to this repository

`Settings > Secrets and variables > Actions`:

| Secret | Value |
| --- | --- |
| `GPG_PRIVATE_KEY` | Full contents of `gpg-private-key.asc`, including the `BEGIN`/`END PGP PRIVATE KEY BLOCK` lines |
| `PASSPHRASE` | The passphrase chosen in step 1 |

`GITHUB_TOKEN` is injected by Actions automatically. Keep the canonical copy of
the key in Infisical; [`scripts/release-secrets.md`](scripts/release-secrets.md)
has the storage detail, secret-name mapping, and rotation procedure.

### 3. Grant the release App and its secrets to this repository

The zero-touch trigger uses the org-level `caputchin-release` GitHub App and the
`RELEASE_APP_ID` / `RELEASE_APP_PRIVATE_KEY` org secrets (shared with the SDK
release pipeline). Both are scoped to selected repositories, so add
`terraform-provider-caputchin` to:

- the `caputchin-release` App installation (`Org settings > GitHub Apps >
  caputchin-release > Configure > Repository access`), and
- the selected-repository list of each `RELEASE_APP_*` org secret
  (`Org settings > Secrets and variables > Actions`).

### 4. Register the public key and the provider on the registries

**Terraform Registry**

1. Sign in at <https://registry.terraform.io/sign-in> with the GitHub account
   that administers the `Caputchin` org.
2. `User Settings > Signing Keys > New GPG Key` and paste `gpg-public-key.asc`.
3. `Publish > Provider`, select the `Caputchin` org, select
   `terraform-provider-caputchin`, confirm. The first scan may find no versions;
   that is fine. It re-scans after the first tag is pushed.

**OpenTofu Registry**

Submit the public key plus provider registration as a metadata PR against
`opentofu/registry`. See
[ADR-0042](../docs/adr/0042-publish-terraform-provider-to-both-registries-and-rename-repo.md).

### 5. Confirm the repo is public

Both registries can only fetch public repositories.

### Clean up the local key file

Once the first release is signed and the registry shows the green signed badge,
scrub the exported private key. Plain `rm` does not guarantee unrecoverability on
modern SSDs:

```sh
shred -u gpg-private-key.asc
```

## Cutting a release

1. Land your work on `main` with Conventional Commit messages (`feat`, `fix`,
   `feat!`, etc.). Each push refreshes the open release PR.
2. When ready to ship, review the release PR: confirm the proposed version and
   the `CHANGELOG.md` diff match what you expect.
3. Merge the release PR. The pipeline tags, builds, signs, publishes, and the
   registries index the version (usually within about 10 minutes).

The very first release PR will propose `v0.1.0` (the accumulated `feat` history
bumps `0.0.0` to `0.1.0`). Verify that before merging.

## Post-release verification

In order, each step depends on the previous:

| Layer | Check |
| --- | --- |
| Tag created | `git ls-remote --tags origin vX.Y.Z` returns a SHA |
| Workflow succeeded | `gh run list --workflow=release.yml -L 1` is green |
| Release populated | `gh release view vX.Y.Z` lists the OS/arch zips, `SHA256SUMS`, `SHA256SUMS.sig`, `manifest.json` |
| Signature valid | `gpg --verify <SHA256SUMS.sig> <SHA256SUMS>` reports a good signature |
| Registry indexed | <https://registry.terraform.io/providers/caputchin/caputchin/X.Y.Z> renders the docs |
| End to end | A throwaway config with `source = "caputchin/caputchin"`, `terraform init`, `terraform plan` against a real account |

## Versioning

Pre-1.0 (`v0.x`) is the dogfooding window per [CLAUDE.md](CLAUDE.md): schema
breaking changes are allowed without a major bump. release-please is configured
(`bump-minor-pre-major`) so that while below 1.0, `feat` and `feat!` bump the
minor and `fix` bumps the patch. After `v1.0.0`, mark schema breaks with the `!`
Conventional Commit marker so release-please bumps the major.

## Failure modes

### Tagged but the goreleaser job failed with a GPG error

`GPG_PRIVATE_KEY` is malformed (missing `BEGIN`/`END` lines or mangled line
endings) or `PASSPHRASE` does not match the key. Re-export and re-paste both
secrets, then re-run the failed workflow.

### Registry says signature verification failed

The public key in the registry settings is for a different key than the one CI
signs with. Re-export the public key from the same keyring as the private key,
replace it in the registry, and trigger a re-scan with the next release.

### Release PR never appears

The `caputchin-release` App or the `RELEASE_APP_*` secrets are not granted to
this repo (one-time setup steps 2 and 3), or there are no releasable commits
(only `docs`/`chore`/`ci` since the last release).

### A version is wrong after publishing

You cannot move a tag once a registry has indexed it, and a published version is
immutable. Ship the corrected code as the next patch and let version constraints
carry consumers forward.
