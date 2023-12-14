package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/marketplacemetering"
)

func CreateMeteringRecords(productCode string, customerIdentifier string, cpuValue, memValue, storageValue int64, cpuTimestamp, memTimestamp, storageTimestamp time.Time) (marketplacemetering.BatchMeterUsageInput) {
	timezone, _ := time.LoadLocation("UTC")
	meteringRecords := &marketplacemetering.BatchMeterUsageInput{
		ProductCode: aws.String(productCode),
		UsageRecords: []*marketplacemetering.UsageRecord{
			{
				CustomerIdentifier: aws.String(customerIdentifier),
				Dimension:          aws.String("cpu"),
				Quantity:           aws.Int64(cpuValue),
				Timestamp:          aws.Time(cpuTimestamp.In(timezone)),
			},
			{
				CustomerIdentifier: aws.String(customerIdentifier),
				Dimension:          aws.String("memory"),
				Quantity:           aws.Int64(memValue),
				Timestamp:          aws.Time(memTimestamp.In(timezone)),
			},
			{
				CustomerIdentifier: aws.String(customerIdentifier),
				Dimension:          aws.String("storage"),
				Quantity:           aws.Int64(storageValue),
				Timestamp:          aws.Time(storageTimestamp.In(timezone)),
			},
		},
	}

	return *meteringRecords
}
