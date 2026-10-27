package sense

import (
	"fmt"
	"strings"
	"time"
)

// Clock returns the current time (tests override).
var Clock = func() time.Time { return time.Now().UTC() }

func defaultMetric(granularity string) string {
	if strings.EqualFold(granularity, "daily") {
		return "AmortizedCost"
	}
	return "UnblendedCost"
}

func parseGranularity(g string) (ceGranularity, recordGranularity string, err error) {
	switch strings.ToLower(g) {
	case "monthly", "":
		return "MONTHLY", "monthly", nil
	case "daily":
		return "DAILY", "daily", nil
	default:
		return "", "", fmt.Errorf("invalid --granularity %q (want monthly|daily)", g)
	}
}

func defaultRange(granularity string, now time.Time) (from, to string) {
	if strings.EqualFold(granularity, "daily") {
		to = now.Format("2006-01-02")
		from = now.AddDate(0, 0, -29).Format("2006-01-02")
		return from, to
	}
	cur := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	fromT := cur.AddDate(0, -24, 0)
	return fromT.Format("2006-01"), cur.Format("2006-01")
}

func listPeriods(from, to, recordGranularity string) ([]string, error) {
	if recordGranularity == "daily" {
		return listDays(from, to)
	}
	return listMonths(from, to)
}

func listMonths(from, to string) ([]string, error) {
	start, err := time.Parse("2006-01", from)
	if err != nil {
		return nil, fmt.Errorf("invalid --from %q (want YYYY-MM)", from)
	}
	end, err := time.Parse("2006-01", to)
	if err != nil {
		return nil, fmt.Errorf("invalid --to %q (want YYYY-MM)", to)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("--to before --from")
	}
	var out []string
	for t := start; !t.After(end); t = t.AddDate(0, 1, 0) {
		out = append(out, t.Format("2006-01"))
	}
	return out, nil
}

func listDays(from, to string) ([]string, error) {
	start, err := time.Parse("2006-01-02", from)
	if err != nil {
		return nil, fmt.Errorf("invalid --from %q (want YYYY-MM-DD)", from)
	}
	end, err := time.Parse("2006-01-02", to)
	if err != nil {
		return nil, fmt.Errorf("invalid --to %q (want YYYY-MM-DD)", to)
	}
	if end.Before(start) {
		return nil, fmt.Errorf("--to before --from")
	}
	var out []string
	for t := start; !t.After(end); t = t.AddDate(0, 0, 1) {
		out = append(out, t.Format("2006-01-02"))
	}
	return out, nil
}

func periodClosed(period, recordGranularity string, now time.Time) bool {
	if recordGranularity == "daily" {
		t, _ := time.Parse("2006-01-02", period)
		cutoff := now.AddDate(0, 0, -2)
		cutoff = time.Date(cutoff.Year(), cutoff.Month(), cutoff.Day(), 0, 0, 0, 0, time.UTC)
		return t.Before(cutoff) || t.Equal(cutoff)
	}
	t, _ := time.Parse("2006-01", period)
	curMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return t.Before(curMonth)
}

func periodAPIBounds(period, ceGranularity string) (start, end string) {
	if ceGranularity == "DAILY" {
		t, _ := time.Parse("2006-01-02", period)
		return period, t.AddDate(0, 0, 1).Format("2006-01-02")
	}
	t, _ := time.Parse("2006-01", period)
	return t.Format("2006-01-02"), t.AddDate(0, 1, 0).Format("2006-01-02")
}

func coalesceRanges(periods []string, recordGranularity string) [][2]string {
	if len(periods) == 0 {
		return nil
	}
	ceG := "MONTHLY"
	if recordGranularity == "daily" {
		ceG = "DAILY"
	}
	var ranges [][2]string
	startP := periods[0]
	prev := periods[0]
	for i := 1; i < len(periods); i++ {
		if adjacent(prev, periods[i], recordGranularity) {
			prev = periods[i]
			continue
		}
		ranges = append(ranges, rangeBounds(startP, prev, ceG))
		startP = periods[i]
		prev = periods[i]
	}
	ranges = append(ranges, rangeBounds(startP, prev, ceG))
	return ranges
}

func adjacent(a, b, recordGranularity string) bool {
	if recordGranularity == "daily" {
		ta, _ := time.Parse("2006-01-02", a)
		tb, _ := time.Parse("2006-01-02", b)
		return tb.Sub(ta) == 24*time.Hour
	}
	ta, _ := time.Parse("2006-01", a)
	tb, _ := time.Parse("2006-01", b)
	return tb.Equal(ta.AddDate(0, 1, 0))
}

func rangeBounds(fromPeriod, toPeriod, ceGranularity string) [2]string {
	start, _ := periodAPIBounds(fromPeriod, ceGranularity)
	_, end := periodAPIBounds(toPeriod, ceGranularity)
	return [2]string{start, end}
}
