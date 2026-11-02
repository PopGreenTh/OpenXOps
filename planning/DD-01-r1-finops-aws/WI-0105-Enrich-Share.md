# WI-0105 — `finops enrich share`

## Outcome

`finops enrich share` splits each shared account's cost between its Apps by the App tag
(D-016, D-021), so every cost record ends up owned by exactly one App, and the total never
changes.

## Contracts

`finops enrich share --taxonomy FILE [--from-cost FILE]`

Reads `allocation` and `cost` records (pipe or `--from-cost`; `map`'s output holds both).
Every record other than a shared account's `cost` passes through unchanged.
Output order: the `cost` records (sorted as below), then `share`'s findings, then **every other
input record exactly once, unchanged, in input order**: `allocation`, `ou`, `account` and
earlier commands' findings included.

For each `cost` record with `shared` true, given the account's Apps from its `allocation`
records:

1. **Split the tag** on `,` and `__`; trim each part; drop empty parts. No part left → untagged.
2. **Untagged** → one record per account App, `share` `1/n`, `share_rule` `untagged-equal`;
   finding `shared-untagged` (subject `<account>/<period>`).
3. **Tagged with k parts** → one share of `1/k` per part (`share_rule` `tag` when k = 1,
   `tag-multi` when k > 1). Each part, matched to App codes ignoring case:
   - an App the account serves → that App;
   - an App in the taxonomy the account doesn't serve → `APP-INVALID`, finding
     `shared-tag-wrong-app` (subject `<account>/<part>`);
   - no App at all → `APP-INVALID`, finding `shared-tag-unknown` (subject `<account>/<part>`).
   Two parts landing on the same App stay two records.
4. Each output record copies the input `cost` record, sets `amount` to its part, adds
   `source_amount` (the input amount) and `share`, and fills `project`, `app`,
   `business_unit`, `cost_center`, `app_rule` from the App's `allocation` record. `APP-INVALID`
   gets `PRJ-UNALLOC`, `BU-UNALLOC`, `CC-UNALLOC`, `app_rule` `invalid`, and the account's
   Environment.
5. **Remainder.** Parts are ordered by App code (then by tag order); each but the last gets
   `amount × share`, the last gets the rest, so the parts add up to the input exactly.
6. **Conservation.** For every period, metric and currency, output `amount` total equals input
   total within `1e-6`, or the command fails (that is a bug, not a data problem).

A non-shared `cost` record gets `share` `1`, `share_rule` `single`, `source_amount` = `amount`.
If the input has at least one shared `cost` record and none of them carries a tag value:
finding `app-tag-not-seen` (subject: the App tag key from the taxonomy), the tag is probably not
activated for cost allocation.

### Reference cases

`finops/enrich/testdata/cases/share/*.yaml`: `taxonomy`, input `records` (`allocation` and
`cost`), `expect.costs` (fields listed; numbers within `1e-6`; both sides sorted by
`account_id`, `period`, `app`, `amount`) and `expect.findings`.

## WO-0105-A — Share by App tag

**Assignee:** open
**Depends on:** WO-0104-A
**Packages:** `finops/enrich/share/` (new), `finops/enrich/commands.go` (the `share` command
only), `finops/enrich/share_test.go` (new), `finops/enrich/testdata/` (new fixtures only, not
`cases/`), `Taskfile.yml`, `.vscode/launch.json`, `README.md`

`finops/enrich/` already has the 5 non-test `.go` files `AGENTS.md` §5 allows. So the split
itself (plain functions: records and the taxonomy's App codes in, records and findings out) goes
in a new package, `finops/enrich/share/`, which does not import `finops/enrich`. The `share`
command, its flags and taxonomy loading go in `finops/enrich/commands.go`, which calls
`share`. The reference-case test lives in `finops/enrich/share_test.go`, so it reads
`testdata/cases/share/` in place and can use the existing taxonomy loader.

**Goal.** `share` as specified.

**In scope.**
- The command, the split, the remainder, the conservation check.
- Task and launch entry on a fixture file of `map` output: `testdata/demo-map.jsonl`, the
  demo accounts' `map` output plus hand-written `cost` records for the shared account that cover
  every tag case (single App, `APP-CHK,APP-PAY`, lower case, a wrong App, an unknown value,
  empty).
- Debug logging as WI-0101 requires (`record.LogStart`); deterministic output; unhandled kinds
  passed through (`record.PassThrough`).
- Tests: every share reference case; the conservation check failing on a hand-broken split.
- README: the share rules with an example.

**Out of scope.** Splitting by anything but the App tag.

**Done when.**
- Every `cases/share/` case passes, unchanged.
- `finops enrich share --taxonomy finops/enrich/testdata/demo-taxonomy.yaml --from-cost
  finops/enrich/testdata/demo-map.jsonl` prints no `cost` record with `shared` true and an empty
  `app`. If WO-0103-A is on `main`, the PR also shows the same for the full mock pipeline
  (`aws-accounts | map | share`, with `aws-cost --mock`).
- The conservation-failure test passes.

**Verify.** `go run ./scripts/check`

## WO-0105-B — Pass-through fix

**Assignee:** open
**Depends on:** WO-0105-A
**Packages:** `finops/enrich/share/` only (`share.go`, and a new `share_test.go` inside it)

Found while reviewing WO-0101-B (#33). On `main`, `share.Split` writes every record it doesn't
handle twice: once from its own `other` slice and again from `record.PassThrough`. It also never
writes `allocation` records, because it lists `allocation` as handled. Measured with the mock
data: `map` gives `{ou: 10, account: 9, allocation: 10, cost: 1300, finding: 154}`. `share` on
that output gives `{ou: 20, account: 18, allocation: 0, cost: 1500, finding: 508}`, where
508 = 2 × 154 + 200 of `share`'s own.

**Goal.** `share`'s output order as the Contracts now state: every input record that isn't a
`cost` comes out exactly once, unchanged.

**In scope.**
- `share.Split`: its own output is the `cost` records and its findings; every other record
  passes through once (`record.PassThrough` with only `cost` handled), in input order.
  `allocation` records are still read for the split.
- `finops/enrich/share/share_test.go` (new, package `share`, table-driven): for an input holding
  `ou`, `account`, `allocation`, shared and non-shared `cost`, and a `finding` from an earlier
  command, each non-`cost` record appears in the output exactly once, unchanged, in input order,
  after `share`'s own records.

**Out of scope.** Anything in `finops/enrich/` outside `share/` (WO-0101-B is open there), and
any change to the split itself.

**Done when.**
- The new test passes, and every `cases/share/` case still passes unchanged.
- On the mock data, `share` over `map`'s output has exactly as many `ou`, `account` and
  `allocation` records as its input, and as many findings as the input plus `share`'s own
  (shown in the PR).

**Verify.** `go run ./scripts/check`

## Delivered (as built)

Built by WO-0105-A (#23, `c62476b`) and WO-0105-B (#36, `ebcb461`).

- The split lives in `finops/enrich/share/`, and the command lives in
  `finops/enrich/commands.go`. All 6 share reference cases pass, and so does the
  conservation-failure test.
- WO-0105-B fixed the pass-through. Every record that isn't a `cost` now comes out once, in
  input order, `allocation` included. Before the fix, unhandled records came out twice and
  `allocation` was dropped.
- On the mock data, the cost after `share` equals Cost Explorer's per period, metric and
  currency (largest difference 5e-13), and no `cost` record is left without an App.

Later changes:
- WO-0109-B (#59): split rows get the five App-level URNs from the allocation they land on. The
  `invalid` App sits under Project `unalloc`, and a cost center's URN sits under its own business
  unit. All 6 share cases pass with URNs.
