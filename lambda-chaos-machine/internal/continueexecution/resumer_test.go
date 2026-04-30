// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause
//
// services/gideon-chaos-platform/lambda-chaos-machine/internal/continueexecution/resumer_test.go
// @file services/gideon-chaos-platform/lambda-chaos-machine/internal/continueexecution/resumer_test.go
// @description Verifies continue-execution resume behavior for FIS callbacks.
// @package continueexecution tests the continue-execution Lambda business logic.
// @update-policy Update this header only when the file's primary responsibility materially changes.
package continueexecution

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sfn"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
)

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

type sfnStub struct {
	successInput *sfn.SendTaskSuccessInput
	failureInput *sfn.SendTaskFailureInput
	successErr   error
	failureErr   error
}

func (s *sfnStub) SendTaskSuccess(_ context.Context, params *sfn.SendTaskSuccessInput, _ ...func(*sfn.Options)) (*sfn.SendTaskSuccessOutput, error) {
	s.successInput = params
	if s.successErr != nil {
		return nil, s.successErr
	}

	return &sfn.SendTaskSuccessOutput{}, nil
}

func (s *sfnStub) SendTaskFailure(_ context.Context, params *sfn.SendTaskFailureInput, _ ...func(*sfn.Options)) (*sfn.SendTaskFailureOutput, error) {
	s.failureInput = params
	if s.failureErr != nil {
		return nil, s.failureErr
	}

	return &sfn.SendTaskFailureOutput{}, nil
}

func TestResumerResume_SkipsNonTerminalStatus(t *testing.T) {
	t.Parallel()

	resumer := NewResumer(&ddbStub{}, &sfnStub{}, "chaos-tests", "experimentId-index")
	result, err := resumer.Resume(context.Background(), experiment.EventBridgeFISEvent{
		Detail: experiment.FISEventDetail{
			ExperimentID: "EXP123",
			State:        experiment.FISState{Status: "running"},
		},
	})
	if err != nil {
		t.Fatalf("Resume() returned error: %v", err)
	}

	if result == nil || !result.Skipped || !result.Success {
		t.Fatalf("Resume() result = %#v, want skipped success", result)
	}
}

func TestResumerResume_ResumesSuccessPath(t *testing.T) {
	t.Parallel()

	sfnClient := &sfnStub{}
	resumer := NewResumer(&ddbStub{}, sfnClient, "chaos-tests", "experimentId-index")
	resumer.queryTaskToken = func(_ context.Context, _ experiment.DDBClient, table, gsiName, experimentID string) (*experiment.DDBItem, error) {
		if table != "chaos-tests" || gsiName != "experimentId-index" || experimentID != "EXP123" {
			t.Fatalf("unexpected query args: table=%q gsi=%q experimentID=%q", table, gsiName, experimentID)
		}

		return &experiment.DDBItem{TestID: "test-001", TaskToken: "token-123"}, nil
	}

	result, err := resumer.Resume(context.Background(), experiment.EventBridgeFISEvent{
		Detail: experiment.FISEventDetail{
			ExperimentID: "EXP123",
			State:        experiment.FISState{Status: "completed"},
		},
	})
	if err != nil {
		t.Fatalf("Resume() returned error: %v", err)
	}

	if result == nil || !result.Success || result.TestID != "test-001" {
		t.Fatalf("Resume() result = %#v, want success for test-001", result)
	}

	if sfnClient.successInput == nil {
		t.Fatal("SendTaskSuccess() was not called")
	}

	if got := aws.ToString(sfnClient.successInput.TaskToken); got != "token-123" {
		t.Fatalf("task token = %q, want token-123", got)
	}
}

func TestResumerResume_ResumesFailurePath(t *testing.T) {
	t.Parallel()

	sfnClient := &sfnStub{}
	resumer := NewResumer(&ddbStub{}, sfnClient, "chaos-tests", "experimentId-index")
	resumer.queryTaskToken = func(_ context.Context, _ experiment.DDBClient, _, _, _ string) (*experiment.DDBItem, error) {
		return &experiment.DDBItem{TestID: "test-001", TaskToken: "token-123"}, nil
	}

	result, err := resumer.Resume(context.Background(), experiment.EventBridgeFISEvent{
		Detail: experiment.FISEventDetail{
			ExperimentID: "EXP123",
			State:        experiment.FISState{Status: "failed"},
		},
	})
	if err != nil {
		t.Fatalf("Resume() returned error: %v", err)
	}

	if result == nil || !result.Success || result.ExperimentID != "EXP123" {
		t.Fatalf("Resume() result = %#v, want success for EXP123", result)
	}

	if sfnClient.failureInput == nil {
		t.Fatal("SendTaskFailure() was not called")
	}

	if got := aws.ToString(sfnClient.failureInput.Error); got != "ExperimentFailed" {
		t.Fatalf("error code = %q, want ExperimentFailed", got)
	}
}

func TestResumerResume_ReturnsMissingTokenResult(t *testing.T) {
	t.Parallel()

	resumer := NewResumer(&ddbStub{}, &sfnStub{}, "chaos-tests", "experimentId-index")
	resumer.queryTaskToken = func(_ context.Context, _ experiment.DDBClient, _, _, _ string) (*experiment.DDBItem, error) {
		return nil, nil
	}

	result, err := resumer.Resume(context.Background(), experiment.EventBridgeFISEvent{
		Detail: experiment.FISEventDetail{
			ExperimentID: "EXP123",
			State:        experiment.FISState{Status: "completed"},
		},
	})
	if err != nil {
		t.Fatalf("Resume() returned error: %v", err)
	}

	if result == nil || result.Success || result.Reason != "task token not found" {
		t.Fatalf("Resume() result = %#v, want missing-token result", result)
	}
}
