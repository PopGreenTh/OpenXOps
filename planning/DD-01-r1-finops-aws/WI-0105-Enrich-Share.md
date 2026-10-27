# WI-0105 — `finops enrich share`

## Outcome

`finops enrich share` splits each shared account's cost between its Apps by the App tag
(D-016, D-021), so every cost record ends up owned by exactly one App, and the total never
changes.

## Contracts

`finops enrich share --taxonomy FILE [--from-cost FILE]`

Reads `allocation` and `cost` records (pipe or `--from-cost`; `map`'s output holds both).
Every record other than a shared account's `cost` passes through unchanged.

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
**Packages:** `finops/enrich/`, `Taskfile.yml`, `.vscode/launch.json`, `README.md`

**Goal.** `share` as specified.

**In scope.**
- The command, the split, the remainder, the conservation check.
- Task and launch entry on a fixture file of `map` output.
- Tests: every share reference case; the conservation check failing on a hand-broken split.
- README: the share rules with an example.

**Out of scope.** Splitting by anything but the App tag.

**Done when.**
- Every `cases/share/` case passes, unchanged.
- `… | finops enrich map … | finops enrich share --taxonomy …` on the mocks prints no `cost`
  record with `shared` true and an empty `app`.
- The conservation-failure test passes.

**Verify.** `go run ./scripts/check`
