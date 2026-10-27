# Taxonomy file

The taxonomy is a YAML file read by `finops enrich validate`, `map`, and `share`. It describes
business structure, landing zones, how OUs and accounts map to Apps and Environments, and which
tag keys to use for cost.

See `finops/enrich/testdata/demo-taxonomy.yaml` for a working example.

## Top-level sections

| Section | Purpose |
|---|---|
| `business_units` | List of `{ code, name }` business units. |
| `cost_centers` | `{ code, name, business_unit }` — cost center rolls up to a business unit. |
| `projects` | `{ code, name, business_unit }` — project for reporting. |
| `apps` | App definitions with `project`, `cost_center`, and `environments`. |
| `lzs` | Landing zones: `{ code, name, csp, currency, management_account }`. |
| `ou_links` | Map OU paths to an App and/or Environment for an LZ. |
| `accounts` | Per-account overrides: `{ account, apps[], environment }`. |
| `tags` | Optional `{ app, project }` keys for Cost Explorer (defaults `App` / `Project`). |
| `defaults` | Optional default App when an account has no other rule. |

Codes must not use reserved suffixes `-UNALLOC`, `-UNSPEC`, or `-INVALID` (validation errors).

## Resolution order (D-015)

For each account in the inventory, `finops enrich map` resolves **App** then **Environment**:

1. **Account override** — if `accounts` lists the account with `apps` and/or `environment`, use
   those values.
2. **OU links** — walk the account's `ou_path` from the leaf upward; the deepest matching
   `ou_links` entry for the LZ wins for App and/or Environment.
3. **Defaults** — taxonomy `defaults.app` when nothing else matched.

**Worked example** (demo taxonomy):

- Account `200000000002` has override `apps: [APP-CHK, APP-PAY]` and `environment: ENV-PROD`.
- Account under `/Workloads/Checkout/Prod` with no override picks `APP-CHK` from the
  `/Workloads/Checkout` link and `ENV-PROD` from `/Workloads/Checkout/Prod`.
- Unmapped accounts get synthetic codes: **`APP-UNALLOC`** / **`ENV-UNALLOC`** when no App or
  Environment rule matched; **`APP-UNSPEC`** / **`ENV-UNSPEC`** when the account is known but
  ambiguous; invalid tag combinations map to **`*-INVALID`** in cost rows after share.

Shared accounts (multiple Apps on one account) are flagged in `allocation.shared`; `share` splits
cost by the App tag on each `cost` row.

## Validate first

```bash
go run ./finops enrich validate --taxonomy finops/enrich/testdata/demo-taxonomy.yaml
```

Fix severity `error` findings before running `map` on live data. Warnings such as unused codes are
safe to review but do not block exit 0 unless you treat warnings as errors in your process.
