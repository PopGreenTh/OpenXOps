# AWS setup

Use a read-only IAM policy on the management account (or a delegated administrator) that matches
what OpenXOps calls. Activate cost allocation tags and Cost Explorer history before monthly runs.

## IAM policy

The commands need exactly these actions (no write access):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "OpenXOpsRelease1ReadOnly",
      "Effect": "Allow",
      "Action": [
        "ce:GetCostAndUsage",
        "organizations:DescribeOrganization",
        "organizations:ListRoots",
        "organizations:ListOrganizationalUnitsForParent",
        "organizations:ListAccountsForParent"
      ],
      "Resource": "*"
    }
  ]
}
```

- **Organizations** (`xops sense aws-accounts`, and `DescribeOrganization` when `--taxonomy` is used
  on cost): list the organization tree read-only.
- **Cost Explorer** (`finops sense aws-cost`): `ce:GetCostAndUsage` only.

Attach the policy to the role or user whose credentials you pass with `--profile` or the
default chain.

## Cost allocation tags

In **Billing → Cost allocation tags**, activate the tag keys you use for App and Project (defaults
`App` and `Project`, or the keys in your taxonomy `tags` section). Untagged usage appears as empty
tag values in cost records.

## Cost Explorer history

Enable **Cost Explorer** and turn on multi-year data (at least **24 months** for monthly
granularity). If history is shorter than requested, `finops sense aws-cost` emits a
`cost-history-short` finding and returns what AWS provides.

## Credentials

Export credentials or use `~/.aws/credentials` with `AWS_PROFILE`. Region defaults to `us-east-1`
for Organizations; Cost Explorer is global but the SDK still uses the configured region.
