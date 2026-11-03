# Taxonomy file

The taxonomy is a YAML file read by `finops enrich validate`, `map` and `share`, by
`finops sense aws-cost --taxonomy`, `xops sense aws-accounts --taxonomy` and
`xops enrich lz-tree`. All of them read it through one loader (`record/taxonomy`). It belongs to
one tenant and describes the three trees of D-046, nested the way people read them: the business
structure, the projects with their Apps and Environments, and the landing zone with its OUs and
accounts. `finops/enrich/testdata/demo-taxonomy.yaml` is a working example and
`finops/enrich/testdata/nested-taxonomy.yaml` a second one with owner teams and aliases. To
start a new tenant, copy the starter template
[`templates/taxonomy.yaml`](../../templates/taxonomy.yaml).

```yaml
tenant: { code: testtnt, name: Test tenant }          # urn optional: urn:tnt:testtnt
business_units:
  - code: retail
    name: Retail
    cost_centers:
      - { code: ecom, name: E-commerce }               # owner_teams: read, not used
projects:
  - code: shop
    name: Online shop
    ref_urn_bu: urn:tnt:testtnt:bzu:retail             # required
    apps:
      - code: chk
        name: Checkout
        ref_urn_cct: urn:tnt:testtnt:bzu:retail:cct:ecom   # required
        environments:
          - { code: prod, name: Production }            # ref_urn_ont: read, not used
          - { code: nonprod, name: Non-production }
lzs:                                                    # or lz_conns; exactly one LZ
  - code: main
    name: Main
    csp: aws
    currency: USD
    management_account: "111111111111"
    default: { app: chk, environment: nonprod }         # optional, codes
    ou_paths:
      - code: checkout
        name: Checkout
        path: /Workloads/Checkout                       # or ref_id; at least one
        ref_id: ou-1a2b-3c4d5e6f                        # optional
        ref_urn_apps: [urn:tnt:testtnt:prj:shop:app:chk]
      - code: checkout-prod
        name: Prod
        path: /Workloads/Checkout/Prod
        ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:prod
        accounts:
          - code: shop-prod
            name: shop-prod
            ref_id: "200000000002"                      # required: the account id
            ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:prod
defaults: { app: chk, environment: nonprod }            # optional, codes
tags: { app: App, project: Project }                    # optional; these are the defaults
```

## Top-level sections

| Section | Purpose |
|---|---|
| `tenant` | Required: `{ code, name }`, optionally `urn`. The code must equal `--tenant`, or `validate`, `map`, `share` and `aws-cost --taxonomy` stop with an error naming both. Same id rule as `--tenant` (the code rule below). A bare `tenant: testtnt` is not read. |
| `business_units` | `{ code, name, cost_centers[] }`. A cost center is `{ code, name }` and may list `owner_teams` (`{ code, name }`), which are read and not used. |
| `projects` | `{ code, name, ref_urn_bu, apps[] }`. `ref_urn_bu` is the URN of the project's business unit. |
| `apps` (under a project) | `{ code, name, ref_urn_cct, environments[] }`. `ref_urn_cct` is the URN of the App's cost center. An Environment is `{ code, name }`; its `ref_urn_ont` is read and not used. |
| `lzs` or `lz_conns` | The tenant's one landing zone: `{ code, name, csp, currency, management_account, ref_id, default, ou_paths[] }`. A tenant has exactly one LZ (D-052); `validate` reports any other count as `taxonomy-lz-count`, and a file with both keys is an error naming both. |
| `ou_paths` (under the LZ) | An OU the taxonomy knows: `{ code, name, path, ref_id, ref_urn_apps[], ref_urn_env, accounts[] }`. It needs `path`, `ref_id` or both. |
| `accounts` (under an OU) | `{ code, name, ref_id, ref_urn_apps[], ref_urn_env }`. `ref_id` is the account id and is required. `contact` is read and not used. |
| `tags` | Optional `{ app, project }` keys for Cost Explorer (defaults `App` / `Project`). |
| `defaults` | Optional `{ app, environment }` (codes) when an account has no other rule. The LZ's `default` has the same shape and wins over it. |

Every node has a `code` and a `name`, and may have a `urn` and an `alias`. The `urn` is checked,
never read: it must equal the URN built from the codes (`taxonomy-urn-mismatch`). Any other key is
an error that names the file and the key. The earlier flat layout (`cost_centers`, `apps`,
`ou_links` or `accounts` at the top of the file) is no longer read; such a file fails with an
error naming the key.

## Codes and reserved ids (D-047, D-056)

Every `code` (business units, cost centers, projects, apps, their `environments`, and `lzs`) is 3
to 50 characters from letters, digits and `. + = @ _ -` (`^[A-Za-z0-9.+=@_-]{3,50}$`), for example
`retail`, `PRJ_Shop`, `b2c-eu` or `x@y=z+1`. The same rule binds `--tenant` and the file's
`tenant`. `validate` reports any other value (`qa` is too short, `a/b` and `a b` have a character
that is not allowed) as `taxonomy-invalid-id`.

- **Case.** Records hold the code as written (`project: PRJ_Shop`); the URN holds it in lowercase
  (`project_urn: urn:tnt:testtnt:prj:prj_shop`). So two codes in one list that differ only in
  case (`Shop` and `shop`) are `taxonomy-duplicate-code`, and App tags still match codes in any
  case. References are URNs (`ref_urn_*`), which are lowercase and match in any case; the
  `default` keys are codes. A code is read as written, never taken from a `urn`.
- **Where codes are unique** (compared in any case): business units, projects, Apps and LZs in the
  tenant; cost centers in their business unit; Environments in their App; OUs in their LZ;
  accounts in their OU. An `alias` is unique among nodes of one type in the tenant.
- **URNs** match `^[a-z0-9.:+=@_/-]{11,250}$`.

Three ids are **reserved**, in any case, for the system nodes `map` and `share` create. `validate`
rejects them as codes with `taxonomy-reserved-code` (`UNALLOC` too):

| Id | Meaning |
|---|---|
| `unalloc` | Not allocated: no LZ, Project, App, Environment, Business unit or Cost center matched. |
| `unspec` | Environment only: the account is known but no Environment rule matched. |
| `invalid` | App only: a tag named an App the account does not serve, or no App at all. It sits under Project `unalloc`. |

## URNs (D-046)

`map` and `share` name every node by a URN, written to the `*_urn` fields of `allocation` and
`cost` ([records.md](records.md)). A URN is `urn:tnt:<tenant>` followed by `:<type>:<id>` for each
level, where `<tenant>` is the file's `tenant` and each `<id>` is the code in lowercase (`PRJ_Shop`
gives `prj_shop`):

| Level | URN |
|---|---|
| Business unit | `urn:tnt:testtnt:bzu:retail` |
| Cost center | `urn:tnt:testtnt:bzu:retail:cct:ecom` (under the cost center's own `business_unit`) |
| Project | `urn:tnt:testtnt:prj:shop` |
| App | `urn:tnt:testtnt:prj:shop:app:chk` |
| Environment | `urn:tnt:testtnt:prj:shop:app:chk:env:prod` |
| Landing zone | `urn:tnt:testtnt:lzc:main` |
| OU | `urn:tnt:testtnt:lzc:main:org:workloads/checkout/prod` |
| Account | `urn:tnt:testtnt:lzc:main:org:workloads/checkout/prod:acc:200000000001` |
| Service (on `cost` only) | `<account URN>:svc:amazon-elastic-compute-cloud-compute` |

Names that come from the cloud (OU names, service names) are slugged (D-053). First CamelCase is
split: a word break goes between a lowercase letter or digit and the uppercase letter after it, and
between two uppercase letters when the second is followed by a lowercase letter. Then the name is
lowercased, every run of characters outside `[a-z0-9]` becomes one `-`, and the ends are trimmed.
An empty result is `unknown`.

| Name | Slug |
|---|---|
| `SharedServices` | `shared-services` |
| `NonProd` | `non-prod` |
| `AWSConfig` | `aws-config` |
| `AmazonEC2` | `amazon-ec2` |
| `Ec2Prod` | `ec2-prod` |
| `Hub2` | `hub2` |
| `Spoke-SIT` | `spoke-sit` |
| `Amazon Elastic Compute Cloud - Compute` | `amazon-elastic-compute-cloud-compute` |
| `---` | `unknown` |

An OU path becomes its slugged segments joined by `/`, without the leading `/`
(`/Workloads/Checkout/NonProd` is `workloads/checkout/non-prod`); the root `/` is `root`.

A slug can be shorter than a code may be: an OU named `IT` slugs to `it`, which is two characters.
It is kept as the node's `code` and as its URN segment, and `xops enrich lz-tree` logs a warning
naming the node.

System nodes build URNs with their reserved ids. An unmapped account's Environment is
`urn:tnt:testtnt:prj:unalloc:app:unalloc:env:unalloc`; an `invalid` App is
`urn:tnt:testtnt:prj:unalloc:app:invalid`; an account whose organization matches no LZ sits under
`urn:tnt:testtnt:lzc:unalloc`.

## Resolution order (D-015, D-061)

For each account in the inventory, `finops enrich map` resolves **App** then **Environment**. An
`ou_paths` entry matches the discovered OU whose id equals its `ref_id` when it has one, else the
one whose path equals its `path`; an entry that matches none gives `ou-not-found`. An `accounts`
entry matches the account whose id equals its `ref_id`.

**App**:

1. the account entry's `ref_urn_apps`, else the App of its `ref_urn_env` (`app_rule: account`);
2. the deepest OU on the account's path with `ref_urn_apps` (`app_rule: ou:<path>`), on a leaf OU
   too;
3. the LZ's `default.app` (`lz-default`), then `defaults.app` (`default`);
4. `unalloc`, with `account-unmapped`.

An OU or an account entry with several Apps makes the account **shared** between them
(`allocation.shared`), unless the account's own entry names one App.

**Environment**:

1. the account entry's `ref_urn_env` (`account`);
2. the deepest OU with `ref_urn_env` (`ou:<path>`);
3. the LZ's `default.environment`, then `defaults.environment`;
4. `unspec` (the App resolved) or `unalloc`.

Nothing is guessed from names: an OU called `Prod` sets no Environment without a `ref_urn_env`. An
OU's `ref_urn_env` sets only the Environment, by the code it ends in; it does not set the App.

**A moved account.** When an account entry sits under an OU the account was not discovered under,
its `ref_urn_apps` and `ref_urn_env` still apply and `map` writes `account-ou-mismatch`. An entry
whose account was not discovered at all is `account-not-discovered`.

**Worked example** (demo taxonomy):

- Account `200000000002` has an entry with `ref_urn_apps: [chk, pay]` and `ref_urn_env: …:env:prod`.
- Account under `/Workloads/Checkout/Prod` with no entry picks `chk` from the
  `/Workloads/Checkout` OU and `prod` from `/Workloads/Checkout/Prod`.
- Unmapped accounts get the reserved ids: **`unalloc`** for every level when no App rule matched
  (and for the Environment when no Environment rule matched either); **`unspec`** for the
  Environment when the App resolved but no Environment rule matched; tags that name no App the
  account serves map to App **`invalid`** in cost rows after share.

Shared accounts (multiple Apps on one account) are flagged in `allocation.shared`; `share` splits
cost by the App tag on each `cost` row.

## The three trees

Each tenant organizes cloud operations across three interconnected hierarchies:

1. **Business value** (`tnt > bzu > cct > ont`): `business_units` hold `cost_centers`, which hold
   `owner_teams`.
2. **Project component** (`tnt > prj > app > env`): `projects` hold `apps`, which hold
   `environments`. A project names its business unit with `ref_urn_bu`, an App its cost center with
   `ref_urn_cct`.
3. **Cloud infrastructure** (`tnt > lzc > org > acc`): the LZ holds `ou_paths`, which hold
   `accounts`. An OU or account names Apps with `ref_urn_apps` and an Environment with
   `ref_urn_env`.

`ref_id` is the id the cloud gives the node: the management account on the LZ, `ou-xxxx-xxxxxxxx`
on an OU, the 12-digit id on an account.

### Node Aliases and Uniqueness Rules

Every node across all three hierarchies can optionally carry an `alias` field:
- **Business Value hierarchy:** `bzu`, `cct`, `ont`.
- **Project Component hierarchy:** `prj`, `app`, `env`.
- **Cloud Infrastructure hierarchy:** `lzc`, `org`, `acc`.

**Uniqueness rule:** Each non-empty alias must be unique across all nodes of the **same type** within the tenant (evaluated case-insensitively).
- Two applications cannot share the same alias (e.g. `alias: core`).
- Two environments cannot share the same alias across the entire tenant.
- Two nodes of *different* types (for example, a Business Unit and an Application) **may** share the same alias without conflict.
- Empty or omitted aliases are ignored and do not conflict.
- `finops enrich validate` verifies alias uniqueness and reports error finding `taxonomy-duplicate-alias` if a collision occurs.

### Scaffolding `lz_conns` from Cloud Discovery

When bootstrapping your taxonomy, discover your AWS organization and scaffold the `lz_conns` hierarchy:

```bash
# 1. Discover active AWS accounts and OUs
go run ./xops sense aws-accounts --tenant <tenant> > accounts.jsonl

# 2. Enrich into hierarchical node records
go run ./xops enrich lz-tree --tenant <tenant> --taxonomy init-taxonomy.yaml \
  --from-accounts accounts.jsonl > tree.jsonl

# 3. Scaffold taxonomy-ready lz_conns block
go run ./xops report tree --tenant <tenant> --from-node tree.jsonl --format taxonomy --out lz-conns.yaml
```

The generated `lz-conns.yaml` formats leaf OUs (`ou_paths`) and their member accounts (`accounts`), populating `code`, `name`, `ref_id`, `path`, `status`, and URNs with codes. You can paste this block directly into your `taxonomy.yaml`.

## Validate first

```bash
go run ./finops enrich validate --tenant testtnt --taxonomy finops/enrich/testdata/demo-taxonomy.yaml
```

Fix severity `error` findings before running `map` on live data. Every `taxonomy-*` code is listed
in [findings.md](findings.md). Warnings such as unused codes are safe to review but do not block
exit 0 unless you treat warnings as errors in your process.
