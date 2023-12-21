package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/forselli-stratio/aws-metering/pkg/metrics"
)

// DynamoDBTableName is the name of the DynamoDB table
const DynamoDBTableName = "AWSMarketplaceMeteringRecords"

// MeteringRecord represents a metering record to be stored in DynamoDB.
type MeteringRecord struct {
	CreateTimestamp   int64
	CustomerIdentifier string
	DimensionUsage     []struct {
		Dimension string
		Value     int64
	}
	MeteringPending string
}

var dynamoDBClient *dynamodb.DynamoDB

// InitDynamoDB initializes the DynamoDB client
func InitDynamoDB() {
	// Create a new session with default credentials
	// Initial credentials loaded from SDK's default credential chain.
	mySession := session.Must(session.NewSession())

	// Create a DynamoDB service client
	dynamoDBClient = dynamodb.New(mySession, aws.NewConfig().WithEndpoint("http://localhost:8000"), aws.NewConfig().WithRegion("eu-west-1"))
}

// InsertMeteringRecord inserts a metering record into DynamoDB
func InsertMeteringRecord(record *MeteringRecord) error {
	// Create DynamoDB item
	item := map[string]*dynamodb.AttributeValue{
		"create_timestamp": {
			N: aws.String(fmt.Sprintf("%d", record.CreateTimestamp)),
		},
		"customerIdentifier": {
			S: aws.String(record.CustomerIdentifier),
		},
		"dimension_usage": {
			M: map[string]*dynamodb.AttributeValue{},
		},
		"metering_pending": {
			S: aws.String(record.MeteringPending),
		},
	}

	// Add dimension usage to DynamoDB item
	for _, usage := range record.DimensionUsage {
		item["dimension_usage"].M[usage.Dimension] = &dynamodb.AttributeValue{
			N: aws.String(fmt.Sprintf("%d", usage.Value)),
		}
	}

	// Create DynamoDB input
	input := &dynamodb.PutItemInput{
		Item:      item,
		TableName: aws.String(DynamoDBTableName),
	}

	// Perform PutItem operation
	_, err := dynamoDBClient.PutItem(input)

	// Update metrics with DynamoDB operation status
	if err != nil {
		metrics.DynamoDBErrorsTotal.Inc()
	} else {
		fmt.Println("Succesfully uploaded data do DynamoDB")
		metrics.DynamoDBPutItemTotal.Inc()
	}

	return err
}