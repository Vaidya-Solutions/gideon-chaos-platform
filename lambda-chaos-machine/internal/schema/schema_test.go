// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause

package schema_test

import (
	"encoding/json"
	"testing"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/schema"
)

func TestInputSchema_Embedded(t *testing.T) {
	if len(schema.InputSchema) == 0 {
		t.Fatal("InputSchema is empty — go:embed failed")
	}
}

func TestInputSchema_ValidJSON(t *testing.T) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(schema.InputSchema, &parsed); err != nil {
		t.Fatalf("InputSchema is not valid JSON: %v", err)
	}
}

func TestInputSchema_HasRequiredFields(t *testing.T) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(schema.InputSchema, &parsed); err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Verify it's a JSON Schema with correct $id
	if id, ok := parsed["$id"].(string); !ok || id != "chaos-machine-input" {
		t.Errorf("unexpected $id: %v", parsed["$id"])
	}

	// Verify required properties
	required, ok := parsed["required"].([]interface{})
	if !ok {
		t.Fatal("required field is not an array")
	}
	reqSet := make(map[string]bool)
	for _, r := range required {
		requiredField, fieldOK := r.(string)
		if !fieldOK {
			t.Fatalf("required entry is not a string: %T", r)
		}

		reqSet[requiredField] = true
	}
	for _, field := range []string{"testId", "experimentTemplateId"} {
		if !reqSet[field] {
			t.Errorf("expected %q in required fields", field)
		}
	}

	// Verify properties exist
	props, ok := parsed["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties field is not an object")
	}
	for _, field := range []string{"testId", "experimentTemplateId", "manifest", "steadyStateMetrics", "steadyStateAlarms", "recoveryDelay", "recoveryDuration"} {
		if _, ok := props[field]; !ok {
			t.Errorf("missing property: %s", field)
		}
	}
}
