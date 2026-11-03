# WI-0111 — The nested taxonomy and cloud URNs from codes

## Outcome

A tenant writes its taxonomy as the three trees of [D-046](../DECISIONS.md#d-046), nested the
way people read them, and every command reads it through one loader
([D-060](../DECISIONS.md#d-060)). URNs are built from codes, and a URN written in the file is
checked against them ([D-058](../DECISIONS.md#d-058)). OUs and accounts get URNs from their own
codes, and two nodes can never share one ([D-059](../DECISIONS.md#d-059)).

This Work Item specifies commit `793fbf8` after the fact ([D-057](../DECISIONS.md#d-057)): what
it built stays where it matches these Contracts and changes where it doesn't.

## Contracts

These amend the Contracts of WI-0104, WI-0109 and WI-0110. Where they disagree, this file wins.

### Taxonomy file (D-060)

```yaml
tenant: { code: testtnt, name: Test tenant }          # urn optional: urn:tnt:testtnt
business_units:
  - code: retail
    name: Retail
    cost_centers:
      - { code: ecom, name: E-commerce }               # owner_teams: read, not used (D-057)
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
lzs:                                                    # or lz_conns; exactly one LZ (D-052)
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

- **Every node** has `code` and `name`, and may have `urn` and `alias`. Cloud nodes (`lzs`,
  `ou_paths`, `accounts`) may have `ref_id`. An LZ's `ref_id`, when given, equals its
  `management_account`.
- **An OU entry** has `path`, `ref_id` or both. With `ref_id` it matches by id, so `path` may be
  left out (reference case 10).
- **References** are URNs: `ref_urn_bu` on a Project, `ref_urn_cct` on an App, `ref_urn_apps`
  (a list of App URNs) on an OU or an account, and `ref_urn_env` (one Environment URN) on an OU
  or an account. Defaults stay codes, as before.
- **Read, not used** (D-057): `owner_teams` under a cost center, `ref_urn_ont` on an
  Environment, `ref_urn_prjs` on an LZ, `contact` on an account. Any other unknown key is an
  error, as before.
- **Uniqueness**, compared in any case: business units, projects, Apps and LZs in the tenant;
  cost centers in their business unit; Environments in their App; OUs in their LZ; accounts in
  their OU. An `alias` is unique among nodes of one type in the tenant.

### One loader

- The loader moves out of `finops/enrich/` to a new package **`record/taxonomy/`**: it reads the
  file into one model, and every command that takes `--taxonomy` uses it (`finops enrich`,
  `finops sense aws-cost`, `xops sense aws-accounts`, `xops enrich lz-tree`). The copies of the
  tenant and LZ parsing in `finops/sense/` and `xops/sense/` go.
- The flat layout (`cost_centers` at the top with a `business_unit` field, `ou_links`, a
  top-level `accounts`) is no longer read; such a file fails with an error naming the key.
- Ids keep the code as written (`project: PRJ_Shop`); URNs lowercase it (D-056).

### `finops enrich validate`

On top of the WI-0104 and WI-0109 rules:

| Code | Severity | When | Subject |
|---|---|---|---|
| `taxonomy-urn-mismatch` | error | a written `urn` differs from the one built from the codes (D-058) | the written URN |
| `taxonomy-unknown-reference` | error | a `ref_urn_*` names no node of its type in the file | the URN |
| `taxonomy-duplicate-code` | error | two codes collide under the uniqueness rules above | the parent's URN and the code |
| `taxonomy-duplicate-alias` | error | two nodes of one type share an alias (D-057) | the type and the alias |
| `taxonomy-environment-not-in-app` | error | a default's Environment isn't under its App, as before | as before |

`taxonomy-lz-count` counts the entries of `lzs` or `lz_conns`; a file with both keys is an
error naming both.

### Resolution (D-061)

The steps of WI-0104 *Resolution*, with these changes:

1. **OU links** come from `ou_paths`. An entry matches the discovered OU whose id equals its
   `ref_id` when it has one, else whose path equals its `path`; none → `ou-not-found` (subject:
   the `path`, or the `ref_id` when it has no `path`). `app-on-leaf-ou` is gone: an App on any
   OU applies.
2. **App**: an account's `ref_urn_apps` › the App of its `ref_urn_env` (both `app_rule`
   `account`) › the deepest OU with `ref_urn_apps` (`ou:<path>`) › LZ default › global default ›
   `unalloc`. An OU with several Apps makes the account shared between them (`shared: true`),
   unless the account's own entry names one App.
3. **Environment**: an account's `ref_urn_env` (`account`) › the deepest OU with `ref_urn_env`
   (`ou:<path>`) › LZ default › global default › `unspec` or `unalloc`, as before. Nothing is
   guessed from names. An OU's `ref_urn_env` sets only the Environment, by the
   Environment code it ends in: the App in that URN must define the Environment (`validate`
   checks it as a reference) but does not set the account's App. An account's `ref_urn_env`
   sets its App only when the account has no `ref_urn_apps`.
4. **A moved account**: a taxonomy account whose `ref_id` is discovered under another OU keeps
   its `ref_urn_apps` and `ref_urn_env`, and `map` writes `account-ou-mismatch` (warning,
   subject the account id, message naming both paths). A taxonomy account not discovered at all
   is `account-not-discovered`, as before.

### Cloud URNs (D-059)

| Field / node | URN |
|---|---|
| `ou_urn`, OU node | `urn:tnt:<t>:lzc:<lz>:org:<OU code>`; the root is `org:root` |
| `account_urn`, account node | `<ou_urn of its discovered OU>:acc:<account code>` |
| `service_urn` | `<account_urn>:svc:<slug of service>`, as before |

- An OU's code is its `ou_paths` `code` when the taxonomy lists it, else the slug of its
  discovered name. An account's code is its `accounts` `code` when listed, else the slug of its
  discovered name.
- One function in `record/urn` (or `record/taxonomy`) builds both, and `map` and `lz-tree` call
  it.
- Two OUs with one URN, or two accounts with one URN, stop `map` and `lz-tree` with an error
  naming both OU paths or both account ids.
- The `node` record (WI-0110) keeps its fields. An OU node's `code` is its OU code, and an
  account node's `code` is its account code; `ref_id` keeps the AWS id.

### Reference cases

The planner converts every case under `finops/enrich/testdata/cases/` to the nested layout:
first with today's URNs on `claude/cases-nested` (for WO-0111-B; done 2026-11-03, `bb6afa8`,
based on `claude/cases-code`), then with D-059's URNs on `claude/cases-cloud-urn` (for
WO-0111-C). Case 04 is renamed `04-app-on-leaf-ou-applies` and rewritten for D-061: the leaf OU's App now applies, with
no finding. Every other case keeps its records and expectations; converted back to the flat
layout, each taxonomy equals the original. Accounts whose OU the case doesn't discover (the
share cases have no `ou` or `account` records) sit under the root entry `path: /`.

## WO-0111-A — Green `main`

**Assignee:** open
**Depends on:** —
**Packages:**
- `finops/enrich/validate_test.go` and `finops/enrich/testdata/` (a new fixture);
- `docs/user/taxonomy.md`

**Goal.** `go run ./scripts/check` passes on `main` again, without any file under `data/`.

**In scope.**
- `TestLoadNestedTaxonomy` reads a new committed fixture, `finops/enrich/testdata/nested-taxonomy.yaml`,
  with made-up codes and account ids (no real names, accounts or emails), instead of
  `data/real/cvie-taxonomy.yaml`. The fixture has the same shape as that file: two business
  units with cost centers, two projects, three Apps with Environments, one LZ under
  `lz_conns`, OUs with `ref_urn_apps` and accounts with `ref_urn_env`.
- `taxonomy.md`: point to that fixture instead of `data/real/cvie-taxonomy.yaml`, and say codes
  allow 3–50 characters (D-056).

**Out of scope.** Everything else in this Work Item.

**Done when.**
- No test or doc names a path under `data/` (`grep -rn "data/real" --include=*.go --include=*.md .` finds nothing outside `planning/`).
- The fixture holds no account id, name or email from the real Organization.
- `go run ./scripts/check` all PASS on Linux CI, with `docs PASS`.

**Verify.** `go run ./scripts/check`

## WO-0111-B — One loader for the nested taxonomy

**Assignee:** open
**Depends on:** WO-0111-A and WO-0110-B (they touch the same packages)
**Packages:**
- `record/taxonomy/` (new) and `record/urn/`;
- `finops/enrich/` (the loader moves out; `validate.go`, `resolve.go`) and
  `finops/enrich/testdata/` (the demo taxonomy, fixtures, and the cases merged from
  `claude/cases-nested`);
- `finops/sense/` and `xops/sense/` (use the loader; their own parsing goes);
- `xops/enrich/` (use the loader);
- every other taxonomy fixture;
- `docs/`, `README.md`, `Taskfile.yml` and `.vscode/launch.json`.

**Reference cases.** Before dispatch, the planner converts every case to the nested layout on
`claude/cases-nested`, keeping today's URNs. Merge that branch into yours with a merge commit,
and don't edit any file under `testdata/cases/`.

**Goal.** *Taxonomy file*, *One loader*, *`finops enrich validate`* and *Resolution* above.

**In scope.**
- `record/taxonomy`: the nested model and loader; table tests for every key, the read-not-used
  keys, an unknown key, a flat file, and `lzs` with `lz_conns`.
- Every command that takes `--taxonomy` reads it through `record/taxonomy`.
- `validate`: the rules in the table above, each with a test.
- `map`: resolution as above, with no name matching.
- The demo taxonomy in the nested layout; every example in `docs/` and `README.md` follows.

**Out of scope.** The cloud URNs (WO-0111-C). `owner_teams` and the other read-not-used keys
beyond reading them. `report tree --format taxonomy` beyond keeping it working.

**Done when.**
- Every case under `finops/enrich/testdata/cases/` passes, unchanged from `claude/cases-nested`.
- Tests: the loader table; each `validate` rule above, including `CVD-Spoke` with
  `urn: …:app:cvd-spk` giving `taxonomy-urn-mismatch`; an OU with two Apps making its account
  shared; an App on a leaf OU applying; an OU named `Prod` without `ref_urn_env` giving no
  Environment from its name; a moved account giving `account-ou-mismatch`.
- `finops/enrich/` has at most 5 non-test `.go` files, and no taxonomy parsing is left in
  `finops/sense/` or `xops/sense/`.
- On the mock pipeline every URN is unchanged from `main` (shown in the PR).
- Release criteria 1 to 5, 7 and 8 still hold on the branch.
- `go run ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run ./scripts/check`

## WO-0111-C — Cloud URNs from OU and account codes

**Assignee:** open
**Depends on:** WO-0111-B
**Packages:**
- `record/urn/` or `record/taxonomy/` (the URN function);
- `finops/enrich/` and `finops/enrich/share/` (`ou_urn`, `account_urn`, `service_urn`, and the
  collision error) and `finops/enrich/testdata/` (the cases merged from
  `claude/cases-cloud-urn`);
- `xops/enrich/`, `xops/report/tree/` and their `testdata/` golden files;
- `record/schema/` if a field description changes;
- every other fixture and golden file with an OU or account URN;
- `docs/`.

**Reference cases.** Before dispatch, the planner updates the cases' expected URNs to D-059 on
`claude/cases-cloud-urn`. Merge that branch into yours with a merge commit, and don't edit any
file under `testdata/cases/`.

**Goal.** *Cloud URNs* above, in `map`, `share` and `lz-tree`.

**In scope.**
- The one URN function, table-tested: a listed OU, an unlisted OU, the root, a listed account
  and an unlisted account.
- The collision errors in `map` and `lz-tree`, each tested with two OUs named `Prod` under
  different parents, and two accounts with one name in one OU.
- Golden files, examples and `records.md` follow the new URNs.

**Out of scope.** URN history or a migration of old URNs (nothing is stored yet).

**Done when.**
- Every case under `finops/enrich/testdata/cases/` passes, unchanged from
  `claude/cases-cloud-urn`.
- On the mock pipeline, for every account, `map`'s `ou_urn` and `account_urn` equal the URN of
  the matching `lz-tree` node, and no URN holds a 12-digit account id (shown in the PR).
- The collision tests above stop with both paths or both ids in the error.
- Release criteria 1 to 5, 7 and 8 still hold on the branch.
- `go run ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run ./scripts/check`
