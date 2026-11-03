package enrich

import (
	"sort"

	"github.com/PopGreenTh/OpenXOps/record"
	"github.com/PopGreenTh/OpenXOps/record/taxonomy"
)

// Map applies taxonomy rules to inventory and cost records.
func Map(t taxonomy.Taxonomy, recs []record.Record) []record.Record {
	r := &resolver{t: t, ix: t.Index(), in: buildInventory(recs)}
	in := r.in
	var passthrough []record.Record
	var costs []record.Record
	var tagCosts []record.Record
	for _, r := range recs {
		switch r.Kind() {
		case "ou", "account":
			passthrough = append(passthrough, r)
		case "cost":
			costs = append(costs, r)
		case "project_tag_cost":
			tagCosts = append(tagCosts, r)
		}
	}

	var findings []record.Record
	r.buildViews(&findings)

	lzUnknownOrgs := map[string]struct{}{}
	var allocations []record.Record
	for _, ac := range sortedAccounts(in.accounts) {
		allocs, f := r.resolveAccount(ac, lzUnknownOrgs)
		allocations = append(allocations, allocs...)
		findings = append(findings, f...)
	}
	findings = append(findings, r.staleAccountEntries()...)

	byAcct := map[string][]record.Record{}
	for _, a := range allocations {
		id := strID(a["account_id"])
		byAcct[id] = append(byAcct[id], a)
	}

	for _, tc := range tagCosts {
		findings = append(findings, checkProjectTags(tc, byAcct)...)
	}

	var enrichedCosts []record.Record
	for _, c := range costs {
		out, f := r.enrichCost(c, byAcct)
		enrichedCosts = append(enrichedCosts, out)
		findings = append(findings, f...)
	}

	sortAllocations(allocations)
	sortFindings(findings)

	var out []record.Record
	out = append(out, passthrough...)
	out = append(out, allocations...)
	out = append(out, enrichedCosts...)
	out = append(out, findings...)
	return record.PassThrough(recs, []string{"ou", "account", "cost", "project_tag_cost"}, out)
}

func sortedAccounts(m map[string]record.Record) []record.Record {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]record.Record, 0, len(ids))
	for _, id := range ids {
		out = append(out, m[id])
	}
	return out
}

func sortAllocations(allocs []record.Record) {
	sort.Slice(allocs, func(i, j int) bool {
		a, b := allocs[i], allocs[j]
		if a["account_id"] != b["account_id"] {
			return strID(a["account_id"]) < strID(b["account_id"])
		}
		return strID(a["app"]) < strID(b["app"])
	})
}
