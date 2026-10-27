package ce

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

//go:embed testdata/mock-accounts.json
var mockAccountsJSON []byte

const (
	mockMonthlyPeriods = 26
	mockDailyPeriods   = 60
	externalAccountID  = "299999999999"
)

// MockClock is the reference time for mock period generation (tests may override via NowFunc).
var NowFunc = func() time.Time { return time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC) }

type mockAccount struct {
	ID       string   `json:"id"`
	AppTags  []string `json:"app_tags"`
	Projects []string `json:"project_tags"`
}

type mockAccountsFile struct {
	Accounts []mockAccount `json:"accounts"`
}

// Mock implements API with deterministic embedded cost data.
type Mock struct{}

// NewMock returns a mock Cost Explorer client.
func NewMock() *Mock {
	return &Mock{}
}

func (m *Mock) GetCostAndUsage(ctx context.Context, in GetCostAndUsageInput) (GetCostAndUsageOutput, error) {
	accounts, err := loadMockAccounts()
	if err != nil {
		return GetCostAndUsageOutput{}, err
	}
	if in.Filter != nil && in.Filter.LinkedAccountID != "" {
		var filtered []mockAccount
		for _, a := range accounts {
			if a.ID == in.Filter.LinkedAccountID {
				filtered = append(filtered, a)
			}
		}
		accounts = filtered
	}
	periods, err := periodsInRange(in.Start, in.End, in.Granularity)
	if err != nil {
		return GetCostAndUsageOutput{}, err
	}
	var results []ResultByTime
	for _, p := range periods {
		start, end := periodBounds(p, in.Granularity)
		res := ResultByTime{Start: start, End: end, Estimated: isOpenPeriod(p, in.Granularity, NowFunc())}
		var rows []UsageRow
		for _, ac := range accounts {
			rows = append(rows, mockRowsForAccount(ac, in, p)...)
		}
		res.Rows = rows
		results = append(results, res)
	}
	return GetCostAndUsageOutput{Results: results}, nil
}

func loadMockAccounts() ([]mockAccount, error) {
	var file mockAccountsFile
	if err := json.Unmarshal(mockAccountsJSON, &file); err != nil {
		return nil, fmt.Errorf("mock accounts: %w", err)
	}
	return file.Accounts, nil
}

func mockRowsForAccount(ac mockAccount, in GetCostAndUsageInput, period string) []UsageRow {
	var tagKey string
	var isProject bool
	for _, g := range in.GroupBy {
		if g.Type == "TAG" {
			tagKey = g.Key
			isProject = strings.EqualFold(g.Key, defaultProjectTag(in))
		}
	}
	if tagKey == "" {
		return nil
	}
	if isProject {
		return mockProjectRows(ac, in.Metric, period, tagKey)
	}
	return mockAppRows(ac, in.Metric, period, tagKey)
}

func defaultProjectTag(in GetCostAndUsageInput) string {
	for _, g := range in.GroupBy {
		if g.Type == "TAG" && g.Key != "App" {
			return g.Key
		}
	}
	return "Project"
}

func mockAppRows(ac mockAccount, metric, period, tagKey string) []UsageRow {
	var rows []UsageRow
	for _, svc := range mockServices {
		for _, app := range ac.AppTags {
			amt := mockAmount(ac.ID, svc, app, period, metric)
			rows = append(rows, UsageRow{
				LinkedAccount: ac.ID,
				Service:       svc,
				TagKey:        tagKey,
				TagValue:      app,
				Amount:        amt,
				Unit:          "USD",
			})
		}
	}
	return rows
}

func mockProjectRows(ac mockAccount, metric, period, tagKey string) []UsageRow {
	var total float64
	for _, svc := range mockServices {
		for _, app := range ac.AppTags {
			total += mockAmount(ac.ID, svc, app, period, metric)
		}
	}
	n := len(ac.Projects)
	if n == 0 {
		return nil
	}
	share := round2(total / float64(n))
	var rows []UsageRow
	for i, proj := range ac.Projects {
		amt := share
		if i == n-1 {
			amt = round2(total - share*float64(n-1))
		}
		rows = append(rows, UsageRow{
			LinkedAccount: ac.ID,
			TagKey:        tagKey,
			TagValue:      proj,
			Amount:        amt,
			Unit:          "USD",
		})
	}
	return rows
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// MockPeriodsAvailable returns how many periods the mock has for granularity.
func MockPeriodsAvailable(granularity string) int {
	if strings.EqualFold(granularity, "DAILY") {
		return mockDailyPeriods
	}
	return mockMonthlyPeriods
}

func periodsInRange(start, end, granularity string) ([]string, error) {
	all := mockAllPeriods(granularity, NowFunc())
	var out []string
	for _, p := range all {
		bStart, bEnd := periodBounds(p, granularity)
		if bStart >= start && bEnd <= end {
			out = append(out, p)
		} else if bStart < end && bEnd > start {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no mock periods in range %s..%s", start, end)
	}
	return out, nil
}

func mockAllPeriods(granularity string, now time.Time) []string {
	if strings.EqualFold(granularity, "DAILY") {
		var days []string
		for i := mockDailyPeriods - 1; i >= 0; i-- {
			d := now.AddDate(0, 0, -i)
			days = append(days, d.Format("2006-01-02"))
		}
		return days
	}
	cur := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	var months []string
	for i := mockMonthlyPeriods - 1; i >= 0; i-- {
		m := cur.AddDate(0, -i, 0)
		months = append(months, m.Format("2006-01"))
	}
	return months
}

func periodBounds(period, granularity string) (start, end string) {
	if strings.EqualFold(granularity, "DAILY") {
		t, _ := time.Parse("2006-01-02", period)
		return period, t.AddDate(0, 0, 1).Format("2006-01-02")
	}
	t, _ := time.Parse("2006-01", period)
	return t.Format("2006-01-02"), t.AddDate(0, 1, 0).Format("2006-01-02")
}

func isOpenPeriod(period, granularity string, now time.Time) bool {
	if strings.EqualFold(granularity, "DAILY") {
		t, _ := time.Parse("2006-01-02", period)
		yesterday := now.AddDate(0, 0, -1)
		return !t.Before(time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC))
	}
	t, _ := time.Parse("2006-01", period)
	curMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return !t.Before(curMonth)
}
