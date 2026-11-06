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
            ref_name: shop-prod                         # optional: the name in AWS (D-070)
            ref_urn_env: urn:tnt:testtnt:prj:shop:app:chk:env:prod
defaults: { app: chk, environment: nonprod }            # optional, codes
tags: { app: App, project: Project }                    # optional; these are the defaults
```

- **Every node** has `code` and `name`, and may have `urn` and `alias`. Cloud nodes (`lzs`,
  `ou_paths`, `accounts`) may have `ref_id` and `ref_name`. An LZ's `ref_id`, when given, equals its
  `management_account`. `ref_name` is the name the cloud uses (*Cloud names*, from WO-0111-D);
  it gets no code rule.
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
- An account that has cost but isn't in the inventory (`account-not-in-inventory`) sits under
  `org:unknown`; its code is its `accounts` `code` when the taxonomy lists it, else its account
  id, since no name is known. Under an unknown LZ (`lzc:unalloc`) the taxonomy isn't used, so
  codes come from discovered names.
- One function in `record/urn` (or `record/taxonomy`) builds both, and `map` and `lz-tree` call
  it.
- Two OUs with one URN, or two accounts with one URN, stop `map` and `lz-tree` with an error
  naming both OU paths or both account ids.
- The `node` record (WI-0110) keeps its fields. An OU node's `code` is its OU code, and an
  account node's `code` is its account code; `ref_id` keeps the AWS id.

### Cloud names (D-070)

From WO-0111-D. A cloud node has two names: `name` is the tenant's, `ref_name` the cloud's.

| Node | `name` | `ref_name` |
|---|---|---|
| LZ | its taxonomy `name` | its taxonomy `ref_name`, else the discovered `name` of its management account |
| OU (the root included) | its `ou_paths` `name` when listed, else its discovered `name` | its `ou_paths` `ref_name` when given, else its discovered `name` |
| Account | its `accounts` `name` when listed, else its discovered `name` | its `accounts` `ref_name` when given, else its discovered `name` |

- An OU entry matches its discovered OU as in *Resolution* (by `ref_id`, else by `path`); an
  account entry by `ref_id`.
- **`ref-name-mismatch`** (warning): a taxonomy `ref_name` and the discovered name are both
  non-empty and differ, compared exactly. The taxonomy's `ref_name` stays on the node. Subject:
  the node's `ref_id` (the management account for an LZ, the OU or root id, the account id).
  Message: both names, taxonomy first. `map` and `lz-tree` both write it, once per node.
- Nothing discovered and no taxonomy `ref_name` (an account outside the inventory, under
  `org:unknown`) gives `ref_name` `""`.
- Codes and URNs don't change: an unlisted node's code is still the slug of its discovered name
  (D-059), never of a taxonomy `ref_name`.
- `allocation`'s `account_name` stays the discovered name.

### Reference cases

The planner converts every case under `finops/enrich/testdata/cases/` to the nested layout:
first with today's URNs on `claude/cases-nested` (for WO-0111-B; done 2026-11-03, `bb6afa8`,
based on `claude/cases-code`), then with D-059's URNs on `claude/cases-cloud-urn` (for
WO-0111-C). Case 04 is renamed `04-app-on-leaf-ou-applies` and rewritten for D-061: the leaf OU's App now applies, with
no finding. Every other case keeps its records and expectations; converted back to the flat
layout, each taxonomy equals the original. Accounts whose OU the case doesn't discover (the
share cases have no `ou` or `account` records) sit under the root entry `path: /`.

### As built in WO-0111-B (#85), where the spec was silent

- `taxonomy-duplicate-code` has the subject `<parent URN> <code>`; an account's parent is
  `<LZ URN>:org:<OU code>`. `taxonomy-duplicate-alias` has the subject `<URN type> <alias>`
  (`app shop`).
- The code rule and reserved ids bind business unit, cost center, owner team, project, App,
  Environment and LZ codes. OU and account codes, and written OU and account URNs, are checked
  from WO-0111-C, once D-059 builds them.
- `tenant` is a mapping with `code` and `name`; a bare `tenant: testtnt` is an error.
- `taxonomy-duplicate-ou-link` stays, for two `ou_paths` entries naming the same `path` or
  `ref_id`. An entry with no `ref_urn_*` is valid, so a scaffold passes `validate`.
- `map` skips a `ref_urn_*` that names no node and goes on to the next source; `validate`
  reports it, and `map` refuses a taxonomy with errors.

### As built in WO-0111-C (#89), where the spec was silent

- The collision check covers every OU and account, under an unlisted LZ (`lzc:unalloc`) too, so
  two unknown organizations whose roots both give `org:root` stop `map`.
- An account's written `urn` is compared with the OU its taxonomy entry sits under; `validate`
  doesn't see the discovered OU.
- `taxonomy-lz-ref-id` needs both `ref_id` and `management_account`.
- Two records for the same OU path, or the same account id, aren't a collision.
- `lz-tree` finds the OU above an account by `parent_id`, else by `ou_path`; an OU missing from
  the input is made from the path (`org:<slug of its last segment>`).
- The demo taxonomies list the mock account outside the inventory (`299999999999`) as
  `outside-inventory`, so no demo URN holds a 12-digit id.

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

- Found while merging WO-0110-B (#77): `793fbf8`'s nested reader takes each code from its URN
  segment, lowercased, so `PRJ_Shop` came out as `prj_shop` in records and a `code:` that
  disagreed with its URN was ignored. The new loader reads `code` as written and only checks a
  written `urn` (D-058).

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
- `record/taxonomy/` and `finops/enrich/validate.go` (the `validate` items below);
- `finops/enrich/` and `finops/enrich/share/` (`ou_urn`, `account_urn`, `service_urn`, and the
  collision error) and `finops/enrich/testdata/` (the cases merged from
  `claude/cases-cloud-urn`);
- `xops/enrich/`, `xops/report/tree/` and their `testdata/` golden files;
- `record/schema/` if a field description changes;
- every other fixture and golden file with an OU or account URN;
- `docs/`.

**Reference cases.** Before dispatch, the planner updates the cases' expected URNs to D-059 on
`claude/cases-cloud-urn`. Done 2026-11-03 (`af6d673`, based on `main` after WO-0111-B): cases 01,
08, 09 and 12 change, for example `org:workloads/checkout/prod:acc:200000000001` becomes
`org:prod:acc:chk-prod`, and `org:root:acc:200000000002` becomes `org:root:acc:shared` (a
listed account's code). No other field changes. Merge that branch into yours with a merge commit, and don't edit any
file under `testdata/cases/`.

**Goal.** *Cloud URNs* above, in `map`, `share` and `lz-tree`.

**In scope.**
- The one URN function, table-tested: a listed OU, an unlisted OU, the root, a listed account
  and an unlisted account.
- The collision errors in `map` and `lz-tree`, each tested with two OUs named `Prod` under
  different parents, and two accounts with one name in one OU.
- Golden files, examples and `records.md` follow the new URNs.
- `share` copies `cost_center_urn` from the account's allocation instead of rebuilding it from
  a cost center code, which is unique only within its business unit (found in #85;
  `share.Options.CostCenterBU` maps a code two business units share to the first).
- `validate`: an LZ whose `ref_id` differs from its `management_account` is
  `taxonomy-lz-ref-id` (error, subject the LZ URN). OU and account codes get the code rule, and
  a written OU or account `urn` is compared with the one D-059 builds (`taxonomy-urn-mismatch`).

- `report tree --format taxonomy` writes each OU's and account's `code` as its slug, not its
  display name (`793fbf8` writes `Shared Services`), so a scaffold read back passes the code
  rule and gives the same URNs.

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

## WO-0111-D — The cloud's own name in `ref_name`

**Assignee:** open
**Depends on:** WO-0111-C
**Packages:**
- `record/taxonomy/` (`ref_name` on LZs, OU entries and accounts);
- `record/schema/` (`ref_name` on `node`);
- `finops/enrich/` (`ref-name-mismatch` in `map`) and `finops/enrich/testdata/` (case 13,
  merged from `claude/cases-ref-name`);
- `xops/enrich/` (`name`, `ref_name` and `ref-name-mismatch` in `lz-tree`), `xops/report/tree/`
  (`ref_name` in the tree YAML and the taxonomy scaffold), and their `testdata/` golden files;
- `templates/taxonomy.yaml`, the demo taxonomies and `docs/`.

**Reference cases.** Map case 13, `13-ref-name-mismatch`, is on `claude/cases-ref-name`
(`5ce455b`, based on `main` at `472ef85`). It fails on `main` because the loader rejects
`ref_name`. Merge that branch into yours with a merge commit, and don't edit any file under
`testdata/cases/`.

**Goal.** *Cloud names* above ([D-070](../DECISIONS.md#d-070)).

**In scope.**
- The loader reads `ref_name` on `lzs`, `ou_paths` and `accounts`; it is optional and gets no
  code rule.
- `lz-tree` sets `name` and `ref_name` on every node by the *Cloud names* table, and writes
  `ref-name-mismatch`. `node` in `record/schema` gains `ref_name`, required.
- `map` writes `ref-name-mismatch` for listed OUs and accounts, and for the LZ when its
  management account is discovered.
- `report tree`: each LZ, OU and account shows `ref_name` in the tree YAML. `--format taxonomy`
  writes `name` and `ref_name` from the node, so a scaffold read back gives the same nodes and no
  `ref-name-mismatch`.
- Tests: `lz-tree` with a listed account whose `ref_name` differs, one whose `ref_name` matches,
  an unlisted account, and an account outside the inventory (`ref_name` `""`).

**Out of scope.** `ref_name` on `allocation` or `cost`; Business Value and Project nodes; name
history (Release 2, D-068).

**Done when.**
- Every case under `finops/enrich/testdata/cases/` passes, case 13 included, unchanged from
  `claude/cases-ref-name`.
- On the mock pipeline, every `node` has `ref_name`. Every account node's `ref_name` equals
  the `name` of its `account` record from `aws-accounts`, and `lz-tree` writes no
  `ref-name-mismatch` (shown in the PR).
- A scaffold from `report tree --format taxonomy` on the mock passes `validate` and, read back
  by `lz-tree`, gives the same `urn`, `name` and `ref_name` on every node (shown in the PR).
- Release criteria 1 to 5, 7 and 8 still hold on the branch.
- `go run ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run ./scripts/check`
