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
	"github.com/google/uuid"
	"time"
)

type MetadataStore struct {
	EntryTableName     string
	ReferenceTableName string
	DynamoDBClient     *dynamodb.Client
}

var (
	ErrNoSuchReference = errors.New("no such reference")
	ErrNoSuchEntry     = errors.New("no such entry")
)

func (m MetadataStore) Init(ctx context.Context) error {
	_, err := m.GetReference(ctx, referenceID)
	if errors.Is(err, ErrNoSuchReference) {
		entryID := uuid.New().String()
		entry := Entry{
			ID:       entryID,
			ParentID: "root",
			Name:     "/",
			Type:     EntryTypeDir,
			Size:     0,
			Modify:   time.Now(),
			Version:  1,
		}
		ref := Reference{
			ID: referenceID,
			Entries: map[string]string{
				"/": entryID,
			},
			Version: 1,
		}

		entryItem, err := attributevalue.MarshalMap(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal entry: %w", err)
		}
		refItem, err := attributevalue.MarshalMap(ref)
		if err != nil {
			return fmt.Errorf("failed to marshal refarence: %w", err)
		}

		_, putErr := m.DynamoDBClient.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
			TransactItems: []types.TransactWriteItem{
				{
					Put: &types.Put{
						TableName: aws.String(m.EntryTableName),
						Item:      entryItem,
					},
				},
				{
					Put: &types.Put{
						TableName: aws.String(m.ReferenceTableName),
						Item:      refItem,
					},
				},
			},
		})

		if putErr != nil {
			return fmt.Errorf("failed to init: %s", putErr)
		}
	}
	if err != nil {
		return fmt.Errorf("failed to get reference: %w", err)
	}
	return nil
}

func (m MetadataStore) AddReference(ctx context.Context, ref Reference) error {
	refItem, err := attributevalue.MarshalMap(ref)
	if err != nil {
		return fmt.Errorf("failed to marshal reference: %w", err)
	}
	_, err = m.DynamoDBClient.PutItem(ctx, &dynamodb.PutItemInput{
		Item:      refItem,
		TableName: aws.String(m.ReferenceTableName),
	})
	if err != nil {
		return err
	}
	return nil
}

func (m MetadataStore) GetReference(ctx context.Context, id string) (Reference, error) {
	resp, err := m.DynamoDBClient.GetItem(ctx, &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		TableName:      aws.String(m.ReferenceTableName),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return Reference{}, fmt.Errorf("failed to get item: %w", err)
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

func (m MetadataStore) GetEntry(ctx context.Context, id string) (Entry, error) {
	out, err := m.DynamoDBClient.GetItem(ctx, &dynamodb.GetItemInput{
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		TableName:      aws.String(m.EntryTableName),
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return Entry{}, fmt.Errorf("failed to get item: %w", err)
	}
	if out.Item == nil {
		return Entry{}, ErrNoSuchEntry
	}
	entry := Entry{}
	err = attributevalue.UnmarshalMap(out.Item, &entry)
	if err != nil {
		return Entry{}, fmt.Errorf("failed to unmarshal map: %w", err)
	}
	return entry, nil
}

func (m MetadataStore) GetEntriesByParentID(ctx context.Context, id string) ([]Entry, error) {

	builder := expression.NewBuilder().
		WithKeyCondition(expression.KeyEqual(expression.Key("parent_id"), expression.Value(id)))
	expr, err := builder.Build()
	if err != nil {
		return nil, err
	}

	out, err := m.DynamoDBClient.Query(ctx, &dynamodb.QueryInput{
		IndexName:                 aws.String("entry-index-parent_id"),
		TableName:                 aws.String(m.EntryTableName),
		ExpressionAttributeNames:  expr.Names(),
		KeyConditionExpression:    expr.KeyCondition(),
		ScanIndexForward:          aws.Bool(true),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query entry: %w", err)
	}
	var entries []Entry
	err = attributevalue.UnmarshalListOfMaps(out.Items, &entries)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal map: %w", err)
	}
	return entries, nil
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

func (m MetadataStore) UpdateEntryName(ctx context.Context, entry Entry, ref Reference) error {
	entryCondition := expression.Name("version").Equal(expression.Value(entry.Version))
	entryUpdate := expression.Set(expression.Name("name"), expression.Value(entry.Name)).
		Add(expression.Name("version"), expression.Value(ref.Version+1))
	entryExpr, err := expression.NewBuilder().
		WithCondition(entryCondition).
		WithUpdate(entryUpdate).
		Build()
	if err != nil {
		return fmt.Errorf("failed to build expression, %w", err)
	}

	refCondition := expression.Name("version").Equal(expression.Value(ref.Version))
	refUpdate := expression.Set(expression.Name("entries"), expression.Value(ref.Entries)).
		Add(expression.Name("version"), expression.Value(ref.Version+1))
	refExpr, err := expression.NewBuilder().
		WithCondition(refCondition).
		WithUpdate(refUpdate).
		Build()
	if err != nil {
		return fmt.Errorf("failed to build expression, %w", err)
	}

	_, err = m.DynamoDBClient.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: []types.TransactWriteItem{
			{
				Update: &types.Update{
					Key: map[string]types.AttributeValue{
						"id": &types.AttributeValueMemberS{
							Value: entry.ID,
						},
					},
					TableName:                           aws.String(m.EntryTableName),
					UpdateExpression:                    entryExpr.Update(),
					ConditionExpression:                 entryExpr.Condition(),
					ExpressionAttributeNames:            entryExpr.Names(),
					ExpressionAttributeValues:           entryExpr.Values(),
					ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureNone,
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
					UpdateExpression:                    refExpr.Update(),
					ConditionExpression:                 refExpr.Condition(),
					ExpressionAttributeNames:            refExpr.Names(),
					ExpressionAttributeValues:           refExpr.Values(),
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

func (m MetadataStore) DeleteEntries(ctx context.Context, ids []string, ref Reference) error {
	refCondition := expression.Name("version").Equal(expression.Value(ref.Version))
	refUpdate := expression.Set(expression.Name("entries"), expression.Value(ref.Entries)).
		Add(expression.Name("version"), expression.Value(ref.Version+1))
	refExpr, err := expression.NewBuilder().
		WithCondition(refCondition).
		WithUpdate(refUpdate).
		Build()
	if err != nil {
		return fmt.Errorf("failed to build expression, %w", err)
	}

	var deleteItems []types.TransactWriteItem
	for _, id := range ids {
		deleteItems = append(deleteItems, types.TransactWriteItem{
			Delete: &types.Delete{
				TableName: aws.String(m.EntryTableName),
				Key: map[string]types.AttributeValue{
					"ID": &types.AttributeValueMemberS{
						Value: id,
					},
				},
			},
		})
	}
	deleteItems = append(deleteItems, types.TransactWriteItem{
		Update: &types.Update{
			Key: map[string]types.AttributeValue{
				"id": &types.AttributeValueMemberS{
					Value: ref.ID,
				},
			},
			TableName:                           aws.String(m.ReferenceTableName),
			UpdateExpression:                    refExpr.Update(),
			ConditionExpression:                 refExpr.Condition(),
			ExpressionAttributeNames:            refExpr.Names(),
			ExpressionAttributeValues:           refExpr.Values(),
			ReturnValuesOnConditionCheckFailure: types.ReturnValuesOnConditionCheckFailureNone,
		},
	})
	_, err = m.DynamoDBClient.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: deleteItems,
	})
	if err != nil {
		return fmt.Errorf("failed to delete items, %w", err)
	}

	return nil
}
