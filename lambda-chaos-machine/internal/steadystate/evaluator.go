// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause
//
// services/gideon-chaos-platform/lambda-chaos-machine/internal/steadystate/evaluator.go
// @file services/gideon-chaos-platform/lambda-chaos-machine/internal/steadystate/evaluator.go
// @description Evaluates chaos steady-state metrics and alarms.
// @update-policy Update this header only when the file's primary responsibility materially changes.

// Package steadystate provides the steady-state Lambda business logic.
package steadystate

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/metrics"
)

type cloudWatchAPI interface {
	GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
	DescribeAlarms(ctx context.Context, params *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error)
}

// Evaluator owns steady-state verification for chaos experiments.
type Evaluator struct {
	cloudWatchClient cloudWatchAPI
	now              func() time.Time
}

// NewEvaluator constructs an Evaluator using the default UTC clock.
func NewEvaluator(cloudWatchClient cloudWatchAPI) *Evaluator {
	return &Evaluator{
		cloudWatchClient: cloudWatchClient,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

// NewEvaluatorWithNow constructs an Evaluator with an injected clock for tests.
func NewEvaluatorWithNow(cloudWatchClient cloudWatchAPI, now func() time.Time) *Evaluator {
	if now == nil {
		return NewEvaluator(cloudWatchClient)
	}

	return &Evaluator{cloudWatchClient: cloudWatchClient, now: now}
}

// Evaluate verifies the configured steady-state metrics and alarms.
func (e *Evaluator) Evaluate(ctx context.Context, event experiment.Input) (*experiment.SteadyStateResult, error) {
	testID := event.TestID
	if testID == "" {
		testID = "unknown"
	}

	slog.Info("steady-state evaluation starting", "testId", testID)

	var failures []string

	if len(event.SteadyStateMetrics) > 0 {
		queryFailures, err := e.evaluateMetrics(ctx, event.SteadyStateMetrics)
		if err != nil {
			slog.Error("CloudWatch GetMetricData failed", "error", err)
			failures = append(failures, fmt.Sprintf("CloudWatch metric query failed: %v", err))
		} else {
			failures = append(failures, queryFailures...)
		}
	}

	if len(event.SteadyStateAlarms) > 0 {
		alarmFailures, err := e.evaluateAlarms(ctx, event.SteadyStateAlarms)
		if err != nil {
			slog.Error("CloudWatch DescribeAlarms failed", "error", err)
			failures = append(failures, fmt.Sprintf("CloudWatch alarm query failed: %v", err))
		} else {
			failures = append(failures, alarmFailures...)
		}
	}

	result := &experiment.SteadyStateResult{
		TestID:            testID,
		SteadyStatePassed: len(failures) == 0,
		Errors:            failures,
	}

	if len(failures) > 0 {
		slog.Error("steady-state FAILED — blocking experiment", "testId", testID, "errorCount", len(failures))

		encodedResult, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return nil, fmt.Errorf("marshal steady-state result: %w", marshalErr)
		}

		return nil, fmt.Errorf("%s", string(encodedResult))
	}

	slog.Info("steady-state PASSED — proceeding with experiment", "testId", testID)

	return result, nil
}

func (e *Evaluator) evaluateMetrics(ctx context.Context, metricDefinitions []experiment.MetricDef) ([]string, error) {
	metricQueries := metrics.ToMetricDataQueries(metricDefinitions)
	windowEnd := e.now()

	response, err := e.cloudWatchClient.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
		MetricDataQueries: metricQueries,
		StartTime:         timePtr(windowEnd.Add(-5 * time.Minute)),
		EndTime:           timePtr(windowEnd),
	})
	if err != nil {
		return nil, err
	}

	var failures []string

	for _, result := range response.MetricDataResults {
		metricID := deref(result.Id)
		if !metrics.IsExpressionMetric(metricID) {
			continue
		}

		label := deref(result.Label)
		if label == "" {
			label = metricID
		}

		if len(result.Values) == 0 {
			slog.Warn("metric has no data points", "metricId", metricID, "label", label)
			continue
		}

		latestValue := result.Values[0]
		if latestValue == 0 {
			failure := fmt.Sprintf("Metric expression '%s' (%s) returned 0 (FAIL)", label, metricID)
			failures = append(failures, failure)

			slog.Error("steady-state metric failed", "metricId", metricID, "label", label, "value", latestValue)

			continue
		}

		slog.Info("steady-state metric ok", "metricId", metricID, "label", label, "value", latestValue)
	}

	return failures, nil
}

func (e *Evaluator) evaluateAlarms(ctx context.Context, alarmNames []string) ([]string, error) {
	response, err := e.cloudWatchClient.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{
		AlarmNames: alarmNames,
	})
	if err != nil {
		return nil, err
	}

	var failures []string

	for index := range response.MetricAlarms {
		alarm := &response.MetricAlarms[index]
		alarmName := deref(alarm.AlarmName)
		alarmState := string(alarm.StateValue)

		if alarm.StateValue == "ALARM" {
			failure := fmt.Sprintf("Alarm '%s' is in ALARM state — system is not in steady state", alarmName)
			failures = append(failures, failure)

			slog.Error("steady-state alarm triggered", "alarm", alarmName, "state", alarmState)

			continue
		}

		slog.Info("steady-state alarm ok", "alarm", alarmName, "state", alarmState)
	}

	return failures, nil
}

func timePtr(t time.Time) *time.Time { return &t }

func deref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
