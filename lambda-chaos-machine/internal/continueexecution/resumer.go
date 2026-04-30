// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause
//
// services/gideon-chaos-platform/lambda-chaos-machine/internal/continueexecution/resumer.go
// @file services/gideon-chaos-platform/lambda-chaos-machine/internal/continueexecution/resumer.go
// @description Resumes Step Functions after chaos experiment completion.
// @update-policy Update this header only when the file's primary responsibility materially changes.

// Package continueexecution provides the continue-execution Lambda business logic.
package continueexecution

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
)

type sfnAPI interface {
	SendTaskSuccess(ctx context.Context, params *sfn.SendTaskSuccessInput, optFns ...func(*sfn.Options)) (*sfn.SendTaskSuccessOutput, error)
	SendTaskFailure(ctx context.Context, params *sfn.SendTaskFailureInput, optFns ...func(*sfn.Options)) (*sfn.SendTaskFailureOutput, error)
}

type queryTaskTokenFunc func(ctx context.Context, client experiment.DDBClient, table, gsiName, experimentID string) (*experiment.DDBItem, error)

// Resumer owns the continue-execution Lambda behavior.
type Resumer struct {
	ddbClient      experiment.DDBClient
	sfnClient      sfnAPI
	tableName      string
	gsiName        string
	queryTaskToken queryTaskTokenFunc
}

// NewResumer constructs a Resumer with the dependencies required to resume Step Functions.
func NewResumer(ddbClient experiment.DDBClient, sfnClient sfnAPI, tableName, gsiName string) *Resumer {
	return &Resumer{
		ddbClient:      ddbClient,
		sfnClient:      sfnClient,
		tableName:      tableName,
		gsiName:        gsiName,
		queryTaskToken: experiment.QueryTaskToken,
	}
}

// Resume handles FIS terminal-state callbacks and resumes the paused Step Functions execution.
func (r *Resumer) Resume(ctx context.Context, event experiment.EventBridgeFISEvent) (*experiment.ContinueResult, error) {
	experimentID := event.Detail.ExperimentID
	status := event.Detail.State.Status

	if experimentID == "" {
		slog.Error("no experiment-id in EventBridge event")
		return &experiment.ContinueResult{Success: false, Reason: "missing experiment-id"}, nil
	}

	slog.Info("FIS experiment state change received", "experimentId", experimentID, "status", status)

	if !isTerminalStatus(status) {
		slog.Info("non-terminal state — ignoring", "experimentId", experimentID, "status", status)

		return &experiment.ContinueResult{
			Success: true,
			Skipped: true,
			Reason:  fmt.Sprintf("non-terminal status: %s", status),
		}, nil
	}

	item, err := r.queryTaskToken(ctx, r.ddbClient, r.tableName, r.gsiName, experimentID)
	if err != nil {
		slog.Error("DynamoDB GSI query failed", "experimentId", experimentID, "error", err)
		return nil, fmt.Errorf("query task token: %w", err)
	}

	if item == nil {
		slog.Warn("no task token found — may have already been processed", "experimentId", experimentID)
		return &experiment.ContinueResult{Success: false, Reason: "task token not found"}, nil
	}

	testID := item.TestID
	slog.Info("task token found — resuming Step Functions", "testId", testID, "experimentId", experimentID)

	if status == "completed" {
		if successErr := r.sendSuccess(ctx, item.TaskToken, testID, experimentID, status); successErr != nil {
			slog.Error("SendTaskSuccess failed", "testId", testID, "error", successErr)
			return nil, fmt.Errorf("SendTaskSuccess: %w", successErr)
		}

		slog.Info("Step Functions resumed (success)", "testId", testID)
	} else {
		if failureErr := r.sendFailure(ctx, item.TaskToken, experimentID, status); failureErr != nil {
			slog.Error("SendTaskFailure failed", "testId", testID, "error", failureErr)
			return nil, fmt.Errorf("SendTaskFailure: %w", failureErr)
		}

		slog.Info("Step Functions resumed (failure)", "testId", testID, "status", status)
	}

	return &experiment.ContinueResult{
		Success:      true,
		TestID:       testID,
		ExperimentID: experimentID,
	}, nil
}

func (r *Resumer) sendSuccess(ctx context.Context, taskToken, testID, experimentID, status string) error {
	payload, err := json.Marshal(map[string]string{
		"testId":           testID,
		"experimentId":     experimentID,
		"experimentStatus": status,
	})
	if err != nil {
		return fmt.Errorf("marshal success payload: %w", err)
	}

	_, err = r.sfnClient.SendTaskSuccess(ctx, &sfn.SendTaskSuccessInput{
		TaskToken: aws.String(taskToken),
		Output:    aws.String(string(payload)),
	})

	return err
}

func (r *Resumer) sendFailure(ctx context.Context, taskToken, experimentID, status string) error {
	cause, err := json.Marshal(map[string]string{
		"experimentId": experimentID,
		"status":       status,
	})
	if err != nil {
		return fmt.Errorf("marshal failure payload: %w", err)
	}

	_, err = r.sfnClient.SendTaskFailure(ctx, &sfn.SendTaskFailureInput{
		TaskToken: aws.String(taskToken),
		Error:     aws.String("ExperimentFailed"),
		Cause:     aws.String(string(cause)),
	})

	return err
}

func isTerminalStatus(status string) bool {
	switch status {
	case "completed", "failed", "stopped":
		return true
	default:
		return false
	}
}
