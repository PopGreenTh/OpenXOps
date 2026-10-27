# WI-0107 — `finops analyze changes`

## Outcome

`finops analyze changes` compares two `map` outputs, today's and an earlier one, and lists what
changed: accounts added, removed, moved between OUs, or remapped to another App or Environment,
with the cost that moved (D-021).

## Contracts

`finops analyze changes --before FILE [--from-map FILE]`

- **Now**: `account` and `allocation` records (and `cost`, if present) from the pipe or
  `--from-map`. **Before**: the same kinds from `--before FILE`, an earlier `map` (or `share`)
  output. Both are required.
- One `change` record per difference, then the input records unchanged:

| Field | Value |
|---|---|
| `subject_type` | `account` or `allocation` |
| `subject` | account ID, or `<account ID>/<app>` |
| `change` | `added`, `removed`, `moved`, `remapped` |
| `field` | `moved`: `ou_path`; `remapped`: `app`, `environment`, `project`, `business_unit`, `cost_center` or `lz`; else `""` |
| `old`, `new` | the values before and now (`""` for added or removed) |
| `cost_moved`, `currency` | with `cost` records in the input: the subject's cost in them, per currency (one `change` per currency); else `null` and `""` |
| `period_from`, `period_to` | the range of the `cost` records counted, else `""` |

- An account whose single App changed is one `remapped` change on the account (`field` `app`),
  not a removed plus an added allocation. An account that gained or lost an App (shared) gives
  `added` or `removed` allocations.
- **Account properties are reported on the account.** An account has one Environment and one LZ
  (D-015), so a change of `environment` or `lz` is one `remapped` change with `subject_type`
  `account`, whatever the number of its allocations. `project`, `business_unit` and
  `cost_center` belong to an App, so they are `allocation` changes; for a single-App account
  whose App changed they are not reported separately (the `app` change covers them).
- **Whose cost counts.** For an `account` subject: all its `cost` records. For an `allocation`
  subject: only the `cost` records with that account **and** that `app`. Before `share`, a
  shared account's cost has `app` `""`, so its allocations count no cost (`cost_moved` `0`), and
  the account's cost is never counted twice.
- Sorted by `subject_type`, `subject`, `change`, `field`.

## WO-0107-A — Changes between two map outputs

**Assignee:** open
**Depends on:** WO-0104-A
**Packages:** `finops/analyze/`, `finops/main.go`, `Taskfile.yml`, `.vscode/launch.json`,
`README.md`

**Goal.** `changes` as specified.

**In scope.**
- The command; a task and launch entry on two fixture `map` outputs.
- Debug logging as WI-0101 requires (`record.LogStart`); deterministic output; unhandled kinds
  passed through (`record.PassThrough`).
- Table-driven tests: each `change` kind; a shared account gaining an App; with and without
  cost; two currencies.

**Out of scope.** Trend and impact (Release 2). Reading a history store (Release 2).

**Done when.**
- The tests above pass.
- Two `map` runs over `finops/enrich/testdata/demo-accounts.jsonl` plus a small cost fixture
  (`finops/analyze/testdata/cost.jsonl`, written by this Work Order), the second run with the
  Environment link of `/Workloads/Checkout/Prod` changed in a copy of the demo taxonomy, give
  exactly one `remapped` change (`field` `environment`) for that OU's account, with its cost
  (shown in the PR).

**Verify.** `go run ./scripts/check`
