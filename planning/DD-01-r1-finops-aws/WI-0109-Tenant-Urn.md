# WI-0109 — Tenants and URNs

## Outcome

Every command runs for one tenant, named on the command line, and every record says which
tenant it belongs to (D-045). Every node from Tenant down to Service is named by a URN in one
of three hierarchies (D-046). Taxonomy codes become short lowercase ids (D-047). `allocation`
and `cost` records carry the URN of every level they belong to (D-048). Release 1 covers only
the levels it has data for (D-049).

## Contracts

These amend the Contracts of WI-0101 to WI-0107. Where they disagree, this file wins.

### Tenant (D-045)

- Every command except `version` takes **`--tenant ID`**, required. A missing or malformed id
  stops the command before it reads anything.
- The taxonomy file starts with **`tenant: <id>`**, required. `validate`, `map`, `share` and
  `aws-cost --taxonomy` stop when it differs from `--tenant`, naming both.
- **Every record a command writes has `tenant`** (its own records: Sense output, `allocation`,
  `cost`, `change`, `finding`). Records passed through are unchanged.
- **Every record a command reads** that has a `tenant` field must equal `--tenant`, or the
  command stops with an error naming the file (or `stdin`), the line and both tenants. This
  applies to every kind, declared or passed through.
- `tenant` is a required string in every schema of `record/schema`.
- The Cost Explorer cache is unchanged: it stays keyed by management account.

### Ids (D-047)

- **Id rule**: `^[a-z0-9]+(-[a-z0-9]+)*$`. It applies to `--tenant`, to the taxonomy's `tenant`,
  and to every `code` in the taxonomy (`business_units`, `cost_centers`, `projects`, `apps`,
  their `environments`, `lzs`). The key stays `code`; only its values change (`code: shop`).
- **Reserved ids**: `unalloc`, `unspec`, `invalid`. `validate` rejects them in the file with
  `taxonomy-reserved-code`. This replaces the reserved `-UNALLOC`, `-UNSPEC` and `-INVALID`
  endings.
- System nodes in records, replacing today's codes:

  | Was | Now |
  |---|---|
  | `LZ-UNALLOC`, `PRJ-UNALLOC`, `APP-UNALLOC`, `ENV-UNALLOC`, `BU-UNALLOC`, `CC-UNALLOC` | `unalloc` |
  | `ENV-UNSPEC` | `unspec` |
  | `APP-INVALID` | `invalid` |

- **Slug rule** (for names from the cloud): first split CamelCase (D-053, from WO-0109-C):
  put a word break between a lowercase letter or digit and the uppercase letter after it, and
  between two uppercase letters when the second is followed by a lowercase letter. Then
  lowercase, turn every run of characters outside `[a-z0-9]` into one `-`, and trim at both
  ends. An empty result is `unknown`.

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
- An **OU path** becomes its segments, each slugged, joined by `/`, without the leading `/`
  (`/Workloads/Checkout/Prod` → `workloads/checkout/prod`). The root `/` is `root`.
- **One LZ per tenant** (D-052, from WO-0109-C): the taxonomy's `lzs` has exactly one entry;
  `validate` reports any other count as `taxonomy-lz-count` (error, subject `lzs`). With
  `--taxonomy`, `--lz` is optional on `aws-accounts` and `aws-cost` and defaults to that LZ; a
  `--lz` naming another code is an error that names both codes.

### URNs (D-046)

`urn:tnt:<tenant>` followed by `:<type>:<id>` for each level. Release 1 builds these (`<t>` is
the tenant):

| Field | URN |
|---|---|
| `business_unit_urn` | `urn:tnt:<t>:bzu:<business_unit>` |
| `cost_center_urn` | `urn:tnt:<t>:bzu:<cost center's business_unit>:cct:<cost_center>` |
| `project_urn` | `urn:tnt:<t>:prj:<project>` |
| `app_urn` | `urn:tnt:<t>:prj:<project>:app:<app>` |
| `environment_urn` | `urn:tnt:<t>:prj:<project>:app:<app>:env:<environment>` |
| `lz_urn` | `urn:tnt:<t>:lzc:<lz>` |
| `ou_urn` | `urn:tnt:<t>:lzc:<lz>:org:<OU path>` |
| `account_urn` | `urn:tnt:<t>:lzc:<lz>:org:<OU path>:acc:<account_id>` |
| `service_urn` (on `cost` only) | `<account_urn>:svc:<slug of service>` |

System nodes build URNs the same way, with their reserved ids: an unmapped account's
`environment_urn` is `urn:tnt:<t>:prj:unalloc:app:unalloc:env:unalloc`. An `invalid` App sits
under Project `unalloc`. A cost center under BusinessUnit `unalloc` is
`urn:tnt:<t>:bzu:unalloc:cct:unalloc`. An account whose LZ is unknown sits under `lzc:unalloc`.

### Records (D-048)

- **`allocation`** (from `map`): the fields of WI-0104, now holding ids, plus `tenant` and all
  eight URN fields above except `service_urn`.
- **`cost`** after `map`: plus `tenant`, `lz_urn`, `ou_urn`, `account_urn`, `service_urn` and
  `environment_urn`. When not shared, it also gets `project_urn`, `app_urn`,
  `business_unit_urn` and `cost_center_urn`. A shared account's cost keeps those four, and
  `environment_urn`, `""` for `share` to fill, as it does the id fields.
- **`cost`** after `share`: every URN field filled, from the App each record lands on.
- The URN fields are optional strings in the `cost` schema, since `aws-cost` doesn't write them.
  On `allocation` they are required.
- Finding subjects and `change` subjects use ids where they used codes.

### What isn't in Release 1 (D-049)

The `ont`, `pdl`, `cst`, `ftr`, `fnc`, `geo`, `rsg` and `sus` levels, and the Environment,
Feature and Function cross-references. The taxonomy rejects them as unknown keys until a later
Work Item adds them.

### Reference cases

The planner converts every case under `finops/enrich/testdata/cases/` to ids, `tenant: testtnt`
and, where the case is about naming, URN fields. The converted cases are on the branch
`claude/cases-urn`. WO-0109-B merges that branch into its own branch and does not edit them.

## WO-0109-A — Tenant

**Assignee:** open
**Depends on:** WO-0108-B
**Packages:** `record/`, `record/schema/`, `record/urn/` (new), the `commands.go` of every
command package, `xops/sense/`, `finops/sense/`, `finops/enrich/taxonomy.go` (the `tenant` key
only), every command's fixtures and golden files, `scripts/doccheck/testdata/`, `docs/`,
`README.md`, `Taskfile.yml`, `.vscode/launch.json`

**Goal.** The tenant rules above, and the `record/urn` package (the id rule, the slug rule, the
OU path rule, and a URN builder) for WO-0109-B to use.

**In scope.**
- `--tenant` on every command; the taxonomy `tenant` key and its check; `tenant` written on
  every record a command produces; the read check; `tenant` in every schema.
- `record/urn/`: `ValidID`, `Slug`, `OUPath` and a builder, table-tested with the examples in
  the Contracts, including an empty slug and the root OU.
- Every fixture, golden file, Task, launch entry and doc example gains `--tenant testtnt` or
  `tenant`. The demo taxonomy gains `tenant: testtnt` and keeps its codes until WO-0109-B.

**Out of scope.** Ids and URN fields in `map` and `share` (WO-0109-B).

**Done when.**
- Tests: a command without `--tenant` fails before reading; a taxonomy for another tenant
  fails; a record from another tenant fails with file, line and both tenants; every record each
  command writes has `tenant`.
- The getting-started `--mock` walkthrough, updated with `--tenant testtnt`, still writes
  `finops.xlsx`, and every output record has `tenant: testtnt` (shown in the PR).
- `go run ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run ./scripts/check`

## WO-0109-B — Ids and URNs

**Assignee:** open
**Depends on:** WO-0109-A
**Packages:** `finops/enrich/` (including `share/`), `finops/analyze/` (fixtures only),
`record/schema/`, `finops/enrich/testdata/` (fixtures, and the converted cases merged from
`claude/cases-urn`), `finops/analyze/testdata/`, `xops/report/testdata/`, `docs/`, `README.md`

**Goal.** Taxonomy codes as ids, system ids, and the URN fields on `allocation` and `cost`.

**In scope.**
- `validate`: the id rule on every `code`; reserved ids; the new `taxonomy-*` codes, if any,
  documented in `findings.md`.
- `map` and `share`: ids for system nodes and URN fields as specified, built with `record/urn`.
- The demo taxonomy and every fixture converted to ids. `docs/user/` updated: `taxonomy.md`
  (ids, reserved ids), `records.md` (the URN fields), and `findings.md` subjects.
- The URN fields in `record/schema` (required on `allocation`, optional on `cost`).

**Out of scope.** The levels and cross-references of D-049's "later" list.

**Done when.**
- Every converted case under `finops/enrich/testdata/cases/` passes, unchanged from
  `claude/cases-urn`.
- On the mock pipeline, every `cost` record after `share` has all nine URN fields non-empty, and
  every URN starts with `urn:tnt:testtnt:` (shown in the PR).
- Release criteria 1 to 5 and 7 still hold on the branch.
- `go run ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run ./scripts/check`

## WO-0109-C — CamelCase slugs and one LZ per tenant

**Assignee:** open
**Depends on:** WO-0109-B and WO-0102-B (they touch the same packages)
**Packages:**
- `record/urn/`;
- `finops/enrich/` (`validate.go`, `commands.go`) and `finops/enrich/testdata/` (fixtures, and
  the cases merged from `claude/cases-slug`);
- `xops/sense/` and `finops/sense/` (`--lz` defaults, and their golden files);
- every other fixture and golden file whose slugs change;
- `docs/`, `README.md`, `Taskfile.yml` and `.vscode/launch.json`.

**Reference cases.** Before dispatch, the planner updates the cases whose slugs change (today,
the two `svc:amazonec2`, which become `svc:amazon-ec2`) on the branch `claude/cases-slug`. Merge
that branch into yours with a merge commit, and don't edit any file under `testdata/cases/`.

**Goal.** The D-053 slug rule and the D-052 one-LZ rule, everywhere they apply.

**In scope.**
- `urn.Slug` splits CamelCase as in the table under *Ids*; table-tested with every row.
- `validate`: `taxonomy-lz-count` when `lzs` has other than one entry. Document it in
  `findings.md`.
- `aws-accounts` and `aws-cost`: `--lz` optional with `--taxonomy`, defaulting to its LZ; a
  `--lz` naming another code stops the command, naming both codes. `--lz` without `--taxonomy`
  is still an error.
- Fixtures, golden files, Tasks, launch entries and doc examples follow the new slugs (the demo
  organization's `NonProd` OUs become `non-prod` in every URN). Drop `--lz` from examples where
  `--taxonomy` now supplies it.
- `taxonomy.md` and `commands.md`: one LZ per tenant, the `--lz` default and the slug rule with
  its examples.

**Out of scope.** The cloud tree (WI-0110). OU URN collisions (Q-009).

**Done when.**
- Every case under `finops/enrich/testdata/cases/` passes, unchanged from `claude/cases-slug`.
- Tests: the slug table; `taxonomy-lz-count` for zero and for two LZs; `--lz` defaulting from
  `--taxonomy`, and a `--lz` that differs from it stopping with both codes.
- On the mock pipeline with `--tenant testtnt`, no URN contains an uppercase letter, and the
  `NonProd` accounts' `ou_urn` ends in `:org:workloads/checkout/non-prod` or
  `:org:workloads/payments/non-prod` (shown in the PR).
- Release criteria 1 to 5, 7 and 8 still hold on the branch.
- `go run ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run ./scripts/check`
