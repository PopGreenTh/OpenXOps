# WI-0301 — Service categories and the full cost URN

## Outcome

Every `cost` record says which service category its service belongs to, and its `service_urn`
follows the whole cloud tree: account › region › category › service
([D-064](../DECISIONS.md#d-064)). Plan lines can then name a category or a service under an
account, and variance can roll up to them.

## Contracts

### Taxonomy: `service_categories`

A new optional tenant-level list in the nested taxonomy (WI-0111):

```yaml
service_categories:
  - code: compute
    name: Compute
    services: [Amazon Elastic Compute Cloud - Compute, AWS Lambda]
  - code: storage
    name: Storage
    services: [Amazon Simple Storage Service]
```

- `code` and `name` follow the node rules (D-056). `other` is reserved, like `unalloc`.
- `services` lists the service names as the cost source spells them (`aws-cost`'s `service`,
  FOCUS's `ServiceName`), matched exactly.
- `validate`:
  - a service in two categories is `taxonomy-service-category-duplicate` (error, subject the
    service);
  - a duplicate category code is `taxonomy-duplicate-code`;
  - the code `other` is `taxonomy-reserved-code`.

### `map`: the new fields

Every `cost` record `map` writes gains:

| Field | Value |
|---|---|
| `region` | the record's `region` if it has one (FOCUS), else `""` |
| `geo_urn` | `<account_urn>:geo:<slug of region>`, or `<account_urn>:geo:global` when `region` is `""` |
| `service_category` | the category whose `services` holds the record's `service`, else `other` |
| `service_category_urn` | `<geo_urn>:sct:<service_category>` |
| `service_urn` | `<service_category_urn>:svc:<slug of service>` |

- A service in no category gives `service-uncategorized` (warning, subject the service name), once
  per service and run.
- `share` copies the five fields from the record it splits, like the other URN fields.
- `trend`'s cost hierarchy becomes `ou` › `account` › `region` › `service_category` › `service`.
  Its node URNs are the record's `ou_urn`, `account_urn`, `geo_urn`, `service_category_urn` and
  `service_urn`, and `region` is always a level. This amends WI-0203's table.
- `allocation` records are unchanged: they stop at the account.

### Reference cases

The planner updates the `map` and `share` cases' expected `service_urn` to the new shape, and
adds a case for an uncategorized service and one for a FOCUS region, on `claude/cases-sct`. The
trend case `04-cost-hierarchy-without-region` is updated there too.

## WO-0301-A — Service categories and the full cost URN

**Assignee:** open
**Depends on:** Release 2 (WO-0203-A for `trend`)
**Packages:**
- `record/taxonomy/` and `finops/enrich/` (`validate.go`, `resolve.go`);
- `finops/enrich/share/`;
- `finops/analyze/` (the trend cost hierarchy);
- `record/schema/` (the new optional `cost` fields);
- the demo taxonomies, fixtures and golden files whose `service_urn` changes;
- `docs/user/` (`taxonomy.md`, `records.md`, `findings.md`), `templates/taxonomy.yaml`

**Reference cases.** Merge `claude/cases-sct` with a merge commit; don't edit `testdata/cases/`.

**Goal.** *`service_categories`* and *`map`: the new fields* above.

**In scope.**
- The taxonomy list, its `validate` rules and the reserved code.
- The five fields in `map` and `share`; the finding.
- `trend`'s cost hierarchy with the region and category levels.
- The demo taxonomy gets categories for every mock service but one, so the mock shows both
  cases.
- Tests:
  - the `validate` rules;
  - a categorized and an uncategorized service;
  - a record with and without a region;
  - `share` keeping the fields.

**Out of scope.** Resource and usage-type levels (`rsg`, `sus`). Built-in category defaults.

**Done when.**
- Every `map`, `share` and `trend` reference case passes, unchanged from `claude/cases-sct`.
- On the mock pipeline, every `cost` record's `service_urn` matches
  `<account_urn>:geo:global:sct:<category>:svc:<slug>`, and exactly one service is `other` with
  one `service-uncategorized` finding (shown in the PR; release criterion 4).
- `go run -tags=no_duckdb_arrow ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run -tags=no_duckdb_arrow ./scripts/check`
