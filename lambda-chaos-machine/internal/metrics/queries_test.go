// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause

package metrics_test

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/metrics"
)

func TestToMetricDataQueries_ExpressionOnly(t *testing.T) {
	defs := []experiment.MetricDef{
		{
			ID:         "e1",
			Expression: "IF(m1 > 5, 0, 1)",
			Label:      "ErrorRateOk",
		},
	}

	queries := metrics.ToMetricDataQueries(defs)
	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}

	q := queries[0]
	if aws.ToString(q.Id) != "e1" {
		t.Errorf("expected Id=e1, got %s", aws.ToString(q.Id))
	}
	if aws.ToString(q.Expression) != "IF(m1 > 5, 0, 1)" {
		t.Errorf("wrong expression: %s", aws.ToString(q.Expression))
	}
	if aws.ToString(q.Label) != "ErrorRateOk" {
		t.Errorf("wrong label: %s", aws.ToString(q.Label))
	}
	if !aws.ToBool(q.ReturnData) {
		t.Error("expression metric should have ReturnData=true")
	}
	if q.MetricStat != nil {
		t.Error("expression metric should not have MetricStat")
	}
}

func TestToMetricDataQueries_MetricStat(t *testing.T) {
	defs := []experiment.MetricDef{
		{
			ID: "m1",
			MetricStat: &experiment.MetricStat{
				Metric: experiment.MetricRef{
					Namespace:  "Gideon/Chat",
					MetricName: "ChatRequestError",
					Dimensions: []experiment.Dimension{
						{Name: "Environment", Value: "dev-aws"},
					},
				},
				Period: 60,
				Stat:   "Sum",
			},
		},
	}

	queries := metrics.ToMetricDataQueries(defs)
	if len(queries) != 1 {
		t.Fatalf("expected 1 query, got %d", len(queries))
	}

	q := queries[0]
	if aws.ToString(q.Id) != "m1" {
		t.Errorf("expected Id=m1, got %s", aws.ToString(q.Id))
	}
	if q.Expression != nil {
		t.Error("metric stat should not have Expression")
	}
	if q.MetricStat == nil {
		t.Fatal("metric stat should not be nil")
	}
	if aws.ToString(q.MetricStat.Metric.Namespace) != "Gideon/Chat" {
		t.Errorf("wrong namespace: %s", aws.ToString(q.MetricStat.Metric.Namespace))
	}
	if aws.ToString(q.MetricStat.Metric.MetricName) != "ChatRequestError" {
		t.Errorf("wrong metric name: %s", aws.ToString(q.MetricStat.Metric.MetricName))
	}
	if len(q.MetricStat.Metric.Dimensions) != 1 {
		t.Fatalf("expected 1 dimension, got %d", len(q.MetricStat.Metric.Dimensions))
	}
	if aws.ToString(q.MetricStat.Metric.Dimensions[0].Name) != "Environment" {
		t.Errorf("wrong dimension name: %s", aws.ToString(q.MetricStat.Metric.Dimensions[0].Name))
	}
	if aws.ToInt32(q.MetricStat.Period) != 60 {
		t.Errorf("wrong period: %d", aws.ToInt32(q.MetricStat.Period))
	}
	if aws.ToString(q.MetricStat.Stat) != "Sum" {
		t.Errorf("wrong stat: %s", aws.ToString(q.MetricStat.Stat))
	}
	// m-prefix metrics should NOT return data
	if aws.ToBool(q.ReturnData) {
		t.Error("m-prefix metric should have ReturnData=false")
	}
}

func TestToMetricDataQueries_MixedExpressionAndMetric(t *testing.T) {
	defs := []experiment.MetricDef{
		{
			ID: "m1",
			MetricStat: &experiment.MetricStat{
				Metric: experiment.MetricRef{
					Namespace:  "Gideon/Chat",
					MetricName: "ChatRequestError",
				},
				Period: 60,
				Stat:   "Sum",
			},
		},
		{
			ID:         "e1",
			Expression: "IF(m1 > 5, 0, 1)",
			Label:      "ErrorRateOk",
		},
	}

	queries := metrics.ToMetricDataQueries(defs)
	if len(queries) != 2 {
		t.Fatalf("expected 2 queries, got %d", len(queries))
	}

	// m1 should not return data
	if aws.ToBool(queries[0].ReturnData) {
		t.Error("m1 should have ReturnData=false")
	}
	// e1 should return data
	if !aws.ToBool(queries[1].ReturnData) {
		t.Error("e1 should have ReturnData=true")
	}
}

func TestToMetricDataQueries_DefaultPeriodAndStat(t *testing.T) {
	defs := []experiment.MetricDef{
		{
			ID: "m1",
			MetricStat: &experiment.MetricStat{
				Metric: experiment.MetricRef{
					Namespace:  "AWS/Lambda",
					MetricName: "Errors",
				},
				// Period and Stat are zero-value
			},
		},
	}

	queries := metrics.ToMetricDataQueries(defs)
	q := queries[0]

	if aws.ToInt32(q.MetricStat.Period) != 60 {
		t.Errorf("expected default period 60, got %d", aws.ToInt32(q.MetricStat.Period))
	}
	if aws.ToString(q.MetricStat.Stat) != "Sum" {
		t.Errorf("expected default stat Sum, got %s", aws.ToString(q.MetricStat.Stat))
	}
}

func TestToMetricDataQueries_EmptySlice(t *testing.T) {
	queries := metrics.ToMetricDataQueries(nil)
	if len(queries) != 0 {
		t.Errorf("expected 0 queries for nil input, got %d", len(queries))
	}

	queries = metrics.ToMetricDataQueries([]experiment.MetricDef{})
	if len(queries) != 0 {
		t.Errorf("expected 0 queries for empty input, got %d", len(queries))
	}
}

func TestIsExpressionMetric(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"e1", true},
		{"e99", true},
		{"eCustomLabel", true},
		{"m1", false},
		{"m99", false},
		{"x1", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := metrics.IsExpressionMetric(tt.id); got != tt.want {
			t.Errorf("IsExpressionMetric(%q) = %v, want %v", tt.id, got, tt.want)
		}
	}
}

func TestExpressionFallbackLabel(t *testing.T) {
	// When label is empty, should use ID as label
	defs := []experiment.MetricDef{
		{
			ID:         "e1",
			Expression: "IF(m1 > 5, 0, 1)",
			// Label deliberately empty
		},
	}

	queries := metrics.ToMetricDataQueries(defs)
	if aws.ToString(queries[0].Label) != "e1" {
		t.Errorf("expected fallback label 'e1', got %s", aws.ToString(queries[0].Label))
	}
}
