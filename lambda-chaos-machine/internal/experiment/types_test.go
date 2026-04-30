// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause

package experiment_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
)

func TestFormatTime(t *testing.T) {
	// 2026-03-02T10:30:00Z
	ts := time.Date(2026, 3, 2, 10, 30, 0, 0, time.UTC)
	got := experiment.FormatTime(ts)
	want := "2026-03-02T10:30:00Z"
	if got != want {
		t.Errorf("FormatTime() = %q, want %q", got, want)
	}
}

func TestFormatTime_NonUTC(t *testing.T) {
	loc := time.FixedZone("EST", -5*60*60)
	ts := time.Date(2026, 3, 2, 10, 30, 0, 0, loc)
	got := experiment.FormatTime(ts)
	// Should convert to UTC
	want := "2026-03-02T15:30:00Z"
	if got != want {
		t.Errorf("FormatTime() = %q, want %q", got, want)
	}
}

func TestManifestJSONRoundTrip(t *testing.T) {
	t.Parallel()

	input := experiment.Input{
		TestID:               "test-001",
		ExperimentTemplateID: "EXT123456789",
		Manifest: experiment.Manifest{
			Scenario: experiment.ScenarioMetadata{
				Class:               "dependency-failure",
				HealthLayer:         "dependency",
				Environment:         "dev-aws",
				ExpectedResultClass: "degraded-but-contained",
			},
			Owner: experiment.OwnerMetadata{
				Name:    "Chaos Owner",
				Team:    "platform",
				Contact: "slack:#ops",
			},
			Approval: experiment.ApprovalMetadata{
				Tier:       "tier-2",
				ApprovedBy: []string{"service-owner"},
			},
			Rollback: experiment.RollbackMetadata{
				Runbook: "docs/shared/operations/chaos-game-day-runbook.md",
				Summary: "Stop the experiment and verify alarm recovery.",
			},
			Evidence: experiment.EvidenceMetadata{
				Destination: "program-management/tasks/active/chaos/findings",
				Retention:   "release-plus-90d",
			},
		},
	}

	encoded, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var decoded experiment.Input
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if decoded.Manifest.Scenario.Class != "dependency-failure" {
		t.Fatalf("scenario class = %q, want dependency-failure", decoded.Manifest.Scenario.Class)
	}

	if decoded.Manifest.Owner.Name != "Chaos Owner" {
		t.Fatalf("owner name = %q, want Chaos Owner", decoded.Manifest.Owner.Name)
	}

	if decoded.Manifest.Rollback.Runbook == "" {
		t.Fatal("rollback runbook was not preserved")
	}

	if len(decoded.Manifest.Approval.ApprovedBy) != 1 {
		t.Fatalf("approvedBy len = %d, want 1", len(decoded.Manifest.Approval.ApprovedBy))
	}
}
