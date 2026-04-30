// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause
//
// services/gideon-chaos-platform/lambda-chaos-machine/internal/evaluatehypothesis/evaluator_test.go
// @file services/gideon-chaos-platform/lambda-chaos-machine/internal/evaluatehypothesis/evaluator_test.go
// @description Verifies hypothesis evaluation behavior for chaos callbacks.
// @package evaluatehypothesis tests the evaluate-hypothesis Lambda business logic.
// @update-policy Update this header only when the file's primary responsibility materially changes.
package evaluatehypothesis

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/fis"
	fistypes "github.com/aws/aws-sdk-go-v2/service/fis/types"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
)

type cloudWatchStub struct {
	metricDataOutput *cloudwatch.GetMetricDataOutput
	metricDataInput  *cloudwatch.GetMetricDataInput
	alarmsOutput     *cloudwatch.DescribeAlarmsOutput
	alarmsInput      *cloudwatch.DescribeAlarmsInput
}

func (s *cloudWatchStub) GetMetricData(_ context.Context, params *cloudwatch.GetMetricDataInput, _ ...func(*cloudwatch.Options)) (*cloudwatch.GetMetricDataOutput, error) {
	s.metricDataInput = params
	return s.metricDataOutput, nil
}

func (s *cloudWatchStub) DescribeAlarms(_ context.Context, params *cloudwatch.DescribeAlarmsInput, _ ...func(*cloudwatch.Options)) (*cloudwatch.DescribeAlarmsOutput, error) {
	s.alarmsInput = params
	return s.alarmsOutput, nil
}

type fisStub struct {
	output *fis.GetExperimentOutput
	err    error
}

func (s *fisStub) GetExperiment(_ context.Context, _ *fis.GetExperimentInput, _ ...func(*fis.Options)) (*fis.GetExperimentOutput, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.output, nil
}

type ddbStub struct{}

func (s *ddbStub) PutItem(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	return &dynamodb.PutItemOutput{}, nil
}

func (s *ddbStub) Query(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return &dynamodb.QueryOutput{}, nil
}

func (s *ddbStub) DeleteItem(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return &dynamodb.DeleteItemOutput{}, nil
}

func TestEvaluatorEvaluate_PassesWithRecoveryWindow(t *testing.T) {
	t.Parallel()

	experimentStart := time.Date(2026, time.April, 27, 11, 30, 0, 0, time.UTC)
	experimentEnd := time.Date(2026, time.April, 27, 11, 35, 0, 0, time.UTC)
	cloudWatchClient := &cloudWatchStub{
		metricDataOutput: &cloudwatch.GetMetricDataOutput{
			MetricDataResults: []cwtypes.MetricDataResult{{
				Id:     aws.String("e1"),
				Label:  aws.String("availability"),
				Values: []float64{1},
			}},
		},
		alarmsOutput: &cloudwatch.DescribeAlarmsOutput{
			MetricAlarms: []cwtypes.MetricAlarm{{
				AlarmName:  aws.String("steady-state-alarm"),
				StateValue: cwtypes.StateValueOk,
			}},
		},
	}
	deleteCalled := false
	evaluator := NewEvaluator(cloudWatchClient, &fisStub{output: &fis.GetExperimentOutput{
		Experiment: &fistypes.Experiment{StartTime: &experimentStart, EndTime: &experimentEnd},
	}}, &ddbStub{}, "chaos-tests")
	evaluator.now = func() time.Time { return time.Date(2026, time.April, 27, 12, 0, 0, 0, time.UTC) }
	evaluator.deleteRecord = func(_ context.Context, _ experiment.DDBClient, table, testID, experimentID string) error {
		deleteCalled = true
		if table != "chaos-tests" || testID != "test-001" || experimentID != "EXP123" {
			t.Fatalf("unexpected delete args: table=%q testID=%q experimentID=%q", table, testID, experimentID)
		}

		return nil
	}

	result, err := evaluator.Evaluate(context.Background(), experiment.Input{
		TestID:           "test-001",
		ExperimentID:     "EXP123",
		RecoveryDelay:    60,
		RecoveryDuration: 180,
		Manifest: experiment.Manifest{
			Scenario: experiment.ScenarioMetadata{HealthLayer: "dependency"},
		},
		SteadyStateMetrics: []experiment.MetricDef{{
			ID:         "e1",
			Expression: "IF(m1 > 0, 1, 0)",
		}},
		SteadyStateAlarms: []string{"steady-state-alarm"},
	})
	if err != nil {
		t.Fatalf("Evaluate() returned error: %v", err)
	}

	if result == nil || !result.HypothesisPassed {
		t.Fatalf("Evaluate() result = %#v, want passing result", result)
	}

	if result.Manifest.Scenario.HealthLayer != "dependency" {
		t.Fatalf("result health layer = %q, want dependency", result.Manifest.Scenario.HealthLayer)
	}

	if !deleteCalled {
		t.Fatal("deleteRecord was not called")
	}

	wantStart := experimentEnd.Add(60 * time.Second)
	if got := cloudWatchClient.metricDataInput.StartTime.UTC(); !got.Equal(wantStart) {
		t.Fatalf("metric start time = %s, want %s", got, wantStart)
	}

	wantEnd := wantStart.Add(180 * time.Second)
	if got := cloudWatchClient.metricDataInput.EndTime.UTC(); !got.Equal(wantEnd) {
		t.Fatalf("metric end time = %s, want %s", got, wantEnd)
	}
}

func TestEvaluatorEvaluate_ReturnsJSONErrorOnFailure(t *testing.T) {
	t.Parallel()

	cloudWatchClient := &cloudWatchStub{
		metricDataOutput: &cloudwatch.GetMetricDataOutput{
			MetricDataResults: []cwtypes.MetricDataResult{{
				Id:     aws.String("e1"),
				Label:  aws.String("availability"),
				Values: []float64{0},
			}},
		},
		alarmsOutput: &cloudwatch.DescribeAlarmsOutput{},
	}
	evaluator := NewEvaluator(cloudWatchClient, &fisStub{}, &ddbStub{}, "chaos-tests")
	evaluator.now = func() time.Time { return time.Date(2026, time.April, 27, 12, 0, 0, 0, time.UTC) }
	evaluator.deleteRecord = func(context.Context, experiment.DDBClient, string, string, string) error { return nil }

	result, err := evaluator.Evaluate(context.Background(), experiment.Input{
		TestID:       "test-001",
		ExperimentID: "EXP123",
		SteadyStateMetrics: []experiment.MetricDef{{
			ID:         "e1",
			Expression: "IF(m1 > 0, 1, 0)",
		}},
	})
	if err == nil {
		t.Fatal("Evaluate() error = nil, want JSON failure payload")
	}

	if result != nil {
		t.Fatalf("Evaluate() result = %#v, want nil", result)
	}

	var failure experiment.HypothesisResult
	if unmarshalErr := json.Unmarshal([]byte(err.Error()), &failure); unmarshalErr != nil {
		t.Fatalf("failure payload is not valid JSON: %v", unmarshalErr)
	}

	if failure.HypothesisPassed {
		t.Fatal("failure HypothesisPassed = true, want false")
	}

	if len(failure.Errors) != 1 {
		t.Fatalf("failure errors = %v, want 1 error", failure.Errors)
	}
}
