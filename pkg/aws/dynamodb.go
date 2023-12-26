package aws

import (
	"fmt"
	"log"

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
	dynamoDBClient = dynamodb.New(mySession, &aws.Config{
		Endpoint: aws.String("http://localhost:8000"),
		Region:   aws.String("eu-west-1"),
		MaxRetries: aws.Int(3),
	})
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
	metrics.DynamoDBOperationsTotal.Inc()
	if err != nil {
		metrics.DynamoDBErrorsTotal.Inc()
	} else {
		log.Printf("Succesfully uploaded data do DynamoDB")
	}

	return err
}