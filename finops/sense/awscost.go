package sense

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/PopGreenTh/OpenXOps/aws"
	"github.com/PopGreenTh/OpenXOps/aws/ce"
	"github.com/PopGreenTh/OpenXOps/record"
)

// CostOptions configures aws-cost.
type CostOptions struct {
	Mock        bool
	Profile     string
	Region      string
	LZ          string
	Taxonomy    string
	Granularity string
	Metric      string
	From        string
	To          string
	AppTag      string
	ProjectTag  string
	Refresh     bool
	Management  string
	CacheRoot   string
	Out         io.Writer
	CE          ce.API
	Org         aws.OrgAPI
	CallCount   *int
}

// RunAwsCost fetches cost and writes JSON Lines records.
func RunAwsCost(ctx context.Context, opt CostOptions) error {
	now := Clock()
	if opt.Mock {
		now = ce.NowFunc()
		Clock = ce.NowFunc
	}
	ceGranularity, recordGranularity, err := parseGranularity(opt.Granularity)
	if err != nil {
		return err
	}
	metric := opt.Metric
	if metric == "" {
		metric = defaultMetric(recordGranularity)
	}
	from, to := opt.From, opt.To
	if from == "" || to == "" {
		df, dt := defaultRange(recordGranularity, now)
		if from == "" {
			from = df
		}
		if to == "" {
			to = dt
		}
	}
	appTag, projectTag, err := resolveTags(opt)
	if err != nil {
		return err
	}
	mgmt := opt.Management
	if mgmt == "" {
		if opt.Mock {
			snap, err := aws.DemoSnapshot()
			if err != nil {
				return err
			}
			mgmt = snap.ManagementAccount
		} else {
			cfg, err := aws.LoadConfig(ctx, aws.Config{Profile: opt.Profile, Region: opt.Region})
			if err != nil {
				return err
			}
			if opt.Org == nil {
				opt.Org = aws.NewOrgClient(cfg)
			}
			var orgID string
			orgID, mgmt, err = opt.Org.DescribeOrganization(ctx)
			if err != nil {
				return err
			}
			_ = orgID
		}
	}
	if opt.LZ != "" {
		want, err := managementAccountForLZ(opt.Taxonomy, opt.LZ)
		if err != nil {
			return err
		}
		if mgmt != want {
			return fmt.Errorf("organization management account %q does not match LZ %q in taxonomy (want %q)", mgmt, opt.LZ, want)
		}
	}
	opt.Management = mgmt
	periods, err := listPeriods(from, to, recordGranularity)
	if err != nil {
		return err
	}
	var findings []record.Record
	if opt.Mock {
		avail := ce.MockPeriodsAvailable(ceGranularity)
		if len(periods) > avail {
			findings = append(findings, record.Finding("warn", "cost-history-short", periods[0],
				fmt.Sprintf("mock has %d %s periods; range asks for %d", avail, recordGranularity, len(periods)),
				"finops sense aws-cost"))
			periods = periods[len(periods)-avail:]
		}
	}
	root := cacheRoot(opt.Mock)
	if opt.CacheRoot != "" {
		root = opt.CacheRoot
	}
	store := &cacheStore{root: root, managementAccount: mgmt, now: now}
	level1Group := []ce.GroupBy{{Type: "DIMENSION", Key: "LINKED_ACCOUNT"}, {Type: "TAG", Key: projectTag}}
	level2Group := []ce.GroupBy{{Type: "DIMENSION", Key: "SERVICE"}, {Type: "TAG", Key: appTag}}

	var projectRecs, costRecs []record.Record
	accounts, err := fetchLevel1(ctx, opt, store, ceGranularity, recordGranularity, metric, projectTag, level1Group, periods, now, &projectRecs)
	if err != nil {
		return err
	}
	if err := fetchLevel2(ctx, opt, store, ceGranularity, recordGranularity, metric, appTag, level2Group, periods, now, accounts, &costRecs); err != nil {
		return err
	}
	recs := append(projectRecs, costRecs...)
	sortRecords(recs)
	if opt.LZ != "" {
		for i := range recs {
			recs[i]["lz"] = opt.LZ
		}
	}
	out := append(findings, recs...)
	if err := record.Write(opt.Out, out); err != nil {
		return err
	}
	calls := 0
	if opt.CallCount != nil {
		calls = *opt.CallCount
	}
	fmt.Fprintf(os.Stderr, "Cost Explorer: %d calls ($%.2f)\n", calls, float64(calls)*0.01)
	return nil
}

func fetchLevel1(ctx context.Context, opt CostOptions, store *cacheStore, ceGranularity, recordGranularity, metric, projectTag string, group []ce.GroupBy, periods []string, now time.Time, out *[]record.Record) ([]string, error) {
	var needFetch []string
	cached := map[string]*cacheEntry{}
	for _, p := range periods {
		if ent, ok := store.read(1, recordGranularity, metric, projectTag, "", p, opt.Refresh); ok {
			cached[p] = ent
			continue
		}
		needFetch = append(needFetch, p)
	}
	if err := fetchAndCache(ctx, opt, store, 1, ceGranularity, recordGranularity, metric, projectTag, "", group, nil, needFetch, now, cached); err != nil {
		return nil, err
	}
	accountSet := map[string]struct{}{}
	for _, p := range periods {
		ent := cached[p]
		if ent == nil {
			continue
		}
		for _, rt := range ent.Results {
			for _, row := range rt.Rows {
				if row.LinkedAccount != "" {
					accountSet[row.LinkedAccount] = struct{}{}
				}
			}
		}
	}
	accounts := sortedKeys(accountSet)
	for _, p := range periods {
		ent := cached[p]
		if ent == nil {
			continue
		}
		for _, rt := range filterResultsForPeriod(ent.Results, p, ceGranularity) {
			for _, row := range rt.Rows {
				*out = append(*out, projectTagRecord(opt, recordGranularity, metric, p, rt.Estimated, row))
			}
		}
	}
	return accounts, nil
}

func fetchLevel2(ctx context.Context, opt CostOptions, store *cacheStore, ceGranularity, recordGranularity, metric, appTag string, group []ce.GroupBy, periods []string, now time.Time, accounts []string, out *[]record.Record) error {
	sem := make(chan struct{}, 5)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	for _, acct := range accounts {
		acct := acct
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			var needFetch []string
			cached := map[string]*cacheEntry{}
			for _, p := range periods {
				if ent, ok := store.read(2, recordGranularity, metric, appTag, acct, p, opt.Refresh); ok {
					cached[p] = ent
					continue
				}
				needFetch = append(needFetch, p)
			}
			if err := fetchAndCache(ctx, opt, store, 2, ceGranularity, recordGranularity, metric, appTag, acct, group, &ce.Filter{LinkedAccountID: acct}, needFetch, now, cached); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			for _, p := range periods {
				ent := cached[p]
				if ent == nil {
					continue
				}
				var local []record.Record
				for _, rt := range filterResultsForPeriod(ent.Results, p, ceGranularity) {
					for _, row := range rt.Rows {
						if row.LinkedAccount == "" {
							row.LinkedAccount = acct
						}
						local = append(local, costRecord(opt, recordGranularity, metric, p, rt.Estimated, row))
					}
				}
				mu.Lock()
				*out = append(*out, local...)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

func fetchAndCache(ctx context.Context, opt CostOptions, store *cacheStore, level int, ceGranularity, recordGranularity, metric, tagKey, account string, group []ce.GroupBy, filter *ce.Filter, periods []string, now time.Time, cached map[string]*cacheEntry) error {
	if len(periods) == 0 {
		return nil
	}
	ranges := coalesceRanges(periods, recordGranularity)
	for _, rg := range ranges {
		in := ce.GetCostAndUsageInput{
			Start:       rg[0],
			End:         rg[1],
			Granularity: ceGranularity,
			Metric:      metric,
			GroupBy:     group,
			Filter:      filter,
		}
		out, err := opt.CE.GetCostAndUsage(ctx, in)
		if err != nil {
			return err
		}
		if opt.CallCount != nil {
			*opt.CallCount++
		}
		for _, p := range periodsInAPIRange(rg, ceGranularity) {
			if !contains(periods, p) {
				continue
			}
			slice := filterResultsForPeriod(out.Results, p, ceGranularity)
			closed := periodClosed(p, recordGranularity, now)
			if err := store.write(level, recordGranularity, metric, tagKey, account, p, closed, slice); err != nil {
				return err
			}
			cached[p] = &cacheEntry{Results: slice}
		}
	}
	return nil
}

func periodsInAPIRange(rg [2]string, ceGranularity string) []string {
	start, _ := time.Parse("2006-01-02", rg[0])
	end, _ := time.Parse("2006-01-02", rg[1])
	var out []string
	if ceGranularity == "DAILY" {
		for t := start; t.Before(end); t = t.AddDate(0, 0, 1) {
			out = append(out, t.Format("2006-01-02"))
		}
		return out
	}
	for t := start; t.Before(end); t = t.AddDate(0, 1, 0) {
		out = append(out, t.Format("2006-01"))
	}
	return out
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

func projectTagRecord(opt CostOptions, granularity, metric, period string, estimated bool, row ce.UsageRow) record.Record {
	return record.Record{
		"kind":               "project_tag_cost",
		"csp":                "aws",
		"management_account": opt.Management,
		"account_id":         row.LinkedAccount,
		"granularity":        granularity,
		"period":             period,
		"metric":             metric,
		"project_tag":        row.TagValue,
		"amount":             row.Amount,
		"currency":           row.Unit,
		"estimated":          estimated,
	}
}

func costRecord(opt CostOptions, granularity, metric, period string, estimated bool, row ce.UsageRow) record.Record {
	return record.Record{
		"kind":               "cost",
		"csp":                "aws",
		"management_account": opt.Management,
		"account_id":         row.LinkedAccount,
		"granularity":        granularity,
		"period":             period,
		"metric":             metric,
		"service":            row.Service,
		"app_tag":            row.TagValue,
		"amount":             row.Amount,
		"currency":           row.Unit,
		"estimated":          estimated,
	}
}

func sortRecords(recs []record.Record) {
	sort.SliceStable(recs, func(i, j int) bool {
		a, b := recs[i], recs[j]
		ka, kb := a.Kind(), b.Kind()
		if ka != kb {
			return ka == "project_tag_cost"
		}
		for _, key := range []string{"account_id", "period"} {
			sa, _ := a[key].(string)
			sb, _ := b[key].(string)
			if sa != sb {
				return sa < sb
			}
		}
		if ka == "cost" {
			sa, _ := a["service"].(string)
			sb, _ := b["service"].(string)
			if sa != sb {
				return sa < sb
			}
			sa, _ = a["app_tag"].(string)
			sb, _ = b["app_tag"].(string)
			return sa < sb
		}
		sa, _ := a["project_tag"].(string)
		sb, _ := b["project_tag"].(string)
		return sa < sb
	})
}

func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
