// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause
//
// services/gideon-chaos-platform/lambda-chaos-machine/cmd/steady-state/main.go
//
// Verifies system is in steady state before a chaos experiment starts.
// Queries CloudWatch metrics/alarms and returns PASS/FAIL.
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

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/steadystate"
)

var evaluator *steadystate.Evaluator

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

	evaluator = steadystate.NewEvaluator(cloudwatch.NewFromConfig(cfg))

	slog.Info("steady-state Lambda initialized")
}

func handler(ctx context.Context, event experiment.Input) (*experiment.SteadyStateResult, error) {
	return evaluator.Evaluate(ctx, event)
}

func main() {
	lambda.Start(handler)
}
