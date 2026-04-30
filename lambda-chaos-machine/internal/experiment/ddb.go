// Copyright (c) Vaidya Solutions
// SPDX-License-Identifier: BSD-3-Clause

package experiment

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DDBClient defines the DynamoDB operations used by chaos-machine Lambdas.
// Extracted as an interface for unit testing.
type DDBClient interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
}

// PutTaskToken stores the experiment→taskToken correlation record in DynamoDB.
func PutTaskToken(ctx context.Context, client DDBClient, table string, item DDBItem) error {
	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return fmt.Errorf("marshal DDB item: %w", err)
	}

	_, err = client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(table),
		Item:      av,
	})
	if err != nil {
		return fmt.Errorf("DDB PutItem: %w", err)
	}

	return nil
}

// QueryTaskToken looks up the task token by experimentId via the GSI.
func QueryTaskToken(ctx context.Context, client DDBClient, table, gsiName, experimentID string) (*DDBItem, error) {
	out, err := client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(table),
		IndexName:              aws.String(gsiName),
		KeyConditionExpression: aws.String("experimentId = :eid"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":eid": &ddbtypes.AttributeValueMemberS{Value: experimentID},
		},
		ProjectionExpression: aws.String("taskToken, experimentType, testId"),
		Limit:                aws.Int32(1),
	})
	if err != nil {
		return nil, fmt.Errorf("DDB GSI query: %w", err)
	}

	if len(out.Items) == 0 {
		return nil, nil
	}

	var item DDBItem
	if unmarshalErr := attributevalue.UnmarshalMap(out.Items[0], &item); unmarshalErr != nil {
		return nil, fmt.Errorf("unmarshal DDB item: %w", unmarshalErr)
	}

	return &item, nil
}

// DeleteRecord removes the correlation record after hypothesis evaluation.
func DeleteRecord(ctx context.Context, client DDBClient, table, testID, experimentID string) error {
	_, err := client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(table),
		Key: map[string]ddbtypes.AttributeValue{
			"testId":       &ddbtypes.AttributeValueMemberS{Value: testID},
			"experimentId": &ddbtypes.AttributeValueMemberS{Value: experimentID},
		},
	})
	if err != nil {
		return fmt.Errorf("DDB DeleteItem: %w", err)
	}

	return nil
}
