// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause

package experiment_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/Vaidya-Solutions/gideon/services/lambda-chaos-machine/internal/experiment"
)

// mockDDBClient satisfies experiment.DDBClient for testing.
type mockDDBClient struct {
	putItemFn    func(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	queryFn      func(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	deleteItemFn func(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
}

func (m *mockDDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if m.putItemFn != nil {
		return m.putItemFn(ctx, params, optFns...)
	}
	return &dynamodb.PutItemOutput{}, nil
}

func (m *mockDDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, params, optFns...)
	}
	return &dynamodb.QueryOutput{}, nil
}

func (m *mockDDBClient) DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	if m.deleteItemFn != nil {
		return m.deleteItemFn(ctx, params, optFns...)
	}
	return &dynamodb.DeleteItemOutput{}, nil
}

func TestPutTaskToken_Success(t *testing.T) {
	var capturedInput *dynamodb.PutItemInput
	client := &mockDDBClient{
		putItemFn: func(_ context.Context, params *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			capturedInput = params
			return &dynamodb.PutItemOutput{}, nil
		},
	}

	item := experiment.DDBItem{
		TestID:          "test-001",
		ExperimentID:    "EXP123",
		ExperimentType:  "FIS",
		TaskToken:       "token-abc",
		ExecutionName:   "chaos-machine-steady-state",
		TestDescription: "Test chaos experiment",
	}

	err := experiment.PutTaskToken(context.Background(), client, "chaos-tests", item)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedInput == nil {
		t.Fatal("PutItem was not called")
	}
	if aws.ToString(capturedInput.TableName) != "chaos-tests" {
		t.Errorf("wrong table: %s", aws.ToString(capturedInput.TableName))
	}

	// Verify key attributes are present
	testID, ok := capturedInput.Item["testId"]
	if !ok {
		t.Fatal("testId not in item")
	}
	if s, ok := testID.(*ddbtypes.AttributeValueMemberS); !ok || s.Value != "test-001" {
		t.Errorf("wrong testId value: %v", testID)
	}
}

func TestPutTaskToken_Error(t *testing.T) {
	client := &mockDDBClient{
		putItemFn: func(_ context.Context, _ *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
			return nil, fmt.Errorf("throttled")
		},
	}

	err := experiment.PutTaskToken(context.Background(), client, "chaos-tests", experiment.DDBItem{
		TestID: "test-001",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "DDB PutItem: throttled" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestQueryTaskToken_Found(t *testing.T) {
	client := &mockDDBClient{
		queryFn: func(_ context.Context, params *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			if aws.ToString(params.IndexName) != "experimentId-index" {
				t.Errorf("wrong GSI name: %s", aws.ToString(params.IndexName))
			}
			return &dynamodb.QueryOutput{
				Items: []map[string]ddbtypes.AttributeValue{
					{
						"testId":         &ddbtypes.AttributeValueMemberS{Value: "test-001"},
						"taskToken":      &ddbtypes.AttributeValueMemberS{Value: "token-abc"},
						"experimentType": &ddbtypes.AttributeValueMemberS{Value: "FIS"},
					},
				},
			}, nil
		},
	}

	item, err := experiment.QueryTaskToken(context.Background(), client, "chaos-tests", "experimentId-index", "EXP123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item == nil {
		t.Fatal("expected item, got nil")
	}
	if item.TestID != "test-001" {
		t.Errorf("wrong testId: %s", item.TestID)
	}
	if item.TaskToken != "token-abc" {
		t.Errorf("wrong taskToken: %s", item.TaskToken)
	}
}

func TestQueryTaskToken_NotFound(t *testing.T) {
	client := &mockDDBClient{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return &dynamodb.QueryOutput{Items: nil}, nil
		},
	}

	item, err := experiment.QueryTaskToken(context.Background(), client, "chaos-tests", "experimentId-index", "EXP-NONEXIST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item != nil {
		t.Errorf("expected nil item, got %+v", item)
	}
}

func TestQueryTaskToken_Error(t *testing.T) {
	client := &mockDDBClient{
		queryFn: func(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
			return nil, fmt.Errorf("service unavailable")
		},
	}

	_, err := experiment.QueryTaskToken(context.Background(), client, "chaos-tests", "experimentId-index", "EXP123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteRecord_Success(t *testing.T) {
	var capturedInput *dynamodb.DeleteItemInput
	client := &mockDDBClient{
		deleteItemFn: func(_ context.Context, params *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			capturedInput = params
			return &dynamodb.DeleteItemOutput{}, nil
		},
	}

	err := experiment.DeleteRecord(context.Background(), client, "chaos-tests", "test-001", "EXP123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedInput == nil {
		t.Fatal("DeleteItem was not called")
	}
	if aws.ToString(capturedInput.TableName) != "chaos-tests" {
		t.Errorf("wrong table: %s", aws.ToString(capturedInput.TableName))
	}

	// Verify composite key
	testID, ok := capturedInput.Key["testId"]
	if !ok {
		t.Fatal("testId not in key")
	}
	if s, ok := testID.(*ddbtypes.AttributeValueMemberS); !ok || s.Value != "test-001" {
		t.Errorf("wrong testId: %v", testID)
	}
	expID, ok := capturedInput.Key["experimentId"]
	if !ok {
		t.Fatal("experimentId not in key")
	}
	if s, ok := expID.(*ddbtypes.AttributeValueMemberS); !ok || s.Value != "EXP123" {
		t.Errorf("wrong experimentId: %v", expID)
	}
}

func TestDeleteRecord_Error(t *testing.T) {
	client := &mockDDBClient{
		deleteItemFn: func(_ context.Context, _ *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
			return nil, fmt.Errorf("conditional check failed")
		},
	}

	err := experiment.DeleteRecord(context.Background(), client, "chaos-tests", "test-001", "EXP123")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
