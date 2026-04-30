// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause
//
// services/gideon-chaos-platform/lambda-chaos-machine/internal/evaluatehypothesis/evaluator.go
// @file services/gideon-chaos-platform/lambda-chaos-machine/internal/evaluatehypothesis/evaluator.go
// @description Evaluates post-experiment hypothesis evidence and outcome.
// @update-policy Update this header only when the file's primary responsibility materially changes.

// Package evaluatehypothesis provides the evaluate-hypothesis Lambda business logic.
package evaluatehypothesis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/fis"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/metrics"
)

type cloudWatchAPI interface {
	GetMetricData(ctx context.Context, params *cloudwatch.GetMetricDataInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error)
	DescribeAlarms(ctx context.Context, params *cloudwatch.DescribeAlarmsInput, optFns ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error)
}

type fisAPI interface {
	GetExperiment(ctx context.Context, params *fis.GetExperimentInput, optFns ...func(*fis.Options)) (*fis.GetExperimentOutput, error)
}

type deleteRecordFunc func(ctx context.Context, client experiment.DDBClient, table, testID, experimentID string) error

// Evaluator owns the evaluate-hypothesis Lambda behavior.
type Evaluator struct {
	cloudWatchClient cloudWatchAPI
	fisClient        fisAPI
	ddbClient        experiment.DDBClient
	tableName        string
	now              func() time.Time
	deleteRecord     deleteRecordFunc
}

// NewEvaluator constructs an Evaluator with the dependencies required to score a hypothesis.
func NewEvaluator(cloudWatchClient cloudWatchAPI, fisClient fisAPI, ddbClient experiment.DDBClient, tableName string) *Evaluator {
	return &Evaluator{
		cloudWatchClient: cloudWatchClient,
		fisClient:        fisClient,
		ddbClient:        ddbClient,
		tableName:        tableName,
		now: func() time.Time {
			return time.Now().UTC()
		},
		deleteRecord: experiment.DeleteRecord,
	}
}

// Evaluate verifies the post-experiment hypothesis and returns the audit record.
func (e *Evaluator) Evaluate(ctx context.Context, event experiment.Input) (*experiment.HypothesisResult, error) {
	testID := event.TestID
	if testID == "" {
		testID = "unknown"
	}

	experimentID := event.ExperimentID
	if experimentID == "" {
		experimentID = "unknown"
	}

	recoveryDelay := time.Duration(event.RecoveryDelay) * time.Second

	recoveryDuration := time.Duration(event.RecoveryDuration) * time.Second
	if recoveryDuration == 0 {
		recoveryDuration = 5 * time.Minute
	}

	slog.Info("hypothesis evaluation starting", "testId", testID, "experimentId", experimentID)

	startTime, endTime := e.computeEvaluationWindow(ctx, testID, experimentID, recoveryDelay, recoveryDuration)

	var (
		failures []string
		evidence []experiment.EvidenceItem
	)

	if len(event.SteadyStateMetrics) > 0 {
		metricEvidence, metricFailures := e.evaluateMetrics(ctx, event.SteadyStateMetrics, startTime, endTime)
		evidence = append(evidence, metricEvidence...)
		failures = append(failures, metricFailures...)
	}

	if len(event.SteadyStateAlarms) > 0 {
		alarmEvidence, alarmFailures := e.evaluateAlarms(ctx, event.SteadyStateAlarms)
		evidence = append(evidence, alarmEvidence...)
		failures = append(failures, alarmFailures...)
	}

	if deleteErr := e.deleteRecord(ctx, e.ddbClient, e.tableName, testID, experimentID); deleteErr != nil {
		slog.Warn("DDB cleanup failed (non-fatal, TTL will expire)", "error", deleteErr)
	} else {
		slog.Info("DynamoDB record cleaned up", "testId", testID)
	}

	result := &experiment.HypothesisResult{
		TestID:           testID,
		ExperimentID:     experimentID,
		Manifest:         event.Manifest,
		HypothesisPassed: len(failures) == 0,
		Errors:           failures,
		Evidence:         evidence,
		EvaluationWindow: experiment.TimeWindow{
			StartTime: experiment.FormatTime(startTime),
			EndTime:   experiment.FormatTime(endTime),
		},
	}

	if len(failures) > 0 {
		slog.Error("hypothesis FAILED — resilience patterns did not hold", "testId", testID, "errorCount", len(failures))

		encodedResult, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("marshal hypothesis result: %w", err)
		}

		return nil, fmt.Errorf("%s", string(encodedResult))
	}

	slog.Info("hypothesis PASSED — system maintained steady state during fault", "testId", testID)

	return result, nil
}

func (e *Evaluator) computeEvaluationWindow(ctx context.Context, testID, experimentID string, recoveryDelay, recoveryDuration time.Duration) (startTime, endTime time.Time) {
	now := e.now()
	endTime = now
	startTime = now.Add(-10 * time.Minute)

	var experimentStart, experimentEnd *time.Time

	out, err := e.fisClient.GetExperiment(ctx, &fis.GetExperimentInput{Id: aws.String(experimentID)})
	if err != nil {
		slog.Warn("could not retrieve FIS experiment details", "error", err)
	} else if out != nil && out.Experiment != nil {
		experimentStart = out.Experiment.StartTime
		experimentEnd = out.Experiment.EndTime

		slog.Info("FIS experiment time window",
			"testId", testID,
			"start", timeStr(experimentStart),
			"end", timeStr(experimentEnd))
	}

	if experimentStart != nil {
		startTime = *experimentStart
	}

	if experimentEnd != nil {
		startTime = experimentEnd.Add(recoveryDelay)
		endTime = startTime.Add(recoveryDuration)
	}

	return startTime, endTime
}

func (e *Evaluator) evaluateMetrics(ctx context.Context, metricDefinitions []experiment.MetricDef, startTime, endTime time.Time) (evidence []experiment.EvidenceItem, failures []string) {
	queries := metrics.ToMetricDataQueries(metricDefinitions)

	output, err := e.cloudWatchClient.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
		MetricDataQueries: queries,
		StartTime:         &startTime,
		EndTime:           &endTime,
	})
	if err != nil {
		slog.Error("metric evaluation failed", "error", err)
		return nil, []string{fmt.Sprintf("Metric evaluation error: %v", err)}
	}

	for _, result := range output.MetricDataResults {
		metricID := deref(result.Id)
		if !metrics.IsExpressionMetric(metricID) {
			continue
		}

		label := deref(result.Label)
		if label == "" {
			label = metricID
		}

		passed := true
		if len(result.Values) > 0 && result.Values[0] == 0 {
			passed = false
			failure := fmt.Sprintf("Post-experiment metric '%s' (%s) = 0 (FAIL)", label, metricID)
			failures = append(failures, failure)

			slog.Error("hypothesis metric failed", "metricId", metricID, "label", label)
		} else {
			slog.Info("hypothesis metric ok", "metricId", metricID, "label", label)
		}

		values := result.Values
		if len(values) > 3 {
			values = values[:3]
		}

		evidence = append(evidence, experiment.EvidenceItem{
			MetricID: metricID,
			Label:    label,
			Passed:   passed,
			Values:   values,
		})
	}

	return evidence, failures
}

func (e *Evaluator) evaluateAlarms(ctx context.Context, alarmNames []string) (evidence []experiment.EvidenceItem, failures []string) {
	output, err := e.cloudWatchClient.DescribeAlarms(ctx, &cloudwatch.DescribeAlarmsInput{AlarmNames: alarmNames})
	if err != nil {
		slog.Error("alarm evaluation failed", "error", err)
		return nil, []string{fmt.Sprintf("Alarm evaluation error: %v", err)}
	}

	for index := range output.MetricAlarms {
		alarm := &output.MetricAlarms[index]
		name := deref(alarm.AlarmName)
		state := string(alarm.StateValue)

		passed := alarm.StateValue != cwtypes.StateValueAlarm
		if !passed {
			failure := fmt.Sprintf("Post-experiment alarm '%s' still in ALARM state", name)
			failures = append(failures, failure)

			slog.Error("hypothesis alarm still triggered", "alarm", name)
		} else {
			slog.Info("hypothesis alarm ok", "alarm", name, "state", state)
		}

		evidence = append(evidence, experiment.EvidenceItem{
			AlarmName: name,
			State:     state,
			Passed:    passed,
		})
	}

	return evidence, failures
}

func timeStr(t *time.Time) string {
	if t == nil {
		return "<nil>"
	}

	return t.UTC().Format(time.RFC3339)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
