# Record kinds

Every line is one flat JSON object with a `kind` field and a `tenant` field. Schemas are enforced
on declared inputs (D-037). Every record a command reads that has `tenant` must equal its
`--tenant`, whatever the kind ([commands.md](commands.md)). Optional fields appear in output when a command sets them (for example `lz` with
`--lz`).

## `ou`

Organizational unit in an AWS Organization.

| Field | Type | Required | Meaning |
|---|---|---|---|
| `kind` | string | yes | Always `ou`. |
| `tenant` | string | yes | Tenant id, the `--tenant` of the command that wrote the record (D-045). |
| `csp` | string | yes | Cloud provider (`aws`). |
| `org_id` | string | yes | Organization ID. |
| `management_account` | string | yes | Management account ID. |
| `id` | string | yes | OU ID. |
| `name` | string | yes | Display name. |
| `parent_id` | string | yes | Parent OU ID (`""` for the root). |
| `path` | string | yes | Human path from root (`/Workloads/Checkout`). |
| `id_path` | string | yes | ID path from root. |
| `has_children` | bool | yes | Whether the OU has child OUs. |
| `lz` | string | no | Landing zone code when `--lz` is used. |

## `account`

Member account in an AWS Organization.

| Field | Type | Required | Meaning |
|---|---|---|---|
| `kind` | string | yes | Always `account`. |
| `tenant` | string | yes | Tenant id, the `--tenant` of the command that wrote the record (D-045). |
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
| `joined_at` | string | no | When the account joined the organization: AWS `JoinedTimestamp` in RFC 3339 UTC, to the second (`2021-04-12T03:15:27Z`). It is the account's creation time only when `joined_method` is `CREATED`; for an `INVITED` account it is when the account accepted the invitation. `aws-accounts` always writes it, as `""` when AWS gives none. |
| `joined_method` | string | no | How the account joined: `CREATED` or `INVITED`, as AWS gives it. `aws-accounts` always writes it, as `""` when AWS gives none. |
| `lz` | string | no | Landing zone code when `--lz` is used. |

## `project_tag_cost`

Project tag total from Cost Explorer (level 1).

| Field | Type | Required | Meaning |
|---|---|---|---|
| `kind` | string | yes | Always `project_tag_cost`. |
| `tenant` | string | yes | Tenant id, the `--tenant` of the command that wrote the record (D-045). |
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
| `tenant` | string | yes | Tenant id, the `--tenant` of the command that wrote the record (D-045). |
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
| `app` | string | no | Resolved App id after `map` / `share` (`unalloc`, or `invalid` for a tag that names no App the account serves). |
| `project` | string | no | Resolved Project code. |
| `environment` | string | no | Resolved Environment code. |
| `business_unit` | string | no | Business unit code. |
| `cost_center` | string | no | Cost center code. |
| `business_unit_urn` | string | no | URN of the Business unit, `urn:tnt:<tenant>:bzu:<id>`. `""` on a shared account's row until `share`. |
| `cost_center_urn` | string | no | URN of the Cost center, under the cost center's own Business unit: `urn:tnt:<tenant>:bzu:<id>:cct:<id>`. `""` on a shared account's row until `share`. |
| `project_urn` | string | no | URN of the Project, `urn:tnt:<tenant>:prj:<id>`. `""` on a shared account's row until `share`. |
| `app_urn` | string | no | URN of the App, `urn:tnt:<tenant>:prj:<id>:app:<id>`. `""` on a shared account's row until `share`. |
| `environment_urn` | string | no | URN of the Environment under its App, `…:app:<id>:env:<id>`. `""` on a shared account's row until `share`. |
| `lz_urn` | string | no | URN of the landing zone, `urn:tnt:<tenant>:lzc:<id>`. |
| `ou_urn` | string | no | URN of the OU, `…:lzc:<id>:org:<OU path>`. The path is slugged and has no leading `/` (`/Workloads/Checkout` is `workloads/checkout`); the root is `root`; an account missing from the inventory is `unknown`. |
| `account_urn` | string | no | URN of the account, `<ou_urn>:acc:<account_id>`. |
| `service_urn` | string | no | URN of the service in that account, `<account_urn>:svc:<slug of service>`. |

## `allocation`

One row per account (and App) after `map`: who owns cost in that account.

| Field | Type | Required | Meaning |
|---|---|---|---|
| `kind` | string | yes | Always `allocation`. |
| `tenant` | string | yes | Tenant id, the `--tenant` of the command that wrote the record (D-045). |
| `account_id` | string | yes | Account ID. |
| `account_name` | string | yes | Account name. |
| `lz` | string | yes | Landing zone code. |
| `ou_path` | string | yes | OU path for the account. |
| `project` | string | yes | Project code. |
| `app` | string | yes | App id (`unalloc` when no App rule matched). |
| `environment` | string | yes | Environment id (`unalloc`, or `unspec` when the account is known but no Environment rule matched). |
| `business_unit` | string | yes | Business unit code. |
| `cost_center` | string | yes | Cost center code. |
| `business_unit_urn` | string | yes | URN of the Business unit, `urn:tnt:<tenant>:bzu:<id>`. |
| `cost_center_urn` | string | yes | URN of the Cost center, under the cost center's own Business unit: `urn:tnt:<tenant>:bzu:<id>:cct:<id>`. |
| `project_urn` | string | yes | URN of the Project, `urn:tnt:<tenant>:prj:<id>`. |
| `app_urn` | string | yes | URN of the App, `urn:tnt:<tenant>:prj:<id>:app:<id>`. |
| `environment_urn` | string | yes | URN of the Environment under its App, `…:app:<id>:env:<id>`. |
| `lz_urn` | string | yes | URN of the landing zone, `urn:tnt:<tenant>:lzc:<id>`. |
| `ou_urn` | string | yes | URN of the OU, `…:lzc:<id>:org:<OU path>`. The path is slugged and has no leading `/`; the root is `root`. |
| `account_urn` | string | yes | URN of the account, `<ou_urn>:acc:<account_id>`. |
| `app_rule` | string | yes | How App was resolved (rule name). |
| `environment_rule` | string | yes | How Environment was resolved. |
| `shared` | bool | yes | Whether the account is treated as shared. |

## `change`

Month-on-month mapping change from `finops analyze changes`.

| Field | Type | Required | Meaning |
|---|---|---|---|
| `kind` | string | yes | Always `change`. |
| `tenant` | string | yes | Tenant id, the `--tenant` of the command that wrote the record (D-045). |
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
| `tenant` | string | yes | Tenant id, the `--tenant` of the command that wrote the record (D-045). |
| `severity` | string | yes | `error` or `warning`. |
| `code` | string | yes | Stable code (see [findings.md](findings.md)). |
| `subject` | string | yes | What the finding refers to. |
| `message` | string | yes | Human-readable detail. |
| `command` | string | yes | Command that emitted the finding. |
