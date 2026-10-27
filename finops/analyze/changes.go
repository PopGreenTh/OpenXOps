package analyze

import (
	"fmt"
	"sort"

	"github.com/PopGreenTh/OpenXOps/record"
)

const cmdChanges = "finops analyze changes"

type allocKey struct {
	account, app string
}

type snap struct {
	accounts    map[string]record.Record
	allocations map[allocKey]record.Record
	costs       []record.Record
}

func buildSnap(recs []record.Record) snap {
	s := snap{
		accounts:    map[string]record.Record{},
		allocations: map[allocKey]record.Record{},
	}
	for _, r := range recs {
		switch r.Kind() {
		case "account":
			s.accounts[strID(r["id"])] = r
		case "allocation":
			k := allocKey{strID(r["account_id"]), strID(r["app"])}
			s.allocations[k] = r
		case "cost":
			s.costs = append(s.costs, r)
		}
	}
	return s
}

// CompareChanges lists differences between before and now map outputs.
func CompareChanges(before, now []record.Record) []record.Record {
	b := buildSnap(before)
	n := buildSnap(now)
	var changes []record.Record

	bAcctIDs := sortedKeys(b.accounts)
	nAcctIDs := sortedKeys(n.accounts)
	bSet, nSet := toSet(bAcctIDs), toSet(nAcctIDs)
	for _, id := range bAcctIDs {
		if !nSet[id] {
			pf, pt := periodRange(b.costs, id)
			appendChanges(&changes, "account", id, "removed", "", "", "", costMoved(b.costs, id), pf, pt, len(b.costs) > 0)
		}
	}
	for _, id := range nAcctIDs {
		if !bSet[id] {
			pf, pt := periodRange(n.costs, id)
			appendChanges(&changes, "account", id, "added", "", "", "", costMoved(n.costs, id), pf, pt, len(n.costs) > 0)
		}
	}
	for _, id := range nAcctIDs {
		if !bSet[id] || !nSet[id] {
			continue
		}
		ba, na := b.accounts[id], n.accounts[id]
		if strID(ba["ou_path"]) != strID(na["ou_path"]) {
			pf, pt := periodRange(n.costs, id)
			appendChanges(&changes, "account", id, "moved", "ou_path", strID(ba["ou_path"]), strID(na["ou_path"]), costMoved(n.costs, id), pf, pt, len(n.costs) > 0)
		}
	}

	bApps := map[string][]string{}
	nApps := map[string][]string{}
	for k := range b.allocations {
		bApps[k.account] = append(bApps[k.account], k.app)
	}
	for k := range n.allocations {
		nApps[k.account] = append(nApps[k.account], k.app)
	}
	for acct, bList := range bApps {
		sort.Strings(bList)
		nList := append([]string{}, nApps[acct]...)
		sort.Strings(nList)
		if len(bList) == 1 && len(nList) == 1 && bList[0] != nList[0] {
			pf, pt := periodRange(n.costs, acct)
			appendChanges(&changes, "account", acct, "remapped", "app", bList[0], nList[0], costMoved(n.costs, acct), pf, pt, len(n.costs) > 0)
			continue
		}
		bA, nA := toSet(bList), toSet(nList)
		for _, app := range bList {
			if !nA[app] {
				subject := acct + "/" + app
				pf, pt := periodRangeForApp(b.costs, acct, app)
				appendChanges(&changes, "allocation", subject, "removed", "", app, "", costMovedForApp(b.costs, acct, app), pf, pt, len(b.costs) > 0)
			}
		}
		for _, app := range nList {
			if !bA[app] {
				subject := acct + "/" + app
				pf, pt := periodRangeForApp(n.costs, acct, app)
				appendChanges(&changes, "allocation", subject, "added", "", "", app, costMovedForApp(n.costs, acct, app), pf, pt, len(n.costs) > 0)
			}
		}
	}

	for _, id := range accountsWithAllocations(b, n) {
		ba, bOK := repAllocation(b, id)
		na, nOK := repAllocation(n, id)
		if !bOK || !nOK {
			continue
		}
		for _, f := range []string{"environment", "lz"} {
			if strID(ba[f]) != strID(na[f]) {
				pf, pt := periodRange(n.costs, id)
				appendChanges(&changes, "account", id, "remapped", f, strID(ba[f]), strID(na[f]), costMoved(n.costs, id), pf, pt, len(n.costs) > 0)
			}
		}
	}

	for k, na := range n.allocations {
		ba, ok := b.allocations[k]
		if !ok {
			continue
		}
		for _, f := range []string{"project", "business_unit", "cost_center"} {
			if strID(ba[f]) != strID(na[f]) {
				subject := k.account + "/" + k.app
				pf, pt := periodRangeForApp(n.costs, k.account, k.app)
				appendChanges(&changes, "allocation", subject, "remapped", f, strID(ba[f]), strID(na[f]), costMovedForApp(n.costs, k.account, k.app), pf, pt, len(n.costs) > 0)
			}
		}
	}

	sort.Slice(changes, func(i, j int) bool {
		a, b := changes[i], changes[j]
		if a["subject_type"] != b["subject_type"] {
			return strID(a["subject_type"]) < strID(b["subject_type"])
		}
		if a["subject"] != b["subject"] {
			return strID(a["subject"]) < strID(b["subject"])
		}
		if a["change"] != b["change"] {
			return strID(a["change"]) < strID(b["change"])
		}
		if a["field"] != b["field"] {
			return strID(a["field"]) < strID(b["field"])
		}
		return strID(a["currency"]) < strID(b["currency"])
	})
	return changes
}

func appendChanges(changes *[]record.Record, subjectType, subject, change, field, old, newVal string, moved map[string]float64, prFrom, prTo string, inputHasCost bool) {
	if len(moved) == 0 {
		*changes = append(*changes, changeRec(subjectType, subject, change, field, old, newVal, "", 0, prFrom, prTo, !inputHasCost))
		return
	}
	for _, cur := range sortedCurrencyKeys(moved) {
		*changes = append(*changes, changeRec(subjectType, subject, change, field, old, newVal, cur, moved[cur], prFrom, prTo, false))
	}
}

func changeRec(subjectType, subject, change, field, old, newVal, currency string, costMovedAmt float64, prFrom, prTo string, nullCost bool) record.Record {
	rec := record.Record{
		"kind":         "change",
		"subject_type": subjectType,
		"subject":      subject,
		"change":       change,
		"field":        field,
		"old":          old,
		"new":          newVal,
		"command":      cmdChanges,
	}
	if nullCost {
		rec["cost_moved"] = nil
		rec["currency"] = ""
		rec["period_from"] = ""
		rec["period_to"] = ""
		return rec
	}
	if currency == "" {
		rec["cost_moved"] = 0.0
		rec["currency"] = ""
		rec["period_from"] = ""
		rec["period_to"] = ""
		return rec
	}
	rec["cost_moved"] = costMovedAmt
	rec["currency"] = currency
	rec["period_from"] = prFrom
	rec["period_to"] = prTo
	return rec
}

func sortedCurrencyKeys(moved map[string]float64) []string {
	out := make([]string, 0, len(moved))
	for k := range moved {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func accountsWithAllocations(b, n snap) []string {
	set := map[string]struct{}{}
	for k := range n.allocations {
		if _, ok := b.allocations[k]; !ok {
			continue
		}
		set[k.account] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func repAllocation(s snap, accountID string) (record.Record, bool) {
	var apps []string
	for k := range s.allocations {
		if k.account == accountID {
			apps = append(apps, k.app)
		}
	}
	if len(apps) == 0 {
		return nil, false
	}
	sort.Strings(apps)
	return s.allocations[allocKey{accountID, apps[0]}], true
}

func costMoved(costs []record.Record, accountID string) map[string]float64 {
	return costMovedForApp(costs, accountID, "")
}

func costMovedForApp(costs []record.Record, accountID, app string) map[string]float64 {
	out := map[string]float64{}
	for _, c := range costs {
		if strID(c["account_id"]) != accountID {
			continue
		}
		if app != "" && strID(c["app"]) != app {
			continue
		}
		cur := strID(c["currency"])
		amt, _ := toFloat(c["amount"])
		out[cur] += amt
	}
	return out
}

func periodRangeForApp(costs []record.Record, accountID, app string) (string, string) {
	var periods []string
	for _, c := range costs {
		if strID(c["account_id"]) != accountID {
			continue
		}
		if app != "" && strID(c["app"]) != app {
			continue
		}
		periods = append(periods, strID(c["period"]))
	}
	if len(periods) == 0 {
		return "", ""
	}
	sort.Strings(periods)
	return periods[0], periods[len(periods)-1]
}

func periodRange(costs []record.Record, accountID string) (string, string) {
	return periodRangeForApp(costs, accountID, "")
}

func sortedKeys(m map[string]record.Record) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func toSet(list []string) map[string]bool {
	s := map[string]bool{}
	for _, v := range list {
		s[v] = true
	}
	return s
}

func strID(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	default:
		return fmt.Sprint(v)
	}
}

func toFloat(v any) (float64, bool) {
	f, ok := v.(float64)
	return f, ok
}
