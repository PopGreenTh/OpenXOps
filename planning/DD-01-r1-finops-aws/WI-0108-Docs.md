# WI-0108 — User and developer docs

## Outcome

A FinOps analyst goes from a clean clone to an Excel report, first on `--mock` and then on their
own AWS Organization, using the user docs alone. A developer can add a command or a record kind
using the developer docs alone. Both sets are in English only (D-044), and a check keeps the
references in step with the code.

## Contracts

### Audiences and files

| Audience | File | Holds |
|---|---|---|
| Everyone | `README.md` | What OpenXOps is, the licence line, a five-command quick start on `--mock`, and links to the docs below. Nothing else: the per-command sections move to `docs/user/commands.md` |
| User | `docs/user/getting-started.md` | The walkthrough: build, run the Release 1 pipeline on `--mock` to `finops.xlsx`, then the same pipeline against a real Organization |
| User | `docs/user/aws-setup.md` | One IAM policy as JSON holding exactly the actions the commands call, how to activate the App and Project tags as cost allocation tags, and how to turn on multi-year Cost Explorer data |
| User | `docs/user/taxonomy.md` | The taxonomy file, field by field; the resolution order (D-015) with a worked example taken from `finops/enrich/testdata/demo-taxonomy.yaml`; the reserved `-UNALLOC`, `-UNSPEC` and `-INVALID` codes |
| User | `docs/user/commands.md` | One section per command: synopsis, flags, kinds read, kinds written, findings it can emit, one example |
| User | `docs/user/records.md` | One table per record kind in `record/schema`: field, type, required, meaning |
| User | `docs/user/findings.md` | One row per finding code: severity, command, subject, meaning, what to do |
| Developer | `docs/develop.md` | Today's content, plus three new sections: how the pieces fit (short, linking `planning/ARCHITECTURE.md`), adding a record kind (schema table, `records.md` row, tests), and running the release checks (criteria 1–5 of `DELIVERABLE.md`) |
| Contributor | `CONTRIBUTING.md` | The licence (FSL-1.1-Apache-2.0, D-024) in plain words; that outside contributions need a signed CLA before merge; how to propose a change; that agents follow `AGENTS.md` |

### Doc check

`scripts/doccheck/` is a new program, run by `scripts/check` as a `docs` step that prints PASS
or FAIL like the other steps. It fails when:

1. a kind in `record/schema.ByKind`, or one of its fields, is missing from `docs/user/records.md`;
2. a finding code in any non-test `.go` file is missing from `docs/user/findings.md`. A code is
   a string literal passed as the code argument to `record.Finding`, or to a helper that passes
   its own parameter on as that argument (as `addErr` and `addWarn` do in
   `finops/enrich/validate.go` for the `taxonomy-*` codes). Besides the `taxonomy-*` codes, on
   `main` today these are `account-not-discovered`,
   `account-not-in-inventory`, `account-unmapped`, `app-on-leaf-ou`, `app-tag-not-seen`,
   `cost-history-short`, `environment-not-in-app`, `environment-unresolved`, `lz-unknown`,
   `ou-not-found`, `ou-path-ambiguous`, `project-tag-mismatch`, `shared-tag-unknown`,
   `shared-tag-wrong-app` and `shared-untagged`;
3. a command registered in the `xops` or `finops` binary has no section in
   `docs/user/commands.md`;
4. a relative link in `README.md`, `CONTRIBUTING.md` or `docs/**/*.md` points to a file that
   doesn't exist.

Its error names the file and the missing item. It reads the code with the Go standard library
(`go/parser` for finding codes, the `record/schema` package for kinds, the Cobra command tree for
commands), with no new dependency.

## WO-0108-A — User docs and the doc check

**Assignee:** open
**Depends on:** WO-0101-C
**Packages:** `README.md`, `docs/user/` (new), `scripts/doccheck/` (new), `scripts/check/`
(adding the `docs` step only)

**Goal.** The user docs and the doc check, as specified.

**In scope.**
- `README.md` cut down to the quick start and links; the per-command sections moved, updated, to
  `docs/user/commands.md`.
- The five `docs/user/` pages in the table above.
- `scripts/doccheck/` and its `docs` step in `scripts/check`, with a test for each of the four
  failures (a fixture tree under `scripts/doccheck/testdata/`).

**Out of scope.** `docs/develop.md` and `CONTRIBUTING.md` (WO-0108-B). Any change to a command.

**Done when.**
- `go run ./scripts/check` prints `docs PASS`, and the doc check's four failure tests pass.
- Every command in `docs/user/getting-started.md`'s `--mock` part runs as written on a clean
  clone with no AWS credentials and ends with `finops.xlsx` written (shown in the PR).
- The IAM policy in `docs/user/aws-setup.md` holds exactly the actions listed in the
  Contracts of WI-0102 and WI-0103, no more.

**Verify.** `go run ./scripts/check`

## WO-0108-B — Developer and contributor docs

**Assignee:** open
**Depends on:** WO-0101-C
**Packages:** `docs/develop.md`, `CONTRIBUTING.md` (new)

**Goal.** The developer and contributor docs, as specified.

**In scope.**
- `docs/develop.md`: the three new sections; existing sections checked against the code as it is
  on `main` and corrected where they drifted.
- `CONTRIBUTING.md` as in the table above.

**Out of scope.** `README.md` and `docs/user/` (WO-0108-A). A `LICENSE` file: its text needs the
licensor's name from the repo owner, so it isn't part of this Work Order.

**Done when.**
- Following "Add a record kind" step by step on a scratch branch gives a passing
  `scripts/check` (the PR lists the steps run; the scratch branch isn't pushed).
- Every command in `docs/develop.md` runs as written on a clean clone on Linux (shown in the PR).
- Every relative link in both files resolves.

**Verify.** `go run ./scripts/check`

## Delivered (as built)

Built by WO-0108-A (#48, `816973f`) and WO-0108-B (#49, `7802af5`), one review round each.

- `README.md` is a quick start. `docs/user/` holds getting started, AWS setup, taxonomy,
  commands, records and findings. `docs/develop.md` gains three sections, and `CONTRIBUTING.md`
  is new.
- `scripts/doccheck/` runs as the `docs` step of `scripts/check`. It also collects codes passed
  through a local helper function literal that forwards its first parameter to `record.Finding`
  (validate's `addErr` and `addWarn`).
- **Known differences.**
  - The doc check's failure fixtures carry full copies of `commands.md` and `records.md`.
  - There is no `LICENSE` file yet: it needs the licensor's name from the repo owner.

