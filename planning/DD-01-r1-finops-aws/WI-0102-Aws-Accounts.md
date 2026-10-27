# WI-0102 — `xops sense aws-accounts`

## Outcome

`xops sense aws-accounts` reads an AWS Organization (root, nested OU tree, accounts) and writes
`ou` and `account` records. `--mock` gives a demo organization with no AWS access.

## Contracts

`xops sense aws-accounts [--mock] [--profile P] [--region R] [--lz CODE --taxonomy FILE]`

- AWS calls, read-only, from the management account or a delegated administrator:
  `organizations:DescribeOrganization`, `ListRoots`, `ListOrganizationalUnitsForParent`,
  `ListAccountsForParent`. Credentials from the default chain or `--profile`; region default
  `us-east-1`. Every call goes through `aws/`'s retry wrapper (exponential backoff on throttling
  and 5xx), all pages are read, at most 5 calls run at once.
- `--lz CODE` (with `--taxonomy`) checks that the organization's management account is that
  LZ's and adds `lz` to every record; a mismatch is an error.

### Records

OUs first, then accounts; each sorted by `id`. Every record has `csp` `aws`, `org_id` and
`management_account` (and `lz` with `--lz`).

| `kind` | Fields |
|---|---|
| `ou` | `id`, `name`, `parent_id`, `path`, `id_path`, `has_children` |
| `account` | `id`, `name`, `email`, `status`, `parent_id`, `ou_path`, `ou_id_path` |

- The root is an `ou` record with `name` `Root`, `parent_id` `""` and `path` `/`.
- An OU's `path` is the names from the root down joined by `/` (`/Workloads/Checkout`); its
  `id_path` is the IDs the same way (`r-ab12/ou-ab12-1/ou-ab12-2`). An account's `ou_path` and
  `ou_id_path` are its parent's.
- Every account status is listed (`ACTIVE`, `SUSPENDED`, …); the management account appears
  under the root.
- Two sibling OUs with the same name: finding `ou-path-ambiguous` (warning, subject: the path).
- Two runs over the same organization give the same output.

### Mock

`aws/testdata/demo-org.json`, embedded in the binary. One organization, management account
`111111111111`, with at least: `Security` (two accounts); `Workloads` holding two App OUs
(`Checkout`, `Payments`), each with `Prod` and `NonProd` child OUs holding one account each;
`Shared` with one account that will serve two Apps; one account directly under the root besides
the management account. WI-0104's demo taxonomy is written for this organization.

## WO-0102-A — Organizations discovery and mock

**Assignee:** open
**Depends on:** WO-0101-A
**Packages:** `aws/`, `xops/sense/`, `xops/main.go`, `Taskfile.yml`, `.vscode/launch.json`,
`README.md`, `go.mod`, `go.sum`

**Goal.** `xops sense aws-accounts` as specified, live and `--mock`.

**In scope.**
- `aws/`: config loading, the retry wrapper, the Organizations client behind an interface, the
  embedded demo organization.
- The command, path building, `ou-path-ambiguous`, `--lz`.
- Task `run:xops:sense-aws-accounts` and a launch entry, both on `--mock`.
- Tests: path and ID-path building; ambiguity; discovery against a fake client (nested OUs,
  paging, retry on throttling); golden file `xops/sense/testdata/aws-accounts.jsonl` for
  `--mock`.
- README: the command, its records and the IAM actions.

**Out of scope.** Cost (WI-0103). Other clouds.

**Done when.**
- `xops sense aws-accounts --mock` matches the golden file, twice in a row.
- The fake-client tests pass, including a throttled call that succeeds on retry.
- The launch entry stops at a breakpoint in the discovery code (said in the PR).

**Verify.** `go run ./scripts/check` and `go run ./xops sense aws-accounts --mock`
