// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause
//
// services/gideon-chaos-platform/lambda-chaos-machine/cmd/start-experiment/main.go
//
// Starts the FIS experiment and stores the Step Functions task token
// in DynamoDB for async callback by continue-execution.
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
	"github.com/aws/aws-sdk-go-v2/service/fis"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/startexperiment"
)

var starter *startexperiment.Starter

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

	fisClient := fis.NewFromConfig(cfg)
	ddbClient := dynamodb.NewFromConfig(cfg)

	executionName := os.Getenv("AWS_LAMBDA_FUNCTION_NAME")

	tableName := os.Getenv("CHAOS_DDB_TABLE")
	if tableName == "" {
		slog.Error("CHAOS_DDB_TABLE environment variable not set")
		os.Exit(1)
	}

	starter = startexperiment.NewStarter(fisClient, ddbClient, tableName, executionName)

	slog.Info("start-experiment Lambda initialized")
}

func handler(ctx context.Context, event experiment.Input) (*experiment.StartResult, error) {
	return starter.Start(ctx, event)
}

func main() {
	lambda.Start(handler)
}
