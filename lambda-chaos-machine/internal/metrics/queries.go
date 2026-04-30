// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause

// Package metrics converts chaos-machine experiment metric definitions into
// CloudWatch GetMetricData queries. This logic is shared between the
// steady-state and evaluate-hypothesis Lambda functions.
package metrics

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
)

// ToMetricDataQueries converts experiment input metric definitions to CloudWatch
// GetMetricData format. Expression metrics (e* prefix) are verdicts (0=FAIL, 1=PASS);
// raw metrics (m* prefix) are data inputs.
func ToMetricDataQueries(defs []experiment.MetricDef) []cwtypes.MetricDataQuery {
	queries := make([]cwtypes.MetricDataQuery, 0, len(defs))
	for _, m := range defs {
		q := cwtypes.MetricDataQuery{
			Id: aws.String(m.ID),
		}

		if m.Expression != "" {
			label := m.Label
			if label == "" {
				label = m.ID
			}

			q.Expression = aws.String(m.Expression)
			q.Label = aws.String(label)
			q.ReturnData = aws.Bool(true)
		} else if m.MetricStat != nil {
			dims := make([]cwtypes.Dimension, 0, len(m.MetricStat.Metric.Dimensions))
			for _, d := range m.MetricStat.Metric.Dimensions {
				dims = append(dims, cwtypes.Dimension{
					Name:  aws.String(d.Name),
					Value: aws.String(d.Value),
				})
			}

			period := m.MetricStat.Period
			if period == 0 {
				period = 60
			}

			stat := m.MetricStat.Stat
			if stat == "" {
				stat = "Sum"
			}

			q.MetricStat = &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{
					Namespace:  aws.String(m.MetricStat.Metric.Namespace),
					MetricName: aws.String(m.MetricStat.Metric.MetricName),
					Dimensions: dims,
				},
				Period: aws.Int32(period),
				Stat:   aws.String(stat),
			}
			// Only return data for expression metrics (e* prefix)
			q.ReturnData = aws.Bool(strings.HasPrefix(m.ID, "e"))
		}

		queries = append(queries, q)
	}

	return queries
}

// IsExpressionMetric returns true if the metric ID is an expression verdict (e* prefix).
func IsExpressionMetric(id string) bool {
	return strings.HasPrefix(id, "e")
}
