// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause
//
// services/gideon-chaos-platform/lambda-chaos-machine/cmd/continue-execution/main.go
//
// EventBridge target — called when FIS experiment completes. Looks up the
// Step Functions task token by experimentId (via DDB GSI) and calls
// sfn.SendTaskSuccess() or sfn.SendTaskFailure() to resume the paused
// state machine execution.
//
//nolint:forbidigo // Standalone Lambda service — os.Getenv in init() is acceptable
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/sfn"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/continueexecution"
	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
)

const ddbGSIName = "experimentId-index"

var resumer *continueexecution.Resumer

func init() {
	level := slog.LevelInfo
	if l := os.Getenv("LOG_LEVEL"); l == "DEBUG" || l == "debug" {
		level = slog.LevelDebug
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})))

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		slog.Error("failed to load AWS config", "error", err)
		os.Exit(1)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)
	sfnClient := sfn.NewFromConfig(cfg)

	tableName := os.Getenv("CHAOS_DDB_TABLE")
	if tableName == "" {
		slog.Error("CHAOS_DDB_TABLE environment variable not set")
		os.Exit(1)
	}

	resumer = continueexecution.NewResumer(ddbClient, sfnClient, tableName, ddbGSIName)

	slog.Info("continue-execution Lambda initialized")
}

func handler(ctx context.Context, event experiment.EventBridgeFISEvent) (*experiment.ContinueResult, error) {
	return resumer.Resume(ctx, event)
}

func main() {
	lambda.Start(handler)
}
