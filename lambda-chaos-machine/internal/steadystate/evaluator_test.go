// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause
//
// services/gideon-chaos-platform/lambda-chaos-machine/internal/steadystate/evaluator_test.go
/**
 * @file services/gideon-chaos-platform/lambda-chaos-machine/internal/steadystate/evaluator_test.go
 * @description Verifies steady-state evaluator behavior for metrics and alarms.
 * @package steadystate tests the steady-state Lambda business logic.
 * @update-policy Update this header only when the file's primary responsibility materially changes.
 */
package steadystate

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cloudwatchtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"

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

func TestEvaluatorEvaluate_PassesAndUsesExpectedMetricWindow(t *testing.T) {
	t.Parallel()

	windowEnd := time.Date(2026, time.April, 27, 12, 0, 0, 0, time.UTC)
	client := &cloudWatchStub{
		metricDataOutput: &cloudwatch.GetMetricDataOutput{
			MetricDataResults: []cloudwatchtypes.MetricDataResult{{
				Id:     aws.String("e1"),
				Label:  aws.String("steady-state"),
				Values: []float64{1},
			}},
		},
		alarmsOutput: &cloudwatch.DescribeAlarmsOutput{
			MetricAlarms: []cloudwatchtypes.MetricAlarm{{
				AlarmName:  aws.String("steady-state-alarm"),
				StateValue: cloudwatchtypes.StateValueOk,
			}},
		},
	}

	evaluator := NewEvaluatorWithNow(client, func() time.Time { return windowEnd })
	result, err := evaluator.Evaluate(context.Background(), experiment.Input{
		TestID: "steady-pass",
		SteadyStateMetrics: []experiment.MetricDef{{
			ID:         "e1",
			Expression: "IF(m1 > 0, 1, 0)",
		}},
		SteadyStateAlarms: []string{"steady-state-alarm"},
	})
	if err != nil {
		t.Fatalf("Evaluate() returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Evaluate() returned nil result")
	}

	if !result.SteadyStatePassed {
		t.Fatalf("Evaluate() steadyStatePassed = false, want true")
	}

	if len(result.Errors) != 0 {
		t.Fatalf("Evaluate() errors = %v, want none", result.Errors)
	}

	if client.metricDataInput == nil {
		t.Fatal("GetMetricData() was not called")
	}

	if client.alarmsInput == nil {
		t.Fatal("DescribeAlarms() was not called")
	}

	if got := client.metricDataInput.EndTime.UTC(); !got.Equal(windowEnd) {
		t.Fatalf("metric EndTime = %s, want %s", got, windowEnd)
	}

	wantStart := windowEnd.Add(-5 * time.Minute)
	if got := client.metricDataInput.StartTime.UTC(); !got.Equal(wantStart) {
		t.Fatalf("metric StartTime = %s, want %s", got, wantStart)
	}

	if len(client.alarmsInput.AlarmNames) != 1 || client.alarmsInput.AlarmNames[0] != "steady-state-alarm" {
		t.Fatalf("alarm names = %v, want [steady-state-alarm]", client.alarmsInput.AlarmNames)
	}
}

func TestEvaluatorEvaluate_ReturnsJSONErrorWhenSteadyStateFails(t *testing.T) {
	t.Parallel()

	client := &cloudWatchStub{
		metricDataOutput: &cloudwatch.GetMetricDataOutput{
			MetricDataResults: []cloudwatchtypes.MetricDataResult{{
				Id:     aws.String("e1"),
				Label:  aws.String("steady-state"),
				Values: []float64{0},
			}},
		},
		alarmsOutput: &cloudwatch.DescribeAlarmsOutput{},
	}

	evaluator := NewEvaluatorWithNow(client, func() time.Time {
		return time.Date(2026, time.April, 27, 12, 0, 0, 0, time.UTC)
	})

	result, err := evaluator.Evaluate(context.Background(), experiment.Input{
		TestID: "steady-fail",
		SteadyStateMetrics: []experiment.MetricDef{{
			ID:         "e1",
			Expression: "IF(m1 > 0, 1, 0)",
		}},
	})
	if err == nil {
		t.Fatal("Evaluate() error = nil, want JSON failure payload")
	}

	if result != nil {
		t.Fatalf("Evaluate() result = %#v, want nil on failure", result)
	}

	var failure experiment.SteadyStateResult
	if unmarshalErr := json.Unmarshal([]byte(err.Error()), &failure); unmarshalErr != nil {
		t.Fatalf("failure payload is not valid JSON: %v", unmarshalErr)
	}

	if failure.TestID != "steady-fail" {
		t.Fatalf("failure TestID = %q, want steady-fail", failure.TestID)
	}

	if failure.SteadyStatePassed {
		t.Fatal("failure payload steadyStatePassed = true, want false")
	}

	if len(failure.Errors) != 1 {
		t.Fatalf("failure errors = %v, want 1 error", failure.Errors)
	}
}
