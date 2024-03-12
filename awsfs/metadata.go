package awsfs

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type MetadataStore struct {
	EntryTableName     string
	ReferenceTableName string
	DynamoDBClient     dynamodb.Client
}

var ErrNoSuchReference = errors.New("no such reference")

func (m MetadataStore) GetReference(ctx context.Context, id string) (Reference, error) {
	resp, err := m.DynamoDBClient.GetItem(ctx, &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		TableName:      aws.String(m.ReferenceTableName),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return Reference{}, err
	}
	if resp.Item == nil {
		return Reference{}, ErrNoSuchReference
	}
	item := Reference{}
	err = attributevalue.UnmarshalMap(resp.Item, &item)
	if err != nil {
		return Reference{}, err
	}
	return item, nil
}

func (m MetadataStore) AddEntry(ctx context.Context, entry Entry, ref Reference) error {
	entryItem, err := attributevalue.MarshalMap(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	condition := expression.Name("version").Equal(expression.Value(ref.Version))
	update := expression.Set(expression.Name("entries"), expression.Value(ref.Entries)).
		Add(expression.Name("version"), expression.Value(ref.Version+1))
	expr, err := expression.NewBuilder().
		WithCondition(condition).
		WithUpdate(update).
		Build()

	if err != nil {
		return fmt.Errorf("failed to build expression, %w", err)
	}

	_, err = m.DynamoDBClient.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Put: &types.Put{
					TableName: aws.String(m.EntryTableName),
					Item:      entryItem,
				},
			},
			{
				Update: &types.Update{
					Key: map[string]types.AttributeValue{
						"id": &types.AttributeValueMemberS{
							Value: ref.ID,
						},
					},
					TableName:                           aws.String(m.ReferenceTableName),
					UpdateExpression:                    expr.Update(),
					ConditionExpression:                 expr.Condition(),
					ExpressionAttributeNames:            expr.Names(),
					ExpressionAttributeValues:           expr.Values(),
					ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureNone,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to transact write items: %w", err)
	}
	return nil
}
