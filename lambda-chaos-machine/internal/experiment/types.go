// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause

// Package experiment provides shared types and DynamoDB helpers for chaos-machine
// Lambda functions.
package experiment

import "time"

// Input represents the chaos-machine experiment input from Step Functions.
// Fields map to the JSON Schema in internal/schema/chaos-machine-input.json.
type Input struct {
	TestID               string      `json:"testId"`
	TestDescription      string      `json:"testDescription,omitempty"`
	ExperimentTemplateID string      `json:"experimentTemplateId"`
	Manifest             Manifest    `json:"manifest,omitempty"`
	TaskToken            string      `json:"taskToken,omitempty"`
	StopConditionARNs    []string    `json:"stopConditionAlarmArns,omitempty"`
	SteadyStateMetrics   []MetricDef `json:"steadyStateMetrics,omitempty"`
	SteadyStateAlarms    []string    `json:"steadyStateAlarms,omitempty"`
	RecoveryDelay        int         `json:"recoveryDelay,omitempty"`
	RecoveryDuration     int         `json:"recoveryDuration,omitempty"`
	ExperimentID         string      `json:"experimentId,omitempty"`
	ExperimentStatus     string      `json:"experimentStatus,omitempty"`
}

// Manifest carries operator-owned experiment metadata that complements the
// execution payload without changing runtime control flow.
type Manifest struct {
	Scenario ScenarioMetadata `json:"scenario,omitempty"`
	Owner    OwnerMetadata    `json:"owner,omitempty"`
	Approval ApprovalMetadata `json:"approval,omitempty"`
	Rollback RollbackMetadata `json:"rollback,omitempty"`
	Evidence EvidenceMetadata `json:"evidence,omitempty"`
}

// ScenarioMetadata describes the experiment class and intended health surface.
type ScenarioMetadata struct {
	Class               string `json:"class,omitempty"`
	HealthLayer         string `json:"healthLayer,omitempty"`
	Environment         string `json:"environment,omitempty"`
	ExpectedResultClass string `json:"expectedResultClass,omitempty"`
}

// OwnerMetadata identifies who owns the scenario and how to contact them.
type OwnerMetadata struct {
	Name    string `json:"name,omitempty"`
	Team    string `json:"team,omitempty"`
	Contact string `json:"contact,omitempty"`
}

// ApprovalMetadata records the review tier and approvers for the experiment.
type ApprovalMetadata struct {
	Tier       string   `json:"tier,omitempty"`
	ApprovedBy []string `json:"approvedBy,omitempty"`
}

// RollbackMetadata points operators to the expected rollback path.
type RollbackMetadata struct {
	Runbook string `json:"runbook,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// EvidenceMetadata describes where evidence should be retained.
type EvidenceMetadata struct {
	Destination string `json:"destination,omitempty"`
	Retention   string `json:"retention,omitempty"`
}

// MetricDef represents a steadyStateMetrics entry (either expression or metricStat).
type MetricDef struct {
	ID         string      `json:"id"`
	Expression string      `json:"expression,omitempty"`
	Label      string      `json:"label,omitempty"`
	MetricStat *MetricStat `json:"metricStat,omitempty"`
}

// MetricStat represents a CloudWatch MetricStat definition.
type MetricStat struct {
	Metric MetricRef `json:"metric"`
	Period int32     `json:"period"`
	Stat   string    `json:"stat"`
}

// MetricRef identifies a CloudWatch metric by namespace, name, and dimensions.
type MetricRef struct {
	Namespace  string      `json:"namespace"`
	MetricName string      `json:"metricName"`
	Dimensions []Dimension `json:"dimensions,omitempty"`
}

// Dimension is a CloudWatch metric dimension.
type Dimension struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// SteadyStateResult is returned by the steady-state Lambda.
type SteadyStateResult struct {
	TestID            string   `json:"testId"`
	SteadyStatePassed bool     `json:"steadyStatePassed"`
	Errors            []string `json:"errors"`
}

// StartResult is returned by the start-experiment Lambda.
type StartResult struct {
	TestID         string   `json:"testId"`
	ExperimentID   string   `json:"experimentId"`
	ExperimentType string   `json:"experimentType"`
	Manifest       Manifest `json:"manifest,omitempty"`
}

// ContinueResult is returned by the continue-execution Lambda.
type ContinueResult struct {
	Success      bool   `json:"success"`
	TestID       string `json:"testId,omitempty"`
	ExperimentID string `json:"experimentId,omitempty"`
	Skipped      bool   `json:"skipped,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// EvidenceItem records a single metric or alarm evaluation for the SOC 2 audit trail.
type EvidenceItem struct {
	MetricID  string    `json:"metricId,omitempty"`
	AlarmName string    `json:"alarmName,omitempty"`
	Label     string    `json:"label,omitempty"`
	State     string    `json:"state,omitempty"`
	Passed    bool      `json:"passed"`
	Values    []float64 `json:"values,omitempty"`
}

// HypothesisResult is returned by the evaluate-hypothesis Lambda.
type HypothesisResult struct {
	TestID           string         `json:"testId"`
	ExperimentID     string         `json:"experimentId"`
	Manifest         Manifest       `json:"manifest,omitempty"`
	HypothesisPassed bool           `json:"hypothesisPassed"`
	Errors           []string       `json:"errors"`
	Evidence         []EvidenceItem `json:"evidence"`
	EvaluationWindow TimeWindow     `json:"evaluationWindow"`
}

// TimeWindow records the metric evaluation window for audit purposes.
type TimeWindow struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}

// EventBridgeFISEvent represents the EventBridge event for FIS state changes.
type EventBridgeFISEvent struct {
	Source     string         `json:"source"`
	DetailType string         `json:"detail-type"`
	Detail     FISEventDetail `json:"detail"`
}

// FISEventDetail is the detail payload of a FIS state change event.
type FISEventDetail struct {
	ExperimentID string   `json:"experiment-id"`
	State        FISState `json:"state"`
}

// FISState contains the experiment status.
type FISState struct {
	Status string `json:"status"`
}

// DDBItem is the DynamoDB correlation record stored by start-experiment
// and queried by continue-execution.
type DDBItem struct {
	TestID          string `dynamodbav:"testId"`
	ExperimentID    string `dynamodbav:"experimentId"`
	ExperimentType  string `dynamodbav:"experimentType"`
	TaskToken       string `dynamodbav:"taskToken"`
	ExecutionName   string `dynamodbav:"executionName"`
	TestDescription string `dynamodbav:"testDescription"`
}

// FormatTime formats a time.Time as ISO 8601 for audit output.
func FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}
