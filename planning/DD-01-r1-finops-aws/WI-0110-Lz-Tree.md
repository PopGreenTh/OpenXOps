# WI-0110 — The cloud tree: `xops enrich lz-tree` and `xops report tree`

## Outcome

A tenant can see and check its cloud hierarchy as URNs before any cost is mapped
([D-054](../DECISIONS.md#d-054)). `xops sense aws-accounts` stays raw. `xops enrich lz-tree`
turns its `ou` and `account` records into one flat `node` record per node: the LZ, every OU and
every account. `xops report tree` writes those records as a nested YAML file that people can
read.

```
xops sense aws-accounts --tenant cvd --profile P > accounts.jsonl
xops enrich lz-tree --tenant cvd --taxonomy cvd.yaml --from-accounts accounts.jsonl > tree.jsonl
xops report tree --tenant cvd --from-node tree.jsonl --out cvd-tree.yaml
```

## Contracts

### `xops enrich lz-tree`

`xops enrich lz-tree --tenant ID --taxonomy FILE [--from-accounts FILE]`

- Reads `ou` and `account` records (`--from-accounts` or stdin) and the taxonomy. The tenant
  checks of WI-0109 apply to both.
- The LZ is the taxonomy's one LZ (D-052). Every input record's `management_account` must equal
  that LZ's `management_account`; otherwise the command stops with an error naming the file, the
  line and both accounts.
- URNs are built with `record/urn`, exactly as `map` builds `lz_urn`, `ou_urn` and
  `account_urn` (D-046, D-047, D-053), so the two always agree.
- Writes only `node` records, in this order: the LZ node, then the OU nodes sorted by `urn`,
  then the account nodes sorted by `urn`. Input records are not passed through. Two runs over the
  same input give the same output.
- Every node has a non-empty `code`, `urn` and `name` ([D-056](../DECISIONS.md#d-056), from
  WO-0110-B). A `urn` that breaks the URN rule (WI-0109 *Ids*) stops the command, naming the
  node. A `code` that breaks the code rule, such as `it` for an OU named `IT`, is kept, and a
  warning naming the node goes to stderr.
- Plain functions for the tree (records in, records out); the command only wires flags and files
  (D-013).

### `node` record

| Field | Type | LZ node | OU node | Account node |
|---|---|---|---|---|
| `kind` | string | `node` | `node` | `node` |
| `tenant` | string | the tenant | the tenant | the tenant |
| `csp` | string | the LZ's `csp` | the record's `csp` | the record's `csp` |
| `type` | string | `lzc` | `org` | `acc` |
| `urn` | string | `urn:tnt:<t>:lzc:<lz>` | `…:lzc:<lz>:org:<OU path>` | `<parent OU's urn>:acc:<account id>` |
| `parent_urn` | string | `""` | the LZ's `urn` for the root; otherwise the parent OU's `urn` | the parent OU's `urn` |
| `code` | string | the LZ code | the slug of the OU's own name (`hub-sit`, `root`) ([D-056](../DECISIONS.md#d-056)) | the account id |
| `name` | string | the LZ's `name` | the OU's taxonomy `name` when listed, else its discovered `name` (`Root` for the root) | the account's taxonomy `name` when listed, else its discovered `name` |
| `ref_name` | string | the LZ's `ref_name`, else its management account's discovered `name` | the OU's `ref_name` in the taxonomy, else its discovered `name` | the account's `ref_name` in the taxonomy, else its discovered `name` |
| `ref_id` | string | the LZ's `management_account` | the AWS OU or root id (`ou-6ibz-…`, `r-6ibz`) | the account id |
| `path` | string | `""` | the OU's `path` (`/Platform/Hub/Hub-SIT`) | the account's `ou_path` |
| `status` | string | `""` | `""` | the account's `status` |
| `joined_at` | string | `""` | `""` | the account's `joined_at`, or `""` when the record has none ([D-055](../DECISIONS.md#d-055)) |
| `joined_method` | string | `""` | `""` | the account's `joined_method`, or `""` when the record has none (D-055) |
| `alias` | string | the LZ's `alias` | the OU's `alias` in the taxonomy | the account's `alias` in the taxonomy |

`node` goes into `record/schema` with every field above required, except `alias` (optional, `""`
when the taxonomy gives none; added by `793fbf8`, [D-057](../DECISIONS.md#d-057)). From WO-0111-C, the
OU and account `urn` and `code` follow WI-0111 *Cloud URNs* ([D-059](../DECISIONS.md#d-059)). From WO-0111-D, `name` and `ref_name`
follow WI-0111 *Cloud names* ([D-070](../DECISIONS.md#d-070)); `ref_name` is required and `""` only when
nothing is discovered and the taxonomy gives none.

### `xops report tree`

`xops report tree --tenant ID [--from-node FILE] [--out FILE]`

- Reads `node` records (`--from-node` or stdin) and writes one YAML document: to `--out`, or to
  stdout when `--out` is not given. An existing `--out` file is replaced (a tree is a snapshot,
  not a log to append to).
- Shape: the LZ at the top, every node's children under `children`, OUs before accounts, each
  group in `code` order. A node with no children has no `children` key.

  ```yaml
  tenant: cvd
  lz:
    type: lzc
    code: main
    name: Main
    urn: urn:tnt:cvd:lzc:main
    ref_id: "225989366329"
    children:
      - type: org
        code: root
        name: Root
        urn: urn:tnt:cvd:lzc:main:org:root
        ref_id: r-6ibz
        children:
          - type: org
            code: platform
            name: Platform
            urn: urn:tnt:cvd:lzc:main:org:platform
            ref_id: ou-6ibz-mhj79kx9
            children: [...]
          - type: acc
            code: "225989366329"
            name: cvd-org-mgmt
            urn: urn:tnt:cvd:lzc:main:org:root:acc:225989366329
            ref_id: "225989366329"
            status: ACTIVE
            joined_at: "2023-05-08T04:21:10Z"
            joined_method: CREATED
  ```
- An account shows its `name`, `status`, `joined_at` and `joined_method`. `status`,
  `joined_at` and `joined_method` are left out when they are `""`, so LZ and OU nodes don't
  show them.
- The command fails, writing nothing, when there isn't exactly one `lzc` node, or a node's
  `parent_urn` names no node in the input; the error names the file, the line and the URN.
- `xops/report/` already has 5 non-test files, so the YAML writer is a new package,
  `xops/report/tree/`; the command goes in `xops/report/commands.go`.

### Mock

`aws-accounts --mock` piped into `lz-tree` with the demo taxonomy gives the demo tree. Its golden
files are `xops/enrich/testdata/demo-tree.jsonl` and `xops/report/testdata/demo-tree.yaml`.

## WO-0110-A — `lz-tree` and the YAML tree report

**Assignee:** open
**Depends on:** WO-0109-C (the slug rule and the one-LZ taxonomy)
**Packages:**
- `xops/enrich/` (new), `xops/main.go`;
- `xops/report/commands.go`, `xops/report/tree/` (new) and `xops/report/testdata/`;
- `record/schema/` (the `node` kind);
- `docs/user/` (`commands.md`, `records.md`, `getting-started.md`), `README.md`, `Taskfile.yml`,
  `.vscode/launch.json`

**Goal.** Both commands as specified, on real input and on the mock.

**In scope.**
- `xops enrich lz-tree` and the `node` record.
- `xops report tree` and `xops/report/tree/`.
- Tasks `run:xops:enrich-lz-tree` and `run:xops:report-tree`, and a launch entry for each, on the
  mock.
- `getting-started.md`: the tree as an optional step after `aws-accounts`.
- Tests:
  - the demo tree against both golden files, twice in a row;
  - the LZ, a root account, a nested OU and an account in a CamelCase OU (`NonProd`);
  - an input record for another management account fails with its file, line and both accounts;
  - `report tree` with no `lzc` node, two `lzc` nodes, or a dangling `parent_urn` fails and
    writes nothing.

**Out of scope.** OU URN collisions (Q-009). `node` in `table`, `csv` and `excel`. Making `map`
read the tree (D-054).

**Done when.**
- On the mock: every `account` record from `aws-accounts` has exactly one `acc` node, every `ou`
  record has exactly one `org` node, and each node's `urn` equals the matching `ou_urn` or
  `account_urn` that `map` writes for the same account (shown in the PR).
- Every `acc` node in the demo tree has the `name`, `joined_at` and `joined_method` of its
  `account` record, and so does every account in the demo YAML (D-055); an account record
  without the two fields gives `""` and no YAML key (test).
- `xops report tree` writes the demo YAML, matching its golden file.
- The tests above pass.
- `go run ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run ./scripts/check`

## WO-0110-B — Codes, URNs, names and `ref_id` on every node

**Assignee:** open
**Depends on:** WO-0110-A, and WO-0111-A for a green `main`. Merge `main` into your branch: it
now holds commit `793fbf8` (nested taxonomy, `alias`; [D-057](../DECISIONS.md#d-057)) in the same
packages. Keep its `alias` and its nested reader as they are; WO-0111-B reworks them.
**Packages:**
- `record/urn/`, `record/schema/` (the `node` fields) and the `--tenant` check;
- `finops/enrich/` (`validate.go`, and where taxonomy codes become URN ids),
  `finops/enrich/share/` (the same), and `finops/enrich/testdata/` (fixtures, and the cases
  merged from `claude/cases-code`);
- `xops/enrich/`, `xops/report/tree/`, `xops/enrich/testdata/` and `xops/report/testdata/`;
- every other fixture and golden file that holds a `node` record;
- `docs/`, `README.md` and `.vscode/launch.json`.

**Reference cases.** Before dispatch, the planner renamed the codes shorter than 3 characters on
the branch `claude/cases-code`: Apps `a`, `b`, `c` became `aaa`, `bbb`, `ccc` in the share cases,
and Environment `qa` became `quality` in map case 05. Merge that branch into yours with a merge
commit, and don't edit any file under `testdata/cases/`.

**Goal.** The D-056 rules, as amended under *Ids* (WI-0109) and *`node` record* above.

**In scope.**
- `record/urn`: a code check (`^[A-Za-z0-9.+=@_-]{3,50}$`) and a URN check
  (`^[a-z0-9.:+=@_/-]{11,250}$`), each table-tested; URN ids built from codes are lowercased.
- `validate` and `--tenant` use the code check; the case rules for duplicates and reserved ids.
- `lz-tree`: `code` and `ref_id` as in the table; a node whose URN fails the URN check stops the
  command naming the node; a code that fails the code check is logged as a warning.
- `report tree`: the new keys, as in the example.
- `records.md`, `taxonomy.md`, `commands.md` and `findings.md`: the code and URN rules, with
  examples, and the renamed `node` fields.

**Out of scope.** A tenant node and the tenant's name. Codes for the Release 2 levels.
OU URN collisions (Q-009).

**Done when.**
- Every case under `finops/enrich/testdata/cases/` passes, unchanged from `claude/cases-code`.
- Tests:
  - the code check accepts `PRJ_Shop`, `a.b`, `x@y=z+1` and a 50-character code, and rejects
    `ab`, a 51-character code, `a/b` and `a b`;
  - the URN check accepts `urn:tnt:cvd` and the longest demo URN, and rejects a 10-character
    string, a 251-character string and an uppercase letter;
  - `validate` reports `taxonomy-invalid-id` for `ab`, `taxonomy-duplicate-code` for `Shop`
    next to `shop`, and `taxonomy-reserved-code` for `UNALLOC`;
  - `--tenant ab` stops before reading anything;
  - a project coded `PRJ_Shop` gives `project: PRJ_Shop` and
    `project_urn: urn:tnt:testtnt:prj:prj_shop`;
  - an OU named `IT` gives the node code `it`, the URN segment `it` and one warning on stderr.
- On the mock tree, no node has `id` or `source_id`, every OU node's `code` is the slug of its
  own name, and every `urn` is unchanged from WO-0110-A's golden file (shown in the PR).
- Release criteria 1 to 5, 7 and 8 still hold on the branch.
- `go run ./scripts/check` all PASS, with `docs PASS`.

**Verify.** `go run ./scripts/check`

## Delivered (as built)

Built by WO-0110-A (#71, `f237cd5`) by the Claude Code implementer session, after one round of
changes for D-055.

- `xops enrich lz-tree` (new package `xops/enrich/`) writes `node` records: the LZ, then OUs and
  accounts sorted by URN. Account nodes carry `name`, `status`, `joined_at` and `joined_method`.
  The URNs match `map`'s `lz_urn`, `ou_urn` and `account_urn`.
- `xops report tree` (new package `xops/report/tree/`) writes the nested YAML. It fails, writing
  nothing, on no or two `lzc` nodes, a dangling `parent_urn`, a duplicate URN or a cycle.
- On the real Organization: 1 LZ, 19 OUs and 16 accounts.
- Left open: `scripts/doccheck` doesn't list `xops enrich`, so a missing doc section for
  `lz-tree` would not fail `docs`.
