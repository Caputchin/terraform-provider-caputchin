# Caputchin trademark and brand-usage guidelines

This file is a copy of the canonical guidelines published at <https://caputchin.com/legal/trademark>.

**Status:** "Caputchin" is currently an unregistered mark, claimed under common law via continuous use in commerce. Wordmark registration in EU + US is planned but has not been filed yet. Today, our enforcement layer is common-law trademark + these guidelines + the trademark-non-grant clauses in MPL-2.0 §1.10 + §3.4. We can send cease-and-desist letters and file platform takedowns; we cannot yet point to a registration certificate.

## Underlying principle

Trademark protects users from being misled about who builds and operates Caputchin. It does not give Caputchin veto over every use of the word "caputchin". The distinguishing line throughout these guidelines is between **marketed product names** (what a reasonable user reads as the brand of the thing they're using) and **code identifiers** (npm package names, GitHub repo names, CLI binary names, Terraform Registry namespaces — read by developers who already know they are picking a third-party tool). Marketed product names that incorporate "Caputchin" risk implying first-party origin; code identifiers that incorporate "caputchin" are fine when they're plainly third-party.

## Marks covered

The following are marks of Caputchin (claimed under common law; registration pending):

| Mark | Form |
|---|---|
| `Caputchin` | Wordmark — the name of the product. |
| The capuchin-monkey logo | Figurative mark. |
| `@caputchin` | npm scope under which Caputchin first-party packages publish. |
| `caputchin/caputchin` | Terraform Registry and OpenTofu Registry namespace for the official provider. |
| `caputchin-game` | GitHub topic that gates marketplace discovery. |
| `caputchin.com` and subdomains | Primary product domain. |

Stylized writing (`caputchin`, `CAPUTCHIN`, `Caputchin.com`) all refer to the same mark. References below to "the marks" mean all of them.

## What you can do without asking

You do not need permission for any of the following:

- **Factual reference.** Calling Caputchin by name when describing what it is, comparing it to alternatives, writing tutorials, posting screenshots, or referencing the product in blog posts, talks, papers, or social media. Trademark law does not restrict accurate factual use.
- **Integration attribution.** Adding "powered by Caputchin", "uses Caputchin", "Caputchin-compatible", "works with Caputchin", or "integrates with Caputchin" to a product, page, or README, provided the description is accurate. Linking the attribution to <https://caputchin.com> is encouraged.
- **Source-redistribution attribution.** Preserving copyright lines, per-file MPL headers, and `TRADEMARK.md` files when redistributing source code under our license.
- **Code identifiers — third-party packages, repos, CLI commands, and community Terraform providers.** Third-party plugins, extensions, helpers, adapters, tools, and community Terraform providers may use "caputchin" in their npm package name, GitHub repository name, CLI binary name, or Terraform Registry namespace when all of the following are true: (a) the project is plainly third-party in its README and package description, (b) it does not claim "official", "certified", "endorsed", "by Caputchin", or similar status, (c) it does not publish under the `@caputchin` npm scope (locked to first-party), (d) it does not adopt the capuchin-monkey logo as its primary branding, and (e) it does not publish under the `caputchin/caputchin` Terraform Registry namespace (locked to first-party). Examples permitted: `caputchin-react-helper`, a community provider published as `someorg/caputchin-experimental`, `caputchin-stripe-bridge`, the GitHub repo `johndoe/terraform-provider-caputchin-extras`. A brief non-affiliation note in the README (`"Not affiliated with or endorsed by Caputchin."`) is requested but not legally required.
- **Plugin and tooling naming, suffix and descriptive forms.** Naming a third-party plugin or tool `X for Caputchin`, `X (Caputchin)`, `X — Caputchin integration`, `X — works with Caputchin`, or `X — compatible with Caputchin`. The third-party name comes first; "Caputchin" appears only as a descriptor of what the plugin connects to.

## What we ask you to get permission for

We ask that the following uses get our written permission first. Some are backed by trademark law and false-advertising law independently of registration status (logo modification, implying endorsement, advertising for competing products, registering confusingly similar marks); the rest are policy we are publishing now and will formalize on registration.

- **Marketed product, service, or company names that incorporate "Caputchin" or "Caputchin-*" as the brand label.** This is the case where a reasonable end user would read "Caputchin" as the source of the thing. Examples that need permission: a product marketed as `Caputchin Pro`, `Caputchin Plus`, `Caputchin Enterprise`, `Caputchin Cloud`, `Caputchin Studio`, or a company named `Caputchin Solutions Inc.`. The restriction is on the **marketing label**, not on the code identifier; a community provider published as `someorg/caputchin-experimental` is fine (see code-identifier rule above), but marketing that same provider as the product `Caputchin Experimental` needs permission.
- **Standalone brand-looking domain names.** Registering `caputchin.io`, `caputchin-official.com`, `get-caputchin.com`, `caputchin-pro.com`, `caputchinclone.io`, or any domain where "caputchin" forms the primary portion of the second-level domain. Paths under third-party domains are unaffected — `johnsmith.dev/caputchin-guide`, `mysite.com/blog/caputchin-tutorial`, and similar are factual reference, no permission needed.
- **The `@caputchin` npm scope** and **the `caputchin/caputchin` Terraform / OpenTofu Registry namespace.** The npm scope is reserved for first-party packages; the Registry namespace is reserved for the official provider. Community providers publish under their own Registry namespace.
- **Logo modification or appropriation as primary branding.** Distorting, recoloring, redrawing, or otherwise modifying the capuchin-monkey logo. Using the unmodified logo as the primary visual mark of a third-party product. Small "powered by Caputchin" attribution use of the unmodified logo at integration boundaries is fine without asking.
- **Implying endorsement, partnership, or official status when none exists.** Phrases like "official Caputchin partner", "certified by Caputchin", "endorsed by Caputchin", "authorized Caputchin reseller", "Caputchin official integration" — and on the Terraform Registry side, claims of being "the official" or "the Caputchin-published" provider — are reserved for parties Caputchin has explicitly designated.
- **Using the marks in advertising, marketing, or promotional contexts for a competing product.** Even where individual elements above would otherwise be fine, using "Caputchin" to drive comparison-shopping traffic toward a substitute CAPTCHA / verification product needs prior agreement.
- **Trademark, service mark, or domain registration** of "Caputchin" or confusingly similar variants in any jurisdiction.

The MPL-2.0 license under which this code is distributed grants no rights to use the marks (§1.10 and §3.4).

## Code identifier vs marketed product name — disambiguation

The distinction is the surface a reasonable user reads as "the brand of this product":

| Surface | Treated as | Example |
|---|---|---|
| Terraform Registry namespace (community provider) | Code identifier | `someorg/caputchin-experimental` — fine |
| GitHub repo name | Code identifier | `johndoe/terraform-provider-caputchin-extras` — fine |
| npm package name | Code identifier | `caputchin-react-helper` — fine |
| CLI binary name | Code identifier | `caputchin-deploy` (third-party tool) — fine |
| README title / hero text / marketing label | Marketed product name | "Caputchin Experimental Provider" as a product brand — please ask; "experimental provider for Caputchin (third-party)" — fine |
| Logo / favicon / wordmark used as primary branding | Marketed product name | Adopting the capuchin-monkey logo on a landing page — please ask |
| Domain name (second-level) | Marketed product name | `caputchin-bridge.com` — please ask |

When the same project has both — e.g., Registry namespace `someorg/caputchin-experimental` whose README hero says "Caputchin Experimental, a third-party fork of the official provider" — the namespace is fine; the README hero is a marketed name and should not lead with "Caputchin" as the product brand.

## Asking for permission

Submit requests to `info@caputchin.com` with:

- Who you are (individual, company, project).
- The proposed use of the marks (name, context, scope, geography).
- A mockup or sample where possible.

Response target is two weeks. We say yes more often than no for honest integrations; we say no consistently for naming that risks user confusion about who builds and operates Caputchin.

## When in doubt

If the proposed use would cause a reasonable user to believe Caputchin built, operated, endorsed, or stands behind a third-party product, please ask first. If the use is plainly informational, integrative, factual, or is a code identifier in a context where the audience knows they are picking a third-party tool, go ahead — no need to ask.
