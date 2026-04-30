// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause
//
// services/gideon-chaos-platform/lambda-chaos-machine/internal/startexperiment/starter_test.go
// @file services/gideon-chaos-platform/lambda-chaos-machine/internal/startexperiment/starter_test.go
// @description Verifies start-experiment orchestration and persistence behavior.
// @package startexperiment tests the start-experiment Lambda business logic.
// @update-policy Update this header only when the file's primary responsibility materially changes.
package startexperiment

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/fis"
	fistypes "github.com/aws/aws-sdk-go-v2/service/fis/types"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
)

type fisStub struct {
	startOutput *fis.StartExperimentOutput
	startInput  *fis.StartExperimentInput
	startErr    error
}

func (s *fisStub) StartExperiment(_ context.Context, params *fis.StartExperimentInput, _ ...func(*fis.Options)) (*fis.StartExperimentOutput, error) {
	s.startInput = params

	if s.startErr != nil {
		return nil, s.startErr
	}

	return s.startOutput, nil
}

type ddbStub struct {
	putItemInput *dynamodb.PutItemInput
	putItemErr   error
}

func (s *ddbStub) PutItem(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	s.putItemInput = params

	if s.putItemErr != nil {
		return nil, s.putItemErr
	}

	return &dynamodb.PutItemOutput{}, nil
}

func (s *ddbStub) Query(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return &dynamodb.QueryOutput{}, nil
}

func (s *ddbStub) DeleteItem(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return &dynamodb.DeleteItemOutput{}, nil
}

func TestStarterStart_Success(t *testing.T) {
	t.Parallel()

	fisClient := &fisStub{
		startOutput: &fis.StartExperimentOutput{
			Experiment: &fistypes.Experiment{Id: aws.String("EXP123")},
		},
	}
	ddbClient := &ddbStub{}
	starter := NewStarter(fisClient, ddbClient, "chaos-tests", "lambda-chaos-machine")

	result, err := starter.Start(context.Background(), experiment.Input{
		TestID:               "test-001",
		TaskToken:            "token-123",
		ExperimentTemplateID: "tmpl-001",
		TestDescription:      "Start experiment test",
		Manifest: experiment.Manifest{
			Scenario: experiment.ScenarioMetadata{Class: "dependency-failure"},
		},
	})
	if err != nil {
		t.Fatalf("Start() returned error: %v", err)
	}

	if result == nil {
		t.Fatal("Start() returned nil result")
	}

	if result.TestID != "test-001" {
		t.Fatalf("result TestID = %q, want test-001", result.TestID)
	}

	if result.ExperimentID != "EXP123" {
		t.Fatalf("result ExperimentID = %q, want EXP123", result.ExperimentID)
	}

	if result.Manifest.Scenario.Class != "dependency-failure" {
		t.Fatalf("result scenario class = %q, want dependency-failure", result.Manifest.Scenario.Class)
	}

	if fisClient.startInput == nil {
		t.Fatal("StartExperiment() was not called")
	}

	if got := aws.ToString(fisClient.startInput.ExperimentTemplateId); got != "tmpl-001" {
		t.Fatalf("experiment template = %q, want tmpl-001", got)
	}

	if got := aws.ToString(fisClient.startInput.ClientToken); got != "chaos-machine-test-001" {
		t.Fatalf("client token = %q, want chaos-machine-test-001", got)
	}

	if got := fisClient.startInput.Tags["managed-by"]; got != "chaos-machine" {
		t.Fatalf("managed-by tag = %q, want chaos-machine", got)
	}

	if ddbClient.putItemInput == nil {
		t.Fatal("PutItem() was not called")
	}

	if got := aws.ToString(ddbClient.putItemInput.TableName); got != "chaos-tests" {
		t.Fatalf("table name = %q, want chaos-tests", got)
	}
}

func TestStarterStart_ReturnsErrorWhenStartFails(t *testing.T) {
	t.Parallel()

	fisClient := &fisStub{startErr: context.DeadlineExceeded}
	starter := NewStarter(fisClient, &ddbStub{}, "chaos-tests", "lambda-chaos-machine")

	result, err := starter.Start(context.Background(), experiment.Input{
		TestID:               "test-001",
		ExperimentTemplateID: "tmpl-001",
	})
	if err == nil {
		t.Fatal("Start() error = nil, want error")
	}

	if result != nil {
		t.Fatalf("Start() result = %#v, want nil", result)
	}

	if got := err.Error(); got != "FIS StartExperiment: context deadline exceeded" {
		t.Fatalf("error = %q, want wrapped FIS error", got)
	}
}

func TestStarterStart_ReturnsErrorWhenPersistenceFails(t *testing.T) {
	t.Parallel()

	fisClient := &fisStub{
		startOutput: &fis.StartExperimentOutput{
			Experiment: &fistypes.Experiment{Id: aws.String("EXP123")},
		},
	}
	ddbClient := &ddbStub{putItemErr: context.Canceled}
	starter := NewStarter(fisClient, ddbClient, "chaos-tests", "lambda-chaos-machine")

	result, err := starter.Start(context.Background(), experiment.Input{
		TestID:               "test-001",
		TaskToken:            "token-123",
		ExperimentTemplateID: "tmpl-001",
	})
	if err == nil {
		t.Fatal("Start() error = nil, want error")
	}

	if result != nil {
		t.Fatalf("Start() result = %#v, want nil", result)
	}

	if got := err.Error(); got != "store task token: DDB PutItem: context canceled" {
		t.Fatalf("error = %q, want wrapped persistence error", got)
	}
}
