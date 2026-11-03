# Taxonomy file

The taxonomy is a YAML file read by `finops enrich validate`, `map`, and `share` (and by
`finops sense aws-cost --taxonomy`). It belongs to one tenant and describes
business structure, landing zones, how OUs and accounts map to Apps and Environments, and which
tag keys to use for cost.

See `finops/enrich/testdata/demo-taxonomy.yaml` for a working example.

## Top-level sections

| Section | Purpose |
|---|---|
| `tenant` | Required, first line of the file: the tenant id (`tenant: testtnt`). It must equal `--tenant`, or `validate`, `map`, `share` and `aws-cost --taxonomy` stop with an error naming both. Same id rule as `--tenant`: lowercase letters and digits, joined by single dashes. |
| `business_units` | List of `{ code, name }` business units. |
| `cost_centers` | `{ code, name, business_unit }` — cost center rolls up to a business unit. |
| `projects` | `{ code, name, business_unit }` — project for reporting. |
| `apps` | App definitions with `project`, `cost_center`, and `environments`. |
| `lzs` | The tenant's one landing zone: `{ code, name, csp, currency, management_account }`. A tenant has exactly one LZ (D-052); `validate` reports any other count as `taxonomy-lz-count`. |
| `ou_links` | Map OU paths to an App and/or Environment for an LZ. |
| `accounts` | Per-account overrides: `{ account, apps[], environment }`. |
| `tags` | Optional `{ app, project }` keys for Cost Explorer (defaults `App` / `Project`). |
| `defaults` | Optional default App when an account has no other rule. |

## Codes and reserved ids (D-047, D-056)

Every `code` (business units, cost centers, projects, apps, their `environments`, and `lzs`) is 3
to 50 characters from letters, digits and `. + = @ _ -` (`^[A-Za-z0-9.+=@_-]{3,50}$`), for example
`retail`, `PRJ_Shop`, `b2c-eu` or `x@y=z+1`. The same rule binds `--tenant` and the file's
`tenant`. `validate` reports any other value (`qa` is too short, `a/b` and `a b` have a character
that is not allowed) as `taxonomy-invalid-id`.

- **Case.** Records hold the code as written (`project: PRJ_Shop`); the URN holds it in lowercase
  (`project_urn: urn:tnt:testtnt:prj:prj_shop`). So two codes in one list that differ only in
  case (`Shop` and `shop`) are `taxonomy-duplicate-code`, and App tags still match codes in any
  case. References between sections (`project:`, `cost_center:`, `apps:`) are written exactly as
  the code is.
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

## Resolution order (D-015)

For each account in the inventory, `finops enrich map` resolves **App** then **Environment**:

1. **Account override** — if `accounts` lists the account with `apps` and/or `environment`, use
   those values.
2. **OU links** — walk the account's `ou_path` from the leaf upward; the deepest matching
   `ou_links` entry for the LZ wins for App and/or Environment.
3. **Defaults** — taxonomy `defaults.app` when nothing else matched.

**Worked example** (demo taxonomy):

- Account `200000000002` has override `apps: [chk, pay]` and `environment: prod`.
- Account under `/Workloads/Checkout/Prod` with no override picks `chk` from the
  `/Workloads/Checkout` link and `prod` from `/Workloads/Checkout/Prod`.
- Unmapped accounts get the reserved ids: **`unalloc`** for every level when no App rule matched
  (and for the Environment when no Environment rule matched either); **`unspec`** for the
  Environment when the App resolved but no Environment rule matched; tags that name no App the
  account serves map to App **`invalid`** in cost rows after share.

Shared accounts (multiple Apps on one account) are flagged in `allocation.shared`; `share` splits
cost by the App tag on each `cost` row.

## Formats: Flat and Nested

OpenXOps supports two taxonomy layouts:

1. **Flat format** (`finops/enrich/testdata/demo-taxonomy.yaml`) — top-level lists with explicit reference fields (`cost_centers` reference `business_unit`, `apps` reference `project` and `cost_center`, `ou_links` map OUs to apps, `accounts` list overrides).
2. **Nested hierarchical format** (`finops/enrich/testdata/nested-taxonomy.yaml`) — structured representation of the three organizational trees:

### The Three Taxonomy Trees

Each tenant organizes cloud operations across three distinct, interconnected hierarchies:

#### 1. Business Value Hierarchy (`tnt > bzu > cct > ont`)
Organizes corporate structure, financial ownership, and cost accountability:
- `tenant` (`tnt`): Tenant root (`code`, `urn`, `name`).
- `business_units` (`bzu`): Business units representing corporate divisions.
- `cost_centers` (`cct`): Cost centers nested under their owning business unit.
- `owner_teams` (`ont`): Engineering or operations teams nested under their cost center.

#### 2. Project Component Hierarchy (`tnt > prj > app > env`)
Organizes software systems, applications, and runtime environments:
- `projects` (`prj`): Business initiatives or product suites. References its owning BU via `ref_urn_bu`.
- `apps` (`app`): Distinct deployable applications nested under a project. References its cost center via `ref_urn_cct`.
- `environments` (`env`): Deployment tiers (`prd`, `stg`, `sit`, `dev`) nested under each app. Can reference its responsible owner team via `ref_urn_ont`.

#### 3. Cloud Infrastructure Hierarchy (`tnt > lzc > org > acc`)
Organizes cloud provider tenancies, landing zones, and account inventory:
- `lz_conns` (`lzc`): Cloud landing zone connection (`code`, `urn`, `csp`, `currency`, `management_account`, `ref_id`).
- `ou_paths` (`org`): Leaf Organizational Units containing member accounts (`code`, `urn`, `ref_id`, `path`).
- `accounts` (`acc`): Member cloud accounts staying in leaf OUs (`code`, `urn`, `ref_id`, `status`). Maps to an environment via `ref_urn_env`.

### Common Node Fields and Identifier Conventions

Every node in each tree contains:
- `code`: Node identifier matching `^[A-Za-z0-9.+=@_-]{3,50}$` (e.g., `CIF`, `CC-CIF01`, `cvd-aws`).
- `urn`: Stable URN matching `^[a-z0-9.:+=@_/-]{11,250}$` built strictly from slugged node codes down the hierarchy.
- `name`: Human-readable display label.
- `alias`: (Optional) Short alternative identifier.
- `ref_id`: (Cloud tree nodes only) External cloud provider identifier (e.g., AWS 12-digit Account ID or `ou-xxxx-xxxxxxxx`).

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
