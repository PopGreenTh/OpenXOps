# Record kinds

Every line is one flat JSON object with a `kind` field. Schemas are enforced on declared inputs
(D-037). Optional fields appear in output when a command sets them (for example `lz` with
`--lz`).

## `ou`

Organizational unit in an AWS Organization.

| Field | Type | Required | Meaning |
|---|---|---|---|
| `kind` | string | yes | Always `ou`. |
| `csp` | string | yes | Cloud provider (`aws`). |
| `org_id` | string | yes | Organization ID. |
| `management_account` | string | yes | Management account ID. |
| `id` | string | yes | OU ID. |
| `name` | string | yes | Display name. |
| `parent_id` | string | yes | Parent OU ID (`""` for the root). |
| `path` | string | yes | Human path from root (`/Workloads/Checkout`). |
| `id_path` | string | yes | ID path from root. |
| `lz` | string | no | Landing zone code when `--lz` is used. |

## `account`

Member account in an AWS Organization.

| Field | Type | Required | Meaning |
|---|---|---|---|
| `kind` | string | yes | Always `account`. |
| `csp` | string | yes | Cloud provider (`aws`). |
| `org_id` | string | yes | Organization ID. |
| `management_account` | string | yes | Management account ID. |
| `id` | string | yes | Account ID. |
| `name` | string | yes | Account name. |
| `email` | string | yes | Root email. |
| `status` | string | yes | Account status (`ACTIVE`, …). |
| `parent_id` | string | yes | Parent OU ID. |
| `ou_path` | string | yes | Parent OU human path. |
| `ou_id_path` | string | yes | Parent OU ID path. |
| `lz` | string | no | Landing zone code when `--lz` is used. |

## `project_tag_cost`

Project tag total from Cost Explorer (level 1).

| Field | Type | Required | Meaning |
|---|---|---|---|
| `kind` | string | yes | Always `project_tag_cost`. |
| `csp` | string | yes | Cloud provider (`aws`). |
| `management_account` | string | yes | Management account ID. |
| `granularity` | string | yes | `monthly` or `daily`. |
| `period` | string | yes | Period key (`YYYY-MM` or `YYYY-MM-DD`). |
| `metric` | string | yes | Cost metric name. |
| `amount` | number | yes | Cost amount. |
| `currency` | string | yes | Currency code. |
| `estimated` | bool | yes | Whether Cost Explorer marked the value estimated. |
| `account_id` | string | yes | Linked account ID. |
| `project_tag` | string | yes | Project tag value (`""` = untagged). |
| `lz` | string | no | Landing zone code when `--lz` is used. |

## `cost`

Service and App tag cost from Cost Explorer (level 2), or enriched after `map` / `share`.

| Field | Type | Required | Meaning |
|---|---|---|---|
| `kind` | string | yes | Always `cost`. |
| `csp` | string | yes | Cloud provider (`aws`). |
| `management_account` | string | yes | Management account ID. |
| `granularity` | string | yes | `monthly` or `daily`. |
| `period` | string | yes | Period key. |
| `metric` | string | yes | Cost metric name. |
| `amount` | number | yes | Cost amount. |
| `currency` | string | yes | Currency code. |
| `estimated` | bool | yes | Estimated flag from Cost Explorer. |
| `account_id` | string | yes | Linked account ID. |
| `service` | string | yes | AWS service name. |
| `app_tag` | string | yes | App tag value on the row (`""` = untagged). |
| `lz` | string | no | Landing zone code when `--lz` is used. |
| `app` | string | no | Resolved App code after `map` / `share`. |
| `project` | string | no | Resolved Project code. |
| `environment` | string | no | Resolved Environment code. |
| `business_unit` | string | no | Business unit code. |
| `cost_center` | string | no | Cost center code. |

## `allocation`

One row per account (and App) after `map`: who owns cost in that account.

| Field | Type | Required | Meaning |
|---|---|---|---|
| `kind` | string | yes | Always `allocation`. |
| `account_id` | string | yes | Account ID. |
| `account_name` | string | yes | Account name. |
| `lz` | string | yes | Landing zone code. |
| `ou_path` | string | yes | OU path for the account. |
| `project` | string | yes | Project code. |
| `app` | string | yes | App code (`APP-UNALLOC`, `APP-UNSPEC`, …). |
| `environment` | string | yes | Environment code (`ENV-UNALLOC`, …). |
| `business_unit` | string | yes | Business unit code. |
| `cost_center` | string | yes | Cost center code. |
| `app_rule` | string | yes | How App was resolved (rule name). |
| `environment_rule` | string | yes | How Environment was resolved. |
| `shared` | bool | yes | Whether the account is treated as shared. |

## `change`

Month-on-month mapping change from `finops analyze changes`.

| Field | Type | Required | Meaning |
|---|---|---|---|
| `kind` | string | yes | Always `change`. |
| `subject_type` | string | yes | `account` or `allocation`. |
| `subject` | string | yes | Subject identifier. |
| `change` | string | yes | Change type (`added`, `removed`, …). |
| `field` | string | yes | Field that changed. |
| `old` | string | yes | Previous value. |
| `new` | string | yes | New value. |
| `cost_moved` | number or null | yes | Cost affected when applicable. |
| `currency` | string | yes | Currency for `cost_moved`. |
| `period_from` | string | yes | Start period for the comparison. |
| `period_to` | string | yes | End period. |

## `finding`

Validation or mapping issue emitted as a record (stdout); severity `error` makes `validate` exit 1.

| Field | Type | Required | Meaning |
|---|---|---|---|
| `kind` | string | yes | Always `finding`. |
| `severity` | string | yes | `error` or `warning`. |
| `code` | string | yes | Stable code (see [findings.md](findings.md)). |
| `subject` | string | yes | What the finding refers to. |
| `message` | string | yes | Human-readable detail. |
| `command` | string | yes | Command that emitted the finding. |
