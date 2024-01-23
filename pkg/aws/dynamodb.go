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
func CheckIfRecordExists(customerIdentifier string, startTime, endTime int64) bool {
    // Create a query input to check for existing records within the current hour interval
    queryInput := &dynamodb.QueryInput{
        TableName: aws.String(DynamoDBTableName),
        KeyConditions: map[string]*dynamodb.Condition{
            "customerIdentifier": {
                ComparisonOperator: aws.String("EQ"),
                AttributeValueList: []*dynamodb.AttributeValue{
                    {
                        S: aws.String(customerIdentifier),
                    },
                },
            },
            "create_timestamp": {
                ComparisonOperator: aws.String("BETWEEN"),
                AttributeValueList: []*dynamodb.AttributeValue{
                    {
                        N: aws.String(fmt.Sprintf("%d", startTime)),
                    },
                    {
                        N: aws.String(fmt.Sprintf("%d", endTime)),
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

    // Check if any items were returned
    exists := len(result.Items) > 0
    if exists {
        humanReadableStartTime := time.Unix(startTime, 0).Format("2006-01-02 15:04:05")
        humanReadableEndTime := time.Unix(endTime, 0).Format("2006-01-02 15:04:05")
        log.Printf("Record already exists for customer '%s' between %s and %s", customerIdentifier, humanReadableStartTime, humanReadableEndTime)
    }
    return exists
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

    // Check if dimension_usage is empty or has less than 3 items
    if len(item["dimension_usage"].L) < 3 {
        log.Printf("Skipping metering record upload to DynamoDB as dimension_usage is empty or incomplete.")
        return nil
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
	return err
}