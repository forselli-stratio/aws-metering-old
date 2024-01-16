package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/forselli-stratio/aws-metering/pkg/metrics"
)

// DynamoDBTableName is the name of the DynamoDB table
const DynamoDBTableName = "StratioAWSMarketplaceMeteringRecords"

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
		Region:   aws.String("eu-west-1"),
		MaxRetries: aws.Int(3),
	})
}

// CheckIfRecordExists checks if a metering record exists for the specified time interval
func CheckIfRecordExists(record *MeteringRecord) bool {
    // Determine the start of the current hour
    now := time.Now()
    startOfCurrentHour := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location()).Unix()

    // Create a query input to check for existing records in the current hour
    queryInput := &dynamodb.QueryInput{
        TableName: aws.String(DynamoDBTableName),
        KeyConditions: map[string]*dynamodb.Condition{
            "customerIdentifier": {
                ComparisonOperator: aws.String("EQ"),
                AttributeValueList: []*dynamodb.AttributeValue{
                    {
                        S: aws.String(record.CustomerIdentifier),
                    },
                },
            },
            "create_timestamp": {
                ComparisonOperator: aws.String("GE"),
                AttributeValueList: []*dynamodb.AttributeValue{
                    {
                        N: aws.String(fmt.Sprintf("%d", startOfCurrentHour)),
                    },
                },
            },
        },
    }

    // Perform the query
    result, err := dynamoDBClient.Query(queryInput)
    if err!= nil {
		log.Printf("Error querying DynamoDB for existing records: %s", err)
		metrics.DynamoDBErrorsTotal.Inc()
		return false
	}

	// Check if a record exists in the current hour
	if *result.Count > 0 {
		log.Printf("A record for this customer already exists in the current hour.")
		return true
	}

	return false
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
			L: []*dynamodb.AttributeValue{},
		},
		"metering_pending": {
			S: aws.String(record.MeteringPending),
		},
	}

	// Add dimension usage to DynamoDB item
	for _, usage := range record.DimensionUsage {
		dimensionUsageItem := &dynamodb.AttributeValue{
			M: map[string]*dynamodb.AttributeValue{
				"dimension": {
					S: aws.String(usage.Dimension),
				},
				"value": {
					N: aws.String(fmt.Sprintf("%d", usage.Value)),
				},
			},
		}
		item["dimension_usage"].L = append(item["dimension_usage"].L, dimensionUsageItem)
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
		log.Printf("Error uploading data to DynamoDB: %s", err)
		metrics.DynamoDBErrorsTotal.Inc()
	} else {
		log.Printf("Successfully uploaded data to DynamoDB")
	}

	return err
}