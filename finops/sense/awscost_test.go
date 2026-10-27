package sense

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PopGreenTh/OpenXOps/aws/ce"
	"github.com/PopGreenTh/OpenXOps/record"
)

func testNow() time.Time {
	return time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
}

func setupTestClock(t *testing.T) func() {
	t.Helper()
	old := Clock
	ce.NowFunc = testNow
	Clock = testNow
	return func() {
		Clock = old
		ce.NowFunc = func() time.Time { return time.Now().UTC() }
	}
}

func testCacheDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return filepath.Join(dir, "cache")
}

func runCost(t *testing.T, opt CostOptions) []record.Record {
	t.Helper()
	var buf bytes.Buffer
	opt.Out = &buf
	if err := RunAwsCost(context.Background(), opt); err != nil {
		t.Fatal(err)
	}
	recs, err := record.Read(bytes.NewReader(buf.Bytes()), "test")
	if err != nil {
		t.Fatal(err)
	}
	return recs
}

func TestMockMonthlyPeriodCount(t *testing.T) {
	defer setupTestClock(t)()
	recs := runCost(t, CostOptions{Mock: true, CE: ce.NewMock(), CacheRoot: testCacheDir(t)})
	periods := map[string]struct{}{}
	for _, r := range recs {
		if r.Kind() == "cost" {
			p, _ := r["period"].(string)
			periods[p] = struct{}{}
		}
	}
	if len(periods) != 25 {
		t.Fatalf("want 25 monthly periods, got %d", len(periods))
	}
}

func TestMockDailyMetric(t *testing.T) {
	defer setupTestClock(t)()
	recs := runCost(t, CostOptions{Mock: true, Granularity: "daily", CE: ce.NewMock(), CacheRoot: testCacheDir(t)})
	periods := map[string]struct{}{}
	for _, r := range recs {
		if r.Kind() != "cost" {
			continue
		}
		if m, _ := r["metric"].(string); m != "AmortizedCost" {
			t.Fatalf("daily default metric = %q", m)
		}
		p, _ := r["period"].(string)
		periods[p] = struct{}{}
	}
	if len(periods) != 30 {
		t.Fatalf("want 30 daily periods, got %d", len(periods))
	}
}

func TestCostTotalsMatchProjectTotals(t *testing.T) {
	defer setupTestClock(t)()
	recs := runCost(t, CostOptions{Mock: true, CE: ce.NewMock(), CacheRoot: testCacheDir(t)})
	sum := map[string]float64{}
	for _, r := range recs {
		p, _ := r["period"].(string)
		m, _ := r["metric"].(string)
		cur, _ := r["currency"].(string)
		key := p + "|" + m + "|" + cur
		amt, _ := r["amount"].(float64)
		if r.Kind() == "cost" {
			sum[key+"|cost"] += amt
		} else if r.Kind() == "project_tag_cost" {
			sum[key+"|proj"] += amt
		}
	}
	for k, cv := range sum {
		if len(k) < 6 || k[len(k)-4:] != "cost" {
			continue
		}
		base := k[:len(k)-4]
		pv := sum[base+"proj"]
		if diff := cv - pv; diff > 0.02 || diff < -0.02 {
			t.Fatalf("period metric %s: cost=%f project=%f", base, cv, pv)
		}
	}
}

func TestCacheSecondRunZeroCalls(t *testing.T) {
	defer setupTestClock(t)()
	cache := testCacheDir(t)
	from, to := "2024-11", "2026-09"
	var calls int
	opt := CostOptions{
		Mock: true, CE: ce.NewMock(), CallCount: &calls,
		From: from, To: to, CacheRoot: cache,
	}
	_ = runCost(t, opt)
	first := calls
	if first == 0 {
		t.Fatal("expected CE calls on first run")
	}
	calls = 0
	_ = runCost(t, opt)
	if calls != 0 {
		t.Fatalf("second run over closed periods: got %d calls, want 0", calls)
	}
}

func TestOpenPeriodCacheReuse(t *testing.T) {
	defer setupTestClock(t)()
	cache := testCacheDir(t)
	now := testNow()
	Clock = func() time.Time { return now }
	from, to := now.Format("2006-01"), now.Format("2006-01")
	var calls int
	opt := CostOptions{
		Mock: true, CE: ce.NewMock(), CallCount: &calls,
		From: from, To: to, CacheRoot: cache,
	}
	_ = runCost(t, opt)
	first := calls
	calls = 0
	_ = runCost(t, opt)
	if calls != 0 {
		t.Fatalf("expected cache reuse for open month, got %d calls", calls)
	}
	_ = first
}

func TestRefreshDoesNotOverwriteClosed(t *testing.T) {
	defer setupTestClock(t)()
	cache := testCacheDir(t)
	from, to := "2024-11", "2024-11"
	opt := CostOptions{Mock: true, CE: ce.NewMock(), From: from, To: to, CacheRoot: cache}
	_ = runCost(t, opt)
	closed := filepath.Join(cache, "111111111111", "1-monthly-UnblendedCost-Project", "2024-11.json")
	info, err := os.Stat(closed)
	if err != nil {
		t.Fatal(err)
	}
	mod := info.ModTime()
	opt.Refresh = true
	_ = runCost(t, opt)
	info2, err := os.Stat(closed)
	if err != nil {
		t.Fatal(err)
	}
	if !info2.ModTime().Equal(mod) {
		t.Fatal("refresh overwrote closed cache file")
	}
	atFiles, _ := filepath.Glob(filepath.Join(cache, "111111111111", "1-monthly-UnblendedCost-Project", "2024-11@*.json"))
	if len(atFiles) == 0 {
		t.Fatal("refresh should write a timestamped cache file for a closed period")
	}
}

func TestDifferentMetricSeparateCache(t *testing.T) {
	defer setupTestClock(t)()
	cache := testCacheDir(t)
	from, to := "2024-11", "2024-11"
	var calls int
	opt := CostOptions{
		Mock: true, CE: ce.NewMock(), CallCount: &calls,
		From: from, To: to, CacheRoot: cache, Metric: "UnblendedCost",
	}
	_ = runCost(t, opt)
	calls = 0
	opt.Metric = "AmortizedCost"
	_ = runCost(t, opt)
	if calls == 0 {
		t.Fatal("different metric should miss cache")
	}
}

func TestLevel2OneCallPerAccount(t *testing.T) {
	defer setupTestClock(t)()
	from, to := "2024-11", "2024-11"
	var calls int
	_ = runCost(t, CostOptions{
		Mock: true, CE: ce.NewMock(), CallCount: &calls,
		From: from, To: to, CacheRoot: testCacheDir(t),
	})
	want := 1 + 9 // one level-1 range call plus one level-2 call per account from level 1
	if calls != want {
		t.Fatalf("expected %d CE calls (1 level-1 + 9 level-2), got %d", want, calls)
	}
}

func TestRefreshClosedStillUsedAfter13Hours(t *testing.T) {
	defer setupTestClock(t)()
	cache := testCacheDir(t)
	from, to := "2024-11", "2024-11"
	opt := CostOptions{Mock: true, CE: ce.NewMock(), From: from, To: to, CacheRoot: cache}
	_ = runCost(t, opt)
	opt.Refresh = true
	_ = runCost(t, opt)
	later := testNow().Add(13 * time.Hour)
	Clock = func() time.Time { return later }
	ce.NowFunc = Clock
	var calls int
	recs := runCost(t, CostOptions{
		Mock: true, CE: ce.NewMock(), CallCount: &calls,
		From: from, To: to, CacheRoot: cache,
	})
	if calls != 0 {
		t.Fatalf("expected cached closed refresh after 13h, got %d CE calls", calls)
	}
	if len(recs) == 0 {
		t.Fatal("expected cost output from refreshed closed cache")
	}
}

func TestCostHistoryShortLiveLeadingEmpty(t *testing.T) {
	defer setupTestClock(t)()
	inner := ce.NewMock()
	stub := &emptyLeadingCE{inner: inner, emptyPeriod: "2024-11"}
	cache := testCacheDir(t)
	recs := runCost(t, CostOptions{
		Mock: false, CE: stub,
		From: "2024-11", To: "2024-12", CacheRoot: cache,
		Management: "111111111111",
	})
	var finding record.Record
	for _, r := range recs {
		if r.Kind() == "finding" && r["code"] == "cost-history-short" {
			finding = r
			if r["severity"] != "warning" {
				t.Fatalf("severity = %v", r["severity"])
			}
		}
	}
	if finding == nil {
		t.Fatal("expected cost-history-short on live leading empty periods")
	}
	if subj, _ := finding["subject"].(string); subj != "2024-12" {
		t.Fatalf("subject = %q, want first period returned 2024-12", subj)
	}
}

type emptyLeadingCE struct {
	inner       ce.API
	emptyPeriod string
}

func (e *emptyLeadingCE) GetCostAndUsage(ctx context.Context, in ce.GetCostAndUsageInput) (ce.GetCostAndUsageOutput, error) {
	out, err := e.inner.GetCostAndUsage(ctx, in)
	if err != nil {
		return out, err
	}
	for i := range out.Results {
		if strings.HasPrefix(out.Results[i].Start, e.emptyPeriod) {
			out.Results[i].Rows = nil
		}
	}
	return out, nil
}

func TestCostHistoryShortFinding(t *testing.T) {
	defer setupTestClock(t)()
	recs := runCost(t, CostOptions{
		Mock: true, CE: ce.NewMock(),
		From: "2020-01", To: "2026-10",
		CacheRoot: testCacheDir(t),
	})
	var finding record.Record
	var firstCostPeriod string
	for _, r := range recs {
		if r.Kind() == "cost" && firstCostPeriod == "" {
			firstCostPeriod, _ = r["period"].(string)
		}
		if r.Kind() == "finding" && r["code"] == "cost-history-short" {
			finding = r
		}
	}
	if finding == nil {
		t.Fatal("expected cost-history-short finding")
	}
	subj, _ := finding["subject"].(string)
	if subj != firstCostPeriod {
		t.Fatalf("subject = %q, want first period returned %q", subj, firstCostPeriod)
	}
}
