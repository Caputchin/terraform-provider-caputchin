# Changelog

## 1.0.0 (2026-05-25)


### ⚠ BREAKING CHANGES

* **site:** support in-place rotation via secret_version and replacement via rotation_triggers
* rename caputchin_team to caputchin_troop, team_id to troop_id

### Features

* **account:** add caputchin_account_token resource ([849f54c](https://github.com/Caputchin/terraform-provider-caputchin/commit/849f54cd695d6f637da40458363feb437b5912ee))
* **account:** caputchin_account data source + caputchin_site_stats ([492e28a](https://github.com/Caputchin/terraform-provider-caputchin/commit/492e28a3140f8640c4f2781cb14028456cbc278a))
* **provider:** http client + provider config + pat auth wiring ([62829c5](https://github.com/Caputchin/terraform-provider-caputchin/commit/62829c57d61dc0c23ddce4afc46f9406e8a08c6b))
* **provider:** white-label + game-customization override resources (ADR-0061) ([eb91923](https://github.com/Caputchin/terraform-provider-caputchin/commit/eb91923283cfabadfc463cceae3ce7b3a3d97198))
* rename caputchin_team to caputchin_troop, team_id to troop_id ([b065f96](https://github.com/Caputchin/terraform-provider-caputchin/commit/b065f96277c7757527b215658f5c7072d63eb065))
* **site:** add caputchin_hosted_verification resource and list data sources ([323e74f](https://github.com/Caputchin/terraform-provider-caputchin/commit/323e74f362ff04e6928cb7cdf8c6984f562e74ab))
* **site:** caputchin_site_config singleton with nullable patch semantics ([da8c99b](https://github.com/Caputchin/terraform-provider-caputchin/commit/da8c99be705ab7b60fc65cc90b43afa3f3ce2cd1))
* **site:** caputchin_site_key resource + data source with sensitive secret ([f26cc3f](https://github.com/Caputchin/terraform-provider-caputchin/commit/f26cc3f7412059b7cbf95830b8376eb5902a7f8e))
* **site:** support in-place rotation via secret_version and replacement via rotation_triggers ([7be2e87](https://github.com/Caputchin/terraform-provider-caputchin/commit/7be2e877c6d3a0e654a7d92dc26edf35297070df))
* **team:** caputchin_team resource + data source ([09bb08c](https://github.com/Caputchin/terraform-provider-caputchin/commit/09bb08c8fe8d060922c03fea61532421b6516450))
* **token:** in-place rotation via secret_version on caputchin_account_token (ADR-0056) ([4780c43](https://github.com/Caputchin/terraform-provider-caputchin/commit/4780c43f83925c3f277c6973892c85db96332111))
* **troop:** add caputchin_troop_pat and caputchin_troop_member resources ([0e91679](https://github.com/Caputchin/terraform-provider-caputchin/commit/0e91679d1bf14cb29e1f69d1b2709d5e11d89078))
* **troop:** expose can_manage on troop data sources and resource ([b6e48d5](https://github.com/Caputchin/terraform-provider-caputchin/commit/b6e48d50209c09625650afc412f7801813d9a876))
* **troop:** surface owner_email on the troop resource + data sources ([3e93ccf](https://github.com/Caputchin/terraform-provider-caputchin/commit/3e93ccf86209b5c46ce12d17e6fea659e0532136))


### Bug Fixes

* **docs:** correct MPL trademark non-grant section to §2.3 (was §1.10 + §3.4) ([3374e66](https://github.com/Caputchin/terraform-provider-caputchin/commit/3374e6615642c254aa356af878ee256db52b155c))
* **provider:** add computed id to override resources for import + state tracking ([cc3fc10](https://github.com/Caputchin/terraform-provider-caputchin/commit/cc3fc109574203485c54187ffc172b3b049be8e7))
* **provider:** default endpoint to apex caputchin.com/api (api subdomain not live) ([ff624f4](https://github.com/Caputchin/terraform-provider-caputchin/commit/ff624f47c5bf96bfdcaa141e9ee791db5334584c))
* **provider:** drift-proof preset values; game-axis overrides now require a registered game ([830a6bd](https://github.com/Caputchin/terraform-provider-caputchin/commit/830a6bd28fa0726787b2d7f035badc8347998f8c))

## Changelog

All notable changes to this provider are recorded here. This file is
maintained automatically by release-please from Conventional Commit messages;
do not edit it by hand.
