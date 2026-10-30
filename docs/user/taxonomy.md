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
| `lzs` | Landing zones: `{ code, name, csp, currency, management_account }`. |
| `ou_links` | Map OU paths to an App and/or Environment for an LZ. |
| `accounts` | Per-account overrides: `{ account, apps[], environment }`. |
| `tags` | Optional `{ app, project }` keys for Cost Explorer (defaults `App` / `Project`). |
| `defaults` | Optional default App when an account has no other rule. |

## Ids and reserved ids (D-047)

Every `code` (business units, cost centers, projects, apps, their `environments`, and `lzs`) is an
**id**: lowercase letters and digits, joined by single dashes (`^[a-z0-9]+(-[a-z0-9]+)*$`), for
example `retail`, `chk` or `b2c-eu`. The key stays `code`; only its values are ids. `validate`
reports any other value as `taxonomy-invalid-id`.

Three ids are **reserved** for the system nodes `map` and `share` create. `validate` rejects them
as codes with `taxonomy-reserved-code`:

| Id | Meaning |
|---|---|
| `unalloc` | Not allocated: no LZ, Project, App, Environment, Business unit or Cost center matched. |
| `unspec` | Environment only: the account is known but no Environment rule matched. |
| `invalid` | App only: a tag named an App the account does not serve, or no App at all. It sits under Project `unalloc`. |

## URNs (D-046)

`map` and `share` name every node by a URN, written to the `*_urn` fields of `allocation` and
`cost` ([records.md](records.md)). A URN is `urn:tnt:<tenant>` followed by `:<type>:<id>` for each
level, where `<tenant>` is the file's `tenant`:

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

Names that come from the cloud are slugged: lowercase, every run of characters outside
`[a-z0-9]` becomes one `-`, trimmed at both ends (an empty result is `unknown`). An OU path becomes
its slugged segments joined by `/`, without the leading `/`; the root `/` is `root`.

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

## Validate first

```bash
go run ./finops enrich validate --tenant testtnt --taxonomy finops/enrich/testdata/demo-taxonomy.yaml
```

Fix severity `error` findings before running `map` on live data. Every `taxonomy-*` code is listed
in [findings.md](findings.md). Warnings such as unused codes are safe to review but do not block
exit 0 unless you treat warnings as errors in your process.
