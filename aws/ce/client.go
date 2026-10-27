package ce

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/PopGreenTh/OpenXOps/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

type getCostAndUsageAPI interface {
	GetCostAndUsage(context.Context, *costexplorer.GetCostAndUsageInput, ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
}

// Client calls AWS Cost Explorer through the shared retry wrapper.
type Client struct {
	svc getCostAndUsageAPI
}

// NewClient returns a Cost Explorer client for cfg.
func NewClient(cfg awscfg.Config) *Client {
	return &Client{svc: costexplorer.NewFromConfig(cfg)}
}

// GetCostAndUsage fetches one page for in (CE returns all groups in one response for typical queries).
func (c *Client) GetCostAndUsage(ctx context.Context, in GetCostAndUsageInput) (GetCostAndUsageOutput, error) {
	var out GetCostAndUsageOutput
	slog.Debug("cost explorer call", slog.String("start", in.Start), slog.String("end", in.End), slog.String("granularity", in.Granularity))
	err := aws.Retry(ctx, func() error {
		page, err := c.svc.GetCostAndUsage(ctx, buildInput(in))
		if err != nil {
			return err
		}
		out.Results = flattenOutput(page, in)
		return nil
	})
	if err != nil {
		return GetCostAndUsageOutput{}, err
	}
	return out, nil
}

func buildInput(in GetCostAndUsageInput) *costexplorer.GetCostAndUsageInput {
	var groupBy []types.GroupDefinition
	for _, g := range in.GroupBy {
		groupBy = append(groupBy, types.GroupDefinition{
			Type: types.GroupDefinitionType(g.Type),
			Key:  awscfg.String(g.Key),
		})
	}
	input := &costexplorer.GetCostAndUsageInput{
		TimePeriod: &types.DateInterval{
			Start: awscfg.String(in.Start),
			End:   awscfg.String(in.End),
		},
		Granularity: types.Granularity(in.Granularity),
		Metrics:     []string{in.Metric},
		GroupBy:     groupBy,
	}
	if in.Filter != nil && in.Filter.LinkedAccountID != "" {
		input.Filter = &types.Expression{
			Dimensions: &types.DimensionValues{
				Key:    types.DimensionLinkedAccount,
				Values: []string{in.Filter.LinkedAccountID},
			},
		}
	}
	return input
}

func flattenOutput(page *costexplorer.GetCostAndUsageOutput, in GetCostAndUsageInput) []ResultByTime {
	var results []ResultByTime
	for _, rt := range page.ResultsByTime {
		res := ResultByTime{
			Start: awscfg.ToString(rt.TimePeriod.Start),
			End:   awscfg.ToString(rt.TimePeriod.End),
		}
		res.Estimated = rt.Estimated
		for _, g := range rt.Groups {
			amt, unit := metricAmount(g.Metrics, in.Metric)
			row := UsageRow{Amount: amt, Unit: unit}
			for i, gb := range in.GroupBy {
				if i >= len(g.Keys) {
					break
				}
				val := g.Keys[i]
				switch gb.Key {
				case "LINKED_ACCOUNT":
					row.LinkedAccount = val
				case "SERVICE":
					row.Service = val
				default:
					if gb.Type == "TAG" {
						row.TagKey = gb.Key
						row.TagValue = ParseTagGroup(gb.Key, val)
					}
				}
			}
			res.Rows = append(res.Rows, row)
		}
		results = append(results, res)
	}
	return results
}

func metricAmount(metrics map[string]types.MetricValue, metric string) (float64, string) {
	mv, ok := metrics[metric]
	if !ok || mv.Amount == nil {
		return 0, "USD"
	}
	unit := "USD"
	if mv.Unit != nil {
		unit = *mv.Unit
	}
	f, err := strconv.ParseFloat(*mv.Amount, 64)
	if err != nil {
		return 0, unit
	}
	return f, unit
}
