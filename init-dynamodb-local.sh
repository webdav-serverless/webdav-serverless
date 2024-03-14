#!/bin/bash
set -u

aws dynamodb create-table \
    --table-name Entry \
    --region us-east-1 \
    --endpoint-url $DYNAMO_DB_URL \
    --attribute-definitions \
        AttributeName=id,AttributeType=S \
        AttributeName=parent_id,AttributeType=S \
    --key-schema \
        AttributeName=id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST \
    --global-secondary-indexes \
        "[
            {
                \"IndexName\": \"entry-index-parent_id\",
                \"KeySchema\": [{\"AttributeName\":\"parent_id\",\"KeyType\":\"HASH\"}],
                \"Projection\":{
                    \"ProjectionType\":\"ALL\"
                }
            }
        ]"
aws dynamodb create-table \
    --table-name Reference \
    --region us-east-1 \
    --endpoint-url $DYNAMO_DB_URL \
    --attribute-definitions \
        AttributeName=id,AttributeType=S \
    --key-schema \
        AttributeName=id,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST
