# Finding codes

Commands emit `finding` records for issues that do not stop the pipeline (warnings) or that
make `finops enrich validate` exit 1 (errors).

| Code | Severity | Command | Subject | Meaning | What to do |
|---|---|---|---|---|---|
| `account-not-in-inventory` | warning | `finops enrich map` | Account ID | Cost row for an account missing from inventory. | Include the account in `--from-accounts` or pipe accounts. |
| `account-unmapped` | warning | `finops enrich map` | Account ID | No App could be resolved for the account. | Add OU link, account override, or fix taxonomy. |
| `app-on-leaf-ou` | warning | `finops enrich map` | OU path | App linked on a leaf OU that also has child OUs. | Move the link to the correct OU depth. |
| `app-tag-not-seen` | warning | `finops enrich share` | Account and period | Shared account had no App tag on cost rows. | Tag resources or accept equal split (`shared-untagged`). |
| `cost-history-short` | warning | `finops sense aws-cost` | First period returned | Fewer periods returned than requested. | Enable longer Cost Explorer history in AWS. |
| `environment-not-in-app` | warning | `finops enrich map` | Account and App | Environment on account is not defined under the App. | Fix taxonomy environments or account override. |
| `environment-unresolved` | warning | `finops enrich map` | Account ID | Environment could not be resolved. | Add OU environment link or account environment. |
| `lz-unknown` | warning | `finops enrich map` | Organization | Management account does not match any LZ in taxonomy. | Fix `lzs` or use the correct `--lz`. |
| `ou-not-found` | warning | `finops enrich map` | OU path | `ou_link` path not found in inventory. | Fix path spelling or refresh accounts. |
| `ou-path-ambiguous` | warning | `xops sense aws-accounts` | OU path | Two sibling OUs share the same name under one parent. | Rename OUs in AWS Organizations. |
| `project-tag-mismatch` | warning | `finops enrich map` | Account and tag | Project tag total disagrees with taxonomy expectation. | Fix tags or taxonomy Project mapping. |
| `shared-tag-unknown` | warning | `finops enrich share` | Account and tag fragment | Unknown App token in a multi-App tag. | Fix App tag values on shared cost. |
| `shared-tag-wrong-app` | warning | `finops enrich share` | Account and tag fragment | App tag does not match any allocation App. | Align tags with mapped Apps. |
| `shared-untagged` | warning | `finops enrich share` | Account and period | Shared account row had empty App tag; cost split equally. | Add App tags or accept the split. |
