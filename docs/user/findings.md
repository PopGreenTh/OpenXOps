# Finding codes

Commands emit `finding` records on stdout. Severity `error` makes `finops enrich validate` exit 1;
warnings do not stop other commands. Subjects that name a taxonomy node use its id (`chk`, not
`APP-CHK`).

## Taxonomy validation (`finops enrich validate`)

| Code | Severity | Command | Subject | Meaning | What to do |
|---|---|---|---|---|---|
| `taxonomy-missing-field` | error | `finops enrich validate` | Section or code | A required field is empty in the taxonomy YAML. | Fill in the missing field. |
| `taxonomy-invalid-id` | error | `finops enrich validate` | Code | A `code` breaks the code rule: 3 to 50 characters from letters, digits and `. + = @ _ -` (`^[A-Za-z0-9.+=@_-]{3,50}$`). | Rename the code (`qa` becomes `quality`, `my shop` becomes `my_shop`). |
| `taxonomy-lz-count` | error | `finops enrich validate` | `lzs` | The taxonomy does not have exactly one LZ: a tenant has one landing zone (D-052). | Keep one entry under `lzs`. |
| `taxonomy-reserved-code` | error | `finops enrich validate` | Code | A `code` is a reserved id: `unalloc`, `unspec` or `invalid`, in any case (`UNALLOC` too). | Rename the code. |
| `taxonomy-duplicate-code` | error | `finops enrich validate` | The parent's URN and the code | Two codes collide where they must be unique, compared in any case (`Shop` and `shop`, because App tags already match codes in any case): business units, projects, Apps and LZs in the tenant; cost centers and owner teams in their parent; Environments in their App; OUs in their LZ; accounts in their OU. | Remove or rename the duplicate. |
| `taxonomy-duplicate-alias` | error | `finops enrich validate` | The node type and the alias | The same alias appears twice on nodes of the same type in the tenant. | Remove or rename the duplicate alias. |
| `taxonomy-unknown-reference` | error | `finops enrich validate` | The URN, or `defaults` | A `ref_urn_*` names no node of its type in the file (`ref_urn_bu`, `ref_urn_cct`, `ref_urn_apps`, `ref_urn_env`), or a `default` names an App the file does not define. | Fix the URN or add the node. |
| `taxonomy-urn-mismatch` | error | `finops enrich validate` | The written URN | A `urn` written in the file differs from the one built from the codes (`code: CVD-Spoke` gives `…:app:cvd-spoke`, not `…:app:cvd-spk`); an OU's is `…:org:<OU code>`, an account's `<OU URN>:acc:<account code>`. | Fix the `urn` or the `code`, or delete the `urn`: it is optional. |
| `taxonomy-lz-ref-id` | error | `finops enrich validate` | The LZ URN | The LZ's `ref_id` differs from its `management_account`. | Make them equal, or delete one. |
| `taxonomy-duplicate-ou-link` | error | `finops enrich validate` | OU path or `ref_id` | Two `ou_paths` entries of the LZ name the same OU, by path or by `ref_id`. | Keep one entry per OU. |
| `taxonomy-environment-not-in-app` | error | `finops enrich validate` | `defaults`, `<lz> default` or the account code | A default's Environment, or an account entry's `ref_urn_env`, is not under the App it goes with. | Fix the Environment or the App. |
| `taxonomy-unused` | warning | `finops enrich validate` | Code | Business unit, cost center, or project is never referenced. | Remove it or wire it into apps or projects. |

## Inventory, map, cost, and share

| Code | Severity | Command | Subject | Meaning | What to do |
|---|---|---|---|---|---|
| `account-not-discovered` | warning | `finops enrich map` | Account ID | An account entry under `ou_paths` was not in the inventory. | Fix taxonomy or run `aws-accounts` again. |
| `account-ou-mismatch` | warning | `finops enrich map` | Account ID | The account was discovered under a different OU than the one its entry sits under. Its `ref_urn_apps` and `ref_urn_env` still apply. | Move the entry under the account's OU, or move the account back. |
| `account-not-in-inventory` | warning | `finops enrich map` | Account ID | Cost row for an account missing from inventory. | Include the account in `--from-accounts` or pipe accounts. |
| `account-unmapped` | warning | `finops enrich map` | Account ID | No App could be resolved for the account. | Add `ref_urn_apps` to its OU or its account entry, or set a default. |
| `app-tag-not-seen` | warning | `finops enrich share` | App tag key | Shared cost had no row with a value for the taxonomy App tag key (often not activated as a cost allocation tag). | Activate the tag in AWS ([aws-setup.md](aws-setup.md)) and tag resources, or accept equal split (`shared-untagged`). |
| `cost-history-short` | warning | `finops sense aws-cost` | First period returned | Fewer periods returned than requested. | Enable longer Cost Explorer history in AWS. |
| `environment-not-in-app` | warning | `finops enrich map` | Account and App | The Environment resolved for the account is not defined under the App. | Fix the Environments under the App or the reference. |
| `environment-unresolved` | warning | `finops enrich map` | Account ID | Environment could not be resolved. Nothing is guessed from an OU or account name. | Add `ref_urn_env` to its OU or its account entry, or set a default. |
| `lz-unknown` | warning | `finops enrich map` | Organization | Management account does not match any LZ in taxonomy. | Fix the LZ's `management_account` in `lzs`. |
| `ou-not-found` | warning | `finops enrich map` | OU path, or `ref_id` when the entry has no path | An `ou_paths` entry matches no discovered OU: by `ref_id` when it has one, else by `path`. | Fix the path or `ref_id`, or refresh accounts. |
| `ou-path-ambiguous` | warning | `xops sense aws-accounts` | OU path | Two sibling OUs share the same name under one parent. | Rename OUs in AWS Organizations. |
| `project-tag-mismatch` | warning | `finops enrich map` | Account and tag value | A `project_tag_cost` row has a non-empty Project tag value that is not the account's Project (or one of them on a shared account). | Align Project tags with taxonomy or fix the taxonomy. |
| `shared-tag-unknown` | warning | `finops enrich share` | Account and tag part | Tag part matches no App in the taxonomy; cost goes to the `invalid` App. | Fix App tag values on shared cost. |
| `shared-tag-wrong-app` | warning | `finops enrich share` | Account and tag part | Tag part is a known App the account does not serve; cost goes to the `invalid` App. | Align tags with mapped Apps or taxonomy. |
| `shared-untagged` | warning | `finops enrich share` | Account and period | Shared account row had empty App tag; cost split equally across account Apps. | Add App tags or accept the split. |
