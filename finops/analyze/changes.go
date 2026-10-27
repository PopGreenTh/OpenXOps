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
			appendChanges(&changes, "account", id, "removed", "", "", "", costMoved(b.costs, id), pf, pt)
		}
	}
	for _, id := range nAcctIDs {
		if !bSet[id] {
			pf, pt := periodRange(n.costs, id)
			appendChanges(&changes, "account", id, "added", "", "", "", costMoved(n.costs, id), pf, pt)
		}
	}
	for _, id := range nAcctIDs {
		if !bSet[id] || !nSet[id] {
			continue
		}
		ba, na := b.accounts[id], n.accounts[id]
		if strID(ba["ou_path"]) != strID(na["ou_path"]) {
			pf, pt := periodRange(n.costs, id)
			appendChanges(&changes, "account", id, "moved", "ou_path", strID(ba["ou_path"]), strID(na["ou_path"]), costMoved(n.costs, id), pf, pt)
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
			appendChanges(&changes, "account", acct, "remapped", "app", bList[0], nList[0], costMoved(n.costs, acct), pf, pt)
			continue
		}
		bA, nA := toSet(bList), toSet(nList)
		for _, app := range bList {
			if !nA[app] {
				subject := acct + "/" + app
				pf, pt := periodRange(b.costs, acct)
				appendChanges(&changes, "allocation", subject, "removed", "", app, "", costMoved(b.costs, acct), pf, pt)
			}
		}
		for _, app := range nList {
			if !bA[app] {
				subject := acct + "/" + app
				pf, pt := periodRange(n.costs, acct)
				appendChanges(&changes, "allocation", subject, "added", "", "", app, costMoved(n.costs, acct), pf, pt)
			}
		}
	}

	remapFields := []string{"app", "environment", "project", "business_unit", "cost_center", "lz"}
	for k, na := range n.allocations {
		ba, ok := b.allocations[k]
		if !ok {
			continue
		}
		for _, f := range remapFields {
			if strID(ba[f]) != strID(na[f]) {
				subject := k.account + "/" + k.app
				pf, pt := periodRange(n.costs, k.account)
				appendChanges(&changes, "allocation", subject, "remapped", f, strID(ba[f]), strID(na[f]), costMoved(n.costs, k.account), pf, pt)
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

func appendChanges(changes *[]record.Record, subjectType, subject, change, field, old, newVal string, moved map[string]float64, prFrom, prTo string) {
	if len(moved) == 0 {
		*changes = append(*changes, changeRec(subjectType, subject, change, field, old, newVal, "", 0, prFrom, prTo))
		return
	}
	for _, cur := range sortedCurrencyKeys(moved) {
		*changes = append(*changes, changeRec(subjectType, subject, change, field, old, newVal, cur, moved[cur], prFrom, prTo))
	}
}

func changeRec(subjectType, subject, change, field, old, newVal, currency string, costMovedAmt float64, prFrom, prTo string) record.Record {
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
	if currency == "" {
		rec["cost_moved"] = nil
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

func costMoved(costs []record.Record, accountID string) map[string]float64 {
	out := map[string]float64{}
	for _, c := range costs {
		if strID(c["account_id"]) != accountID {
			continue
		}
		cur := strID(c["currency"])
		amt, _ := toFloat(c["amount"])
		out[cur] += amt
	}
	return out
}

func periodRange(costs []record.Record, accountID string) (string, string) {
	var periods []string
	for _, c := range costs {
		if strID(c["account_id"]) == accountID {
			periods = append(periods, strID(c["period"]))
		}
	}
	if len(periods) == 0 {
		return "", ""
	}
	sort.Strings(periods)
	return periods[0], periods[len(periods)-1]
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
