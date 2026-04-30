// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause
//
// services/gideon-chaos-platform/lambda-chaos-machine/cmd/evaluate-hypothesis/main.go
//
// Evaluates whether the system maintained steady state during and after
// the chaos experiment. Queries CloudWatch metrics post-experiment and
// returns PASS/FAIL with evidence for SOC 2 audit trail.
//
//nolint:forbidigo // Standalone Lambda service — os.Getenv in init() is acceptable
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/fis"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/evaluatehypothesis"
	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
)

var evaluator *evaluatehypothesis.Evaluator

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

	cwClient := cloudwatch.NewFromConfig(cfg)
	fisClient := fis.NewFromConfig(cfg)
	ddbClient := dynamodb.NewFromConfig(cfg)

	tableName := os.Getenv("CHAOS_DDB_TABLE")
	if tableName == "" {
		slog.Error("CHAOS_DDB_TABLE environment variable not set")
		os.Exit(1)
	}

	evaluator = evaluatehypothesis.NewEvaluator(cwClient, fisClient, ddbClient, tableName)

	slog.Info("evaluate-hypothesis Lambda initialized")
}

func handler(ctx context.Context, event experiment.Input) (*experiment.HypothesisResult, error) {
	return evaluator.Evaluate(ctx, event)
}

func main() {
	lambda.Start(handler)
}
