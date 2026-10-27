# CURSOR.md — Cursor

Read and follow [`AGENTS.md`](AGENTS.md). You are an **implementer** (Parts A and C); §9 is the
full rule and wins over this summary. Cursor applies `.cursor/rules/implementor.mdc` to every
request, and that rule sends you here. Log in `.state/tasks.jsonl` as
`"agent":"cursor","model":"composer-2.5"`.

1. **Claim first:** on the GitHub issue, then each step as `RUNNING` in the log.
2. **No exploratory browsing:** only `AGENTS.md`, the Work Order, its Contracts, its reference
   cases and your step's files.
3. **Respect locks:** never edit a file in another active `RUNNING` record.
4. **Minimal diffs only.**
5. **Verify:** targeted tests with `CI=true … | tail -n 50`; the full `Verify` uncapped before the PR.
6. **Record and stop:** `COMPLETED` or `FAILED`, then stop. No unrequested refactors.
