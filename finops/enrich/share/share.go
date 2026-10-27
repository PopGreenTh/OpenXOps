package share

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/PopGreenTh/OpenXOps/record"
)

const (
	cmdShare      = "finops enrich share"
	appInvalid    = "APP-INVALID"
	prjUnalloc    = "PRJ-UNALLOC"
	buUnalloc     = "BU-UNALLOC"
	ccUnalloc     = "CC-UNALLOC"
	conservationE = 1e-6
)

// Options holds taxonomy fields the split needs without importing finops/enrich.
type Options struct {
	AppTagKey string
	AppCodes  []string
}

// Split shared-account cost rows by App tag; other records pass through unchanged.
func Split(recs []record.Record, opt Options) ([]record.Record, error) {
	allocs := map[string][]record.Record{}
	var costs []record.Record
	for _, r := range recs {
		switch r.Kind() {
		case "allocation":
			id := strID(r["account_id"])
			allocs[id] = append(allocs[id], r)
		case "cost":
			costs = append(costs, r)
		}
	}
	appCodes := map[string]struct{}{}
	for _, c := range opt.AppCodes {
		appCodes[c] = struct{}{}
	}

	var outCosts []record.Record
	var findings []record.Record
	sharedSawTag := false
	anySharedCost := false
	for _, c := range costs {
		shared, _ := c["shared"].(bool)
		if !shared {
			out := copyRecord(c)
			out["share"] = 1.0
			out["share_rule"] = "single"
			out["source_amount"] = c["amount"]
			outCosts = append(outCosts, out)
			continue
		}
		anySharedCost = true
		tag := strings.TrimSpace(strID(c["app_tag"]))
		if tag != "" {
			sharedSawTag = true
		}
		split, f, err := splitSharedCost(c, allocs[strID(c["account_id"])], appCodes, tag)
		if err != nil {
			return nil, err
		}
		outCosts = append(outCosts, split...)
		findings = append(findings, f...)
	}
	if anySharedCost && !sharedSawTag {
		key := opt.AppTagKey
		if key == "" {
			key = "App"
		}
		findings = append(findings, record.Finding("warning", "app-tag-not-seen", key,
			"no App tag seen on shared cost rows", cmdShare))
	}
	if err := CheckConservation(costs, outCosts); err != nil {
		return nil, err
	}
	sortShareCosts(outCosts)
	sortFindings(findings)

	var own []record.Record
	own = append(own, outCosts...)
	own = append(own, findings...)
	return record.PassThrough(recs, []string{"cost"}, own), nil
}

// CheckConservation verifies output cost totals match input for shared splits.
func CheckConservation(in, out []record.Record) error {
	type key struct {
		acct, period, metric, currency string
	}
	sumIn := map[key]float64{}
	sumOut := map[key]float64{}
	for _, r := range in {
		k := key{strID(r["account_id"]), strID(r["period"]), strID(r["metric"]), strID(r["currency"])}
		v, _ := toFloat(r["amount"])
		sumIn[k] += v
	}
	for _, r := range out {
		k := key{strID(r["account_id"]), strID(r["period"]), strID(r["metric"]), strID(r["currency"])}
		v, _ := toFloat(r["amount"])
		sumOut[k] += v
	}
	for k, vin := range sumIn {
		vout := sumOut[k]
		if math.Abs(vin-vout) > conservationE {
			return fmt.Errorf("conservation failed for account %s period %s: in=%v out=%v", k.acct, k.period, vin, vout)
		}
	}
	return nil
}

func splitSharedCost(cost record.Record, allocs []record.Record, appCodes map[string]struct{}, tag string) ([]record.Record, []record.Record, error) {
	acctID := strID(cost["account_id"])
	period := strID(cost["period"])
	amount, _ := toFloat(cost["amount"])
	served := accountApps(allocs)
	parts := parseTagParts(tag)

	var findings []record.Record
	type partTarget struct {
		part  string
		app   string
		rule  string
		share float64
	}
	var targets []partTarget

	if len(parts) == 0 {
		n := float64(len(served))
		if n == 0 {
			return nil, nil, fmt.Errorf("account %s: shared cost with no allocations", acctID)
		}
		findings = append(findings, record.Finding("warning", "shared-untagged", acctID+"/"+period,
			"untagged shared cost split equally", cmdShare))
		apps := append([]string{}, served...)
		sort.Strings(apps)
		for _, app := range apps {
			targets = append(targets, partTarget{part: "", app: app, rule: "untagged-equal", share: 1 / n})
		}
	} else {
		k := float64(len(parts))
		rule := "tag"
		if len(parts) > 1 {
			rule = "tag-multi"
		}
		for _, part := range parts {
			app, f := matchTagPart(part, served, appCodes, acctID)
			findings = append(findings, f...)
			targets = append(targets, partTarget{part: part, app: app, rule: rule, share: 1 / k})
		}
	}

	sort.SliceStable(targets, func(i, j int) bool {
		return targets[i].app < targets[j].app
	})

	var out []record.Record
	remaining := amount
	for i, tg := range targets {
		shareAmt := amount * tg.share
		if i == len(targets)-1 {
			shareAmt = remaining
		} else {
			remaining -= shareAmt
		}
		rec := copyRecord(cost)
		rec["amount"] = shareAmt
		rec["source_amount"] = amount
		rec["share"] = tg.share
		rec["share_rule"] = tg.rule
		fillFromAllocation(rec, tg.app, allocs)
		out = append(out, rec)
	}
	return out, findings, nil
}

func parseTagParts(tag string) []string {
	tag = strings.ReplaceAll(tag, "__", ",")
	var parts []string
	for _, p := range strings.Split(tag, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}

func matchTagPart(part string, served []string, appCodes map[string]struct{}, acctID string) (string, []record.Record) {
	var findings []record.Record
	for _, s := range served {
		if strings.EqualFold(s, part) {
			return s, findings
		}
	}
	for code := range appCodes {
		if strings.EqualFold(code, part) {
			findings = append(findings, record.Finding("warning", "shared-tag-wrong-app", acctID+"/"+part,
				fmt.Sprintf("tag %q names app account does not serve", part), cmdShare))
			return appInvalid, findings
		}
	}
	findings = append(findings, record.Finding("warning", "shared-tag-unknown", acctID+"/"+part,
		fmt.Sprintf("tag %q matches no app", part), cmdShare))
	return appInvalid, findings
}

func accountApps(allocs []record.Record) []string {
	var apps []string
	for _, a := range allocs {
		apps = append(apps, strID(a["app"]))
	}
	sort.Strings(apps)
	return apps
}

func fillFromAllocation(cost record.Record, app string, allocs []record.Record) {
	if app == appInvalid {
		cost["app"] = appInvalid
		cost["project"] = prjUnalloc
		cost["business_unit"] = buUnalloc
		cost["cost_center"] = ccUnalloc
		cost["app_rule"] = "invalid"
		if len(allocs) > 0 {
			cost["environment"] = allocs[0]["environment"]
		}
		return
	}
	for _, a := range allocs {
		if strID(a["app"]) == app {
			cost["app"] = a["app"]
			cost["project"] = a["project"]
			cost["business_unit"] = a["business_unit"]
			cost["cost_center"] = a["cost_center"]
			cost["app_rule"] = a["app_rule"]
			cost["environment"] = a["environment"]
			return
		}
	}
}

func sortShareCosts(costs []record.Record) {
	sort.SliceStable(costs, func(i, j int) bool {
		a, b := costs[i], costs[j]
		if a["account_id"] != b["account_id"] {
			return strID(a["account_id"]) < strID(b["account_id"])
		}
		if a["period"] != b["period"] {
			return strID(a["period"]) < strID(b["period"])
		}
		if a["app"] != b["app"] {
			return strID(a["app"]) < strID(b["app"])
		}
		af, _ := toFloat(a["amount"])
		bf, _ := toFloat(b["amount"])
		if af != bf {
			return af < bf
		}
		if strID(a["service"]) != strID(b["service"]) {
			return strID(a["service"]) < strID(b["service"])
		}
		return strID(a["app_tag"]) < strID(b["app_tag"])
	})
}

func sortFindings(findings []record.Record) {
	sort.Slice(findings, func(i, j int) bool {
		a, b := findings[i], findings[j]
		if a["code"] != b["code"] {
			return strID(a["code"]) < strID(b["code"])
		}
		return strID(a["subject"]) < strID(b["subject"])
	})
}

func copyRecord(r record.Record) record.Record {
	out := record.Record{}
	for k, v := range r {
		out[k] = v
	}
	return out
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
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}
