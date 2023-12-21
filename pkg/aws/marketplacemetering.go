package aws

import (
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/marketplacemetering"
	"github.com/forselli-stratio/aws-metering/pkg/metrics"
)

func CreateBatchMeterUsageInput(productCode string, customerIdentifier string, records ...*marketplacemetering.UsageRecord) *marketplacemetering.BatchMeterUsageInput {
	timezone, _ := time.LoadLocation("UTC")
	meteringRecords := &marketplacemetering.BatchMeterUsageInput{
		ProductCode: aws.String(productCode),
		UsageRecords: records,
	}

    // Set common fields for all records
    for _, record := range records {
        record.CustomerIdentifier = aws.String(customerIdentifier)
        record.Timestamp = aws.Time(record.Timestamp.In(timezone))
    }

	return meteringRecords
}

func SendBatchMeterUsageRequest(m *marketplacemetering.BatchMeterUsageInput) (*marketplacemetering.BatchMeterUsageOutput, error) {
	// Create a new session with default credentials
	// Initial credentials loaded from SDK's default credential chain. Such as
	// the environment, shared credentials (~/.aws/credentials), or EC2 Instance
	// Role. These credentials will be used to to make the STS Assume Role API.
	mySession := session.Must(session.NewSession())

	// Create a Marketplace Metering service client
	svc := marketplacemetering.New(mySession, aws.NewConfig().WithRegion("ea-west-1"))

	// Create BatchMeterUsage request
	req, resp := svc.BatchMeterUsageRequest(m)

	// Send the request
	err := req.Send()

	// Update metrics with request status code
	metrics.RequestsTotal.WithLabelValues(strconv.Itoa(req.HTTPResponse.StatusCode)).Inc()

	// Handle errors
	if err != nil {
		return nil, fmt.Errorf("%v", err)
	}

	// Log successful request
	fmt.Println("New metering record sent:")
	fmt.Println(req)

	return resp, nil
}