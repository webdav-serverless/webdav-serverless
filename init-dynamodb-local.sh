#!/bin/bash
set -u

aws dynamodb create-table \
    --table-name Metadata \
    --region us-east-1 \
    --endpoint-url $DYNAMO_DB_URL \
    --attribute-definitions \
        AttributeName=ID,AttributeType=S \
    --key-schema \
        AttributeName=ID,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST
aws dynamodb create-table \
    --table-name Reference \
    --region us-east-1 \
    --endpoint-url $DYNAMO_DB_URL \
    --attribute-definitions \
        AttributeName=ID,AttributeType=S \
    --key-schema \
        AttributeName=ID,KeyType=HASH \
    --billing-mode PAY_PER_REQUEST
