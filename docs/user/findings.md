# Finding codes

Commands emit `finding` records on stdout. Severity `error` makes `finops enrich validate` exit 1;
warnings do not stop other commands. Subjects that name a taxonomy node use its id (`chk`, not
`APP-CHK`).

## Taxonomy validation (`finops enrich validate`)

| Code | Severity | Command | Subject | Meaning | What to do |
|---|---|---|---|---|---|
| `taxonomy-missing-field` | error | `finops enrich validate` | Section or code | A required field is empty in the taxonomy YAML. | Fill in the missing field. |
| `taxonomy-invalid-id` | error | `finops enrich validate` | Code | A `code` is not an id: ids are lowercase letters and digits joined by single dashes (`^[a-z0-9]+(-[a-z0-9]+)*$`). | Rename the code (`BU-RETAIL` becomes `retail`). |
| `taxonomy-lz-count` | error | `finops enrich validate` | `lzs` | The taxonomy does not have exactly one LZ: a tenant has one landing zone (D-052). | Keep one entry under `lzs`. |
| `taxonomy-reserved-code` | error | `finops enrich validate` | Code | A `code` is a reserved id: `unalloc`, `unspec` or `invalid`. | Rename the code. |
| `taxonomy-duplicate-code` | error | `finops enrich validate` | Code | The same code appears twice in a section. | Remove or rename the duplicate. |
| `taxonomy-unknown-reference` | error | `finops enrich validate` | Code or path | A reference points to a code that does not exist (for example unknown `business_unit`). | Fix the reference or add the target. |
| `taxonomy-duplicate-ou-link` | error | `finops enrich validate` | OU path | Two `ou_links` rows set the same App or Environment for one LZ and OU. | Keep one link per OU and field. |
| `taxonomy-environment-not-in-app` | error | `finops enrich validate` | Account and App | An account override sets an Environment not defined under the App. | Fix environments under the App or the override. |
| `taxonomy-unused` | warning | `finops enrich validate` | Code | Business unit, cost center, or project is never referenced. | Remove it or wire it into apps or projects. |

## Inventory, map, cost, and share

| Code | Severity | Command | Subject | Meaning | What to do |
|---|---|---|---|---|---|
| `account-not-discovered` | warning | `finops enrich map` | Account ID | Account in taxonomy overrides was not in the inventory. | Fix taxonomy or run `aws-accounts` again. |
| `account-not-in-inventory` | warning | `finops enrich map` | Account ID | Cost row for an account missing from inventory. | Include the account in `--from-accounts` or pipe accounts. |
| `account-unmapped` | warning | `finops enrich map` | Account ID | No App could be resolved for the account. | Add OU link, account override, or fix taxonomy. |
| `app-on-leaf-ou` | warning | `finops enrich map` | OU path | An `app` link on a leaf OU (no child OUs) is ignored; resolution continues with the next rule. The link's `environment`, if any, still applies. | Move the App link to a non-leaf OU if you intended it to set App. |
| `app-tag-not-seen` | warning | `finops enrich share` | App tag key | Shared cost had no row with a value for the taxonomy App tag key (often not activated as a cost allocation tag). | Activate the tag in AWS ([aws-setup.md](aws-setup.md)) and tag resources, or accept equal split (`shared-untagged`). |
| `cost-history-short` | warning | `finops sense aws-cost` | First period returned | Fewer periods returned than requested. | Enable longer Cost Explorer history in AWS. |
| `environment-not-in-app` | warning | `finops enrich map` | Account and App | Environment on account is not defined under the App. | Fix taxonomy environments or account override. |
| `environment-unresolved` | warning | `finops enrich map` | Account ID | Environment could not be resolved. | Add OU environment link or account environment. |
| `lz-unknown` | warning | `finops enrich map` | Organization | Management account does not match any LZ in taxonomy. | Fix the LZ's `management_account` in `lzs`. |
| `ou-not-found` | warning | `finops enrich map` | OU path | `ou_link` path not found in inventory. | Fix path spelling or refresh accounts. |
| `ou-path-ambiguous` | warning | `xops sense aws-accounts` | OU path | Two sibling OUs share the same name under one parent. | Rename OUs in AWS Organizations. |
| `project-tag-mismatch` | warning | `finops enrich map` | Account and tag value | A `project_tag_cost` row has a non-empty Project tag value that is not the account's Project (or one of them on a shared account). | Align Project tags with taxonomy or fix the taxonomy. |
| `shared-tag-unknown` | warning | `finops enrich share` | Account and tag part | Tag part matches no App in the taxonomy; cost goes to the `invalid` App. | Fix App tag values on shared cost. |
| `shared-tag-wrong-app` | warning | `finops enrich share` | Account and tag part | Tag part is a known App the account does not serve; cost goes to the `invalid` App. | Align tags with mapped Apps or taxonomy. |
| `shared-untagged` | warning | `finops enrich share` | Account and period | Shared account row had empty App tag; cost split equally across account Apps. | Add App tags or accept the split. |
