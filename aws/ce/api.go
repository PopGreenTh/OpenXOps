package ce

import "context"

// GroupBy is one Cost Explorer group-by dimension or tag key.
type GroupBy struct {
	Type string // DIMENSION or TAG
	Key  string
}

// Filter matches one linked account when set.
type Filter struct {
	LinkedAccountID string
}

// GetCostAndUsageInput is a single Cost Explorer GetCostAndUsage request (one time span).
type GetCostAndUsageInput struct {
	Start, End  string // inclusive start, exclusive end (YYYY-MM-DD)
	Granularity string // MONTHLY or DAILY
	Metric      string
	GroupBy     []GroupBy
	Filter      *Filter
}

// UsageRow is one grouped cost amount from Cost Explorer.
type UsageRow struct {
	LinkedAccount string
	Service       string
	TagKey        string
	TagValue      string // empty means untagged (key$)
	Amount        float64
	Unit          string
}

// ResultByTime is cost for one time bucket in the response.
type ResultByTime struct {
	Start, End string
	Estimated  bool
	Rows       []UsageRow
}

// GetCostAndUsageOutput is the flattened CE response for one request.
type GetCostAndUsageOutput struct {
	Results []ResultByTime
}

// API calls Cost Explorer GetCostAndUsage (all pages) with retries applied by the client.
type API interface {
	GetCostAndUsage(ctx context.Context, in GetCostAndUsageInput) (GetCostAndUsageOutput, error)
}
