// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause
//
// services/gideon-chaos-platform/lambda-chaos-machine/internal/startexperiment/starter.go
// @file services/gideon-chaos-platform/lambda-chaos-machine/internal/startexperiment/starter.go
// @description Starts FIS experiments and stores callback correlation state.
// @update-policy Update this header only when the file's primary responsibility materially changes.

// Package startexperiment provides the start-experiment Lambda business logic.
package startexperiment

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/fis"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
)

type fisAPI interface {
	StartExperiment(ctx context.Context, params *fis.StartExperimentInput, optFns ...func(*fis.Options)) (*fis.StartExperimentOutput, error)
}

// Starter owns the start-experiment Lambda behavior.
type Starter struct {
	fisClient     fisAPI
	ddbClient     experiment.DDBClient
	tableName     string
	executionName string
}

// NewStarter constructs a Starter with the dependencies required to start and track FIS runs.
func NewStarter(fisClient fisAPI, ddbClient experiment.DDBClient, tableName, executionName string) *Starter {
	return &Starter{
		fisClient:     fisClient,
		ddbClient:     ddbClient,
		tableName:     tableName,
		executionName: executionName,
	}
}

// Start launches the configured FIS experiment and stores the callback correlation record.
func (s *Starter) Start(ctx context.Context, event experiment.Input) (*experiment.StartResult, error) {
	testID := event.TestID
	taskToken := event.TaskToken
	templateID := event.ExperimentTemplateID

	slog.Info("starting FIS experiment", "testId", testID, "templateId", templateID)

	startOutput, err := s.fisClient.StartExperiment(ctx, &fis.StartExperimentInput{
		ExperimentTemplateId: aws.String(templateID),
		Tags: map[string]string{
			"chaos-machine-test-id": testID,
			"managed-by":            "chaos-machine",
		},
		ClientToken: aws.String(fmt.Sprintf("chaos-machine-%s", testID)),
	})
	if err != nil {
		slog.Error("FIS StartExperiment failed", "testId", testID, "error", err)
		return nil, fmt.Errorf("FIS StartExperiment: %w", err)
	}

	experimentID := aws.ToString(startOutput.Experiment.Id)
	slog.Info("FIS experiment started — storing task token",
		"testId", testID, "experimentId", experimentID)

	correlationItem := experiment.DDBItem{
		TestID:          testID,
		ExperimentID:    experimentID,
		ExperimentType:  "FIS",
		TaskToken:       taskToken,
		ExecutionName:   s.executionName,
		TestDescription: event.TestDescription,
	}

	if putErr := experiment.PutTaskToken(ctx, s.ddbClient, s.tableName, correlationItem); putErr != nil {
		slog.Error("failed to store task token", "testId", testID, "error", putErr)
		return nil, fmt.Errorf("store task token: %w", putErr)
	}

	slog.Info("task token stored — waiting for EventBridge callback",
		"testId", testID, "experimentId", experimentID)

	return &experiment.StartResult{
		TestID:         testID,
		ExperimentID:   experimentID,
		ExperimentType: "FIS",
		Manifest:       event.Manifest,
	}, nil
}
